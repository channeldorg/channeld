package channeld

import (
	"container/list"
	"fmt"
	"log"

	"clewcat.com/channeld/proto"
	protobuf "google.golang.org/protobuf/proto"
)

const (
	DefaultFanOutIntervalMs uint32 = 20
)

/* TODO: delete this code block - the definition has been moved to protobuf
type ChannelSubscriptionOptions struct {
	CanUpdateData  bool
	DataFieldMasks []string
	FanOutInterval time.Duration
}
*/

type ChannelSubscription struct {
	options proto.ChannelSubscriptionOptions
	//fanOutDataMsg  Message
	//lastFanOutTime time.Time
	fanOutElement *list.Element
}

func (c *Connection) SubscribeToChannel(ch *Channel, options *proto.ChannelSubscriptionOptions) error {
	cs, exists := ch.subscribedConnections[c.id]
	if exists {
		log.Printf("%s already subscribed to %s, the subscription options will be merged.\n", c, ch)
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

func (c *Connection) UnsubscribeToChannel(ch *Channel) error {
	cs, exists := ch.subscribedConnections[c.id]
	if !exists {
		return fmt.Errorf("%s has not subscribed to %s yet", c, ch)
	} else {
		ch.fanOutQueue.Remove(cs.fanOutElement)
		delete(ch.subscribedConnections, c.id)
	}
	return nil
}

func (c *Connection) sendConnSubscribed(connId ConnectionId, ids ...ChannelId) {
	channelIds := make([]uint32, len(ids))
	for i, id := range ids {
		channelIds[i] = uint32(id)
	}
	subMsg := &proto.SubscribedToChannelsMessage{ConnId: uint32(connId), ChannelIds: channelIds}
	c.SendWithGlobalChannel(proto.MessageType_SUB_TO_CHANNEL, subMsg)
}

func (c *Connection) sendSubscribed(ids ...ChannelId) {
	c.sendConnSubscribed(c.id, ids...)
}

func (c *Connection) sendConnUnsubscribed(connId ConnectionId, ids ...ChannelId) {
	channelIds := make([]uint32, len(ids))
	for i, id := range ids {
		channelIds[i] = uint32(id)
	}
	subMsg := &proto.UnsubscribedToChannelsMessage{ConnId: uint32(connId), ChannelIds: channelIds}
	c.SendWithGlobalChannel(proto.MessageType_UNSUB_TO_CHANNEL, subMsg)
}

func (c *Connection) sendUnsubscribed(ids ...ChannelId) {
	c.sendConnUnsubscribed(c.id, ids...)
}

func (ch *Channel) AddConnectionSubscribedNotification(connId ConnectionId) {

}
