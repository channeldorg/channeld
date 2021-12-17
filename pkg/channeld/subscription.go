package channeld

import (
	"container/list"
	"errors"

	"channeld.clewcat.com/channeld/proto"
	"go.uber.org/zap"
	protobuf "google.golang.org/protobuf/proto"
)

const (
	DefaultFanOutIntervalMs uint32 = 20
)

type ChannelSubscription struct {
	options proto.ChannelSubscriptionOptions
	//fanOutDataMsg  Message
	//lastFanOutTime time.Time
	fanOutElement *list.Element
}

func (c *Connection) SubscribeToChannel(ch *Channel, options *proto.ChannelSubscriptionOptions) error {
	cs, exists := ch.subscribedConnections[c.id]
	if exists {
		c.Logger().Info("already subscribed to channel, the subscription options will be merged",
			zap.String("channelType", ch.channelType.String()),
			zap.Uint32("channelId", uint32(ch.id)),
		)
		if options != nil {
			protobuf.Merge(&cs.options, options)
		}
	} else {

		cs = &ChannelSubscription{
			// Send the whole data to the connection when subscribed
			//fanOutDataMsg: ch.Data().msg,
		}
		if options != nil {
			cs.options = proto.ChannelSubscriptionOptions{
				CanUpdateData:    options.CanUpdateData,
				DataFieldMasks:   options.DataFieldMasks,
				FanOutIntervalMs: options.FanOutIntervalMs,
			}
		} else {
			cs.options = proto.ChannelSubscriptionOptions{
				CanUpdateData:    true,
				DataFieldMasks:   make([]string, 0),
				FanOutIntervalMs: DefaultFanOutIntervalMs,
			}
		}
		cs.fanOutElement = ch.fanOutQueue.PushFront(&fanOutConnection{connId: c.id})
		// Records the maximum fan-out interval for checking if the oldest update message is removable when the buffer is overflowed.
		if ch.data != nil && ch.data.maxFanOutIntervalMs < cs.options.FanOutIntervalMs {
			ch.data.maxFanOutIntervalMs = cs.options.FanOutIntervalMs
		}
		ch.subscribedConnections[c.id] = cs
	}
	return nil
}

func (c *Connection) UnsubscribeFromChannel(ch *Channel) error {
	cs, exists := ch.subscribedConnections[c.id]
	if !exists {
		return errors.New("subscription does not exist")
	} else {
		ch.fanOutQueue.Remove(cs.fanOutElement)
		delete(ch.subscribedConnections, c.id)
	}
	return nil
}

/*
func (c *Connection) sendConnSubscribed(connId ConnectionId, ids ...ChannelId) {
	channelIds := make([]uint32, len(ids))
	for i, id := range ids {
		channelIds[i] = uint32(id)
	}
	subMsg := &proto.SubscribedToChannelsMessage{ConnId: uint32(connId), ChannelIds: channelIds}
	c.SendWithGlobalChannel(proto.MessageType_SUB_TO_CHANNEL, subMsg)
}

func (c *Connection) sendConnUnsubscribed(connId ConnectionId, ids ...ChannelId) {
	channelIds := make([]uint32, len(ids))
	for i, id := range ids {
		channelIds[i] = uint32(id)
	}
	subMsg := &proto.UnsubscribedToChannelsMessage{ConnId: uint32(connId), ChannelIds: channelIds}
	c.SendWithGlobalChannel(proto.MessageType_UNSUB_FROM_CHANNEL, subMsg)
}
*/

func (c *Connection) sendSubscribed(ctx MessageContext, ch *Channel, stubId uint32) {
	ctx.Channel = ch
	ctx.StubId = stubId
	ctx.MsgType = proto.MessageType_SUB_TO_CHANNEL
	ctx.Msg = &proto.SubscribedToChannelMessage{
		ConnId:     uint32(ctx.Connection.id),
		SubOptions: &ch.subscribedConnections[c.id].options,
	}
	c.Send(ctx)
}
func (c *Connection) sendUnsubscribed(ctx MessageContext, ch *Channel, stubId uint32) {
	ctx.Channel = ch
	ctx.StubId = stubId
	ctx.MsgType = proto.MessageType_UNSUB_FROM_CHANNEL
	ctx.Msg = &proto.UnsubscribedFromChannelMessage{
		ConnId: uint32(c.id),
	}
	c.Send(ctx)
}

func (ch *Channel) AddConnectionSubscribedNotification(connId ConnectionId) {

}
