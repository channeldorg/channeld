package channeld

import (
	"container/list"
	"errors"

	"channeld.clewcat.com/channeld/proto"
	"go.uber.org/zap"
)

type ChannelSubscription struct {
	options proto.ChannelSubscriptionOptions
	//fanOutDataMsg  Message
	//lastFanOutTime time.Time
	fanOutElement *list.Element
}

func (c *Connection) SubscribeToChannel(ch *Channel, options *proto.ChannelSubscriptionOptions) {
	if ch.subscribedConnections[c] != nil {
		c.Logger().Info("already subscribed", zap.String("channel", ch.String()))
		return
	}

	cs := &ChannelSubscription{
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
			FanOutIntervalMs: GlobalSettings.GetChannelSettings(ch.channelType).DefaultFanOutIntervalMs,
		}
	}
	cs.fanOutElement = ch.fanOutQueue.PushFront(&fanOutConnection{conn: c})
	// Records the maximum fan-out interval for checking if the oldest update message is removable when the buffer is overflowed.
	if ch.data != nil && ch.data.maxFanOutIntervalMs < cs.options.FanOutIntervalMs {
		ch.data.maxFanOutIntervalMs = cs.options.FanOutIntervalMs
	}
	ch.subscribedConnections[c] = cs
}

func (c *Connection) UnsubscribeFromChannel(ch *Channel) error {
	cs, exists := ch.subscribedConnections[c]
	if !exists {
		return errors.New("subscription does not exist")
	} else {
		ch.fanOutQueue.Remove(cs.fanOutElement)
		delete(ch.subscribedConnections, c)
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

func (c *Connection) sendSubscribed(ctx MessageContext, ch *Channel, connToSub *Connection, stubId uint32, subOptions *proto.ChannelSubscriptionOptions) {
	ctx.Channel = ch
	ctx.StubId = stubId
	ctx.MsgType = proto.MessageType_SUB_TO_CHANNEL
	ctx.Msg = &proto.SubscribedToChannelResultMessage{
		ConnId:      uint32(connToSub.id),
		SubOptions:  subOptions,
		ConnType:    connToSub.connectionType,
		ChannelType: ch.channelType,
	}
	// ctx.Msg = &proto.SubscribedToChannelMessage{
	// 	ConnId:     uint32(ctx.Connection.id),
	// 	SubOptions: &ch.subscribedConnections[c.id].options,
	// }
	c.Send(ctx)
}

func (c *Connection) sendUnsubscribed(ctx MessageContext, ch *Channel, connToUnsub *Connection, stubId uint32) {
	ctx.Channel = ch
	ctx.StubId = stubId
	ctx.MsgType = proto.MessageType_UNSUB_FROM_CHANNEL
	ctx.Msg = &proto.UnsubscribedFromChannelResultMessage{
		ConnId:      uint32(connToUnsub.id),
		ConnType:    connToUnsub.connectionType,
		ChannelType: ch.channelType,
	}
	c.Send(ctx)
}

func (ch *Channel) AddConnectionSubscribedNotification(connId ConnectionId) {

}
