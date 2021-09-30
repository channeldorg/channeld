package channeld

import (
	"container/list"
	"fmt"
	"log"
	"time"

	"clewcat.com/channeld/proto"
)

const (
	defaultFanOutInterval time.Duration = time.Millisecond * 50
)

type ChannelSubscriptionOptions struct {
	CanUpdateData  bool
	DataFieldMasks []string
	FanOutInterval time.Duration
}

type ChannelSubscription struct {
	options ChannelSubscriptionOptions
	//fanOutDataMsg  Message
	//lastFanOutTime time.Time
	fanOutElement *list.Element
}

func (c *Connection) SubscribeToChannel(ch *Channel, options ChannelSubscriptionOptions) error {
	cs, exists := ch.subscribedConnections[c.id]
	if exists {
		log.Printf("%s already subscribed to %s, the subsctiption options will be update.\n", c, ch)
		cs.options = options
	} else {
		cs = &ChannelSubscription{
			options: options,
			// Send the whole data to the connection when subscribed
			//fanOutDataMsg: ch.Data().msg,
		}
		cs.fanOutElement = ch.fanOutQueue.PushFront(&FanOutConnection{connId: c.id})
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
	c.Send(proto.MessageType_SUB_TO_CHANNEL, subMsg)
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
	c.Send(proto.MessageType_UNSUB_TO_CHANNEL, subMsg)
}

func (c *Connection) sendUnsubscribed(ids ...ChannelId) {
	c.sendConnUnsubscribed(c.id, ids...)
}

func (ch *Channel) AddConnectionSubscribedNotification(connId ConnectionId) {

}
