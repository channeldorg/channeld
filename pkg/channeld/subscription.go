package channeld

import (
	"container/list"
	"errors"

	"github.com/metaworking/channeld/pkg/channeldpb"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type ChannelSubscription struct {
	options channeldpb.ChannelSubscriptionOptions
	//fanOutDataMsg  Message
	//lastFanOutTime time.Time
	subTime       ChannelTime
	fanOutElement *list.Element
}

func defaultSubOptions(t channeldpb.ChannelType) *channeldpb.ChannelSubscriptionOptions {
	options := &channeldpb.ChannelSubscriptionOptions{
		DataAccess:           Pointer(channeldpb.ChannelDataAccess_WRITE_ACCESS),
		DataFieldMasks:       make([]string, 0),
		FanOutDelayMs:        proto.Int32(GlobalSettings.GetChannelSettings(t).DefaultFanOutDelayMs),
		FanOutIntervalMs:     proto.Uint32(GlobalSettings.GetChannelSettings(t).DefaultFanOutIntervalMs),
		SkipSelfUpdateFanOut: proto.Bool(false),
		SkipFirstFanOut:      proto.Bool(false),
	}
	return options
}

func (c *Connection) SubscribeToChannel(ch *Channel, options *channeldpb.ChannelSubscriptionOptions) *ChannelSubscription {
	defer func() {
		ch.connectionsLock.Unlock()
	}()
	ch.connectionsLock.Lock()

	if ch.subscribedConnections[c] != nil {
		c.Logger().Info("already subscribed", zap.String("channel", ch.String()))
		return nil
	}

	cs := &ChannelSubscription{
		options: *defaultSubOptions(ch.channelType),
		// Send the whole data to the connection when subscribed
		//fanOutDataMsg: ch.Data().msg,
		subTime: ch.GetTime(),
	}

	if options != nil {
		proto.Merge(&cs.options, options)
	}

	cs.fanOutElement = ch.fanOutQueue.PushFront(&fanOutConnection{
		conn:           c,
		hadFirstFanOut: *cs.options.SkipFirstFanOut,
		// Delay the first fanout, to solve the spawn & update order issue in Mirror & UE.
		lastFanOutTime: ch.GetTime().OffsetMs(*cs.options.FanOutDelayMs),
	})
	// rootLogger.Info("conn sub to channel",
	// 	zap.Uint32("connId", uint32(cs.fanOutElement.Value.(*fanOutConnection).conn.Id())),
	// 	zap.Int32("fanOutDelayMs", cs.options.FanOutDelayMs),
	// 	zap.Int64("channelTime", int64(ch.GetTime())),
	// 	zap.Int64("nextFanOutTime", int64(cs.fanOutElement.Value.(*fanOutConnection).lastFanOutTime)),
	// )

	// Records the maximum fan-out interval for checking if the oldest update message is removable when the buffer is overflowed.
	if ch.data != nil && ch.data.maxFanOutIntervalMs < *cs.options.FanOutIntervalMs {
		ch.data.maxFanOutIntervalMs = *cs.options.FanOutIntervalMs
	}
	ch.subscribedConnections[c] = cs

	if ch.channelType == channeldpb.ChannelType_SPATIAL {
		c.spatialSubscriptions.Store(ch.id, &cs.options)
	}

	ch.Logger().Debug("subscribed connection",
		zap.Uint32("connId", uint32(c.Id())),
		zap.String("dataAccess", channeldpb.ChannelDataAccess_name[int32(*cs.options.DataAccess)]),
		zap.Uint32("fanOutIntervalMs", *cs.options.FanOutIntervalMs),
		zap.Int32("fanOutDelayMs", *cs.options.FanOutDelayMs),
		zap.Bool("skipSelfUpdateFanOut", *cs.options.SkipSelfUpdateFanOut),
	)
	return cs
}

func (c *Connection) UnsubscribeFromChannel(ch *Channel) (*channeldpb.ChannelSubscriptionOptions, error) {
	defer func() {
		ch.connectionsLock.Unlock()
	}()
	ch.connectionsLock.Lock()

	cs, exists := ch.subscribedConnections[c]
	if !exists {
		return nil, errors.New("subscription does not exist")
	}

	ch.fanOutQueue.Remove(cs.fanOutElement)

	delete(ch.subscribedConnections, c)

	if ch.channelType == channeldpb.ChannelType_SPATIAL {
		c.spatialSubscriptions.Delete(ch.id)
	}

	ch.Logger().Debug("unsubscribed connection", zap.Uint32("connId", uint32(c.Id())))
	return &cs.options, nil
}

/*
func (c *Connection) sendConnSubscribed(connId ConnectionId, ids ...ChannelId) {
	channelIds := make([]uint32, len(ids))
	for i, id := range ids {
		channelIds[i] = uint32(id)
	}
	subMsg := &channeldpb.SubscribedToChannelsMessage{ConnId: uint32(connId), ChannelIds: channelIds}
	c.SendWithGlobalChannel(channeldpb.MessageType_SUB_TO_CHANNEL, subMsg)
}

func (c *Connection) sendConnUnsubscribed(connId ConnectionId, ids ...ChannelId) {
	channelIds := make([]uint32, len(ids))
	for i, id := range ids {
		channelIds[i] = uint32(id)
	}
	subMsg := &channeldpb.UnsubscribedToChannelsMessage{ConnId: uint32(connId), ChannelIds: channelIds}
	c.SendWithGlobalChannel(channeldpb.MessageType_UNSUB_FROM_CHANNEL, subMsg)
}
*/

func (c *Connection) sendSubscribed(ctx MessageContext, ch *Channel, connToSub ConnectionInChannel, stubId uint32, subOptions *channeldpb.ChannelSubscriptionOptions) {
	ctx.ChannelId = uint32(ch.id)
	ctx.StubId = stubId
	ctx.MsgType = channeldpb.MessageType_SUB_TO_CHANNEL
	ctx.Msg = &channeldpb.SubscribedToChannelResultMessage{
		ConnId:      uint32(connToSub.Id()),
		SubOptions:  subOptions,
		ConnType:    connToSub.GetConnectionType(),
		ChannelType: ch.channelType,
	}
	// ctx.Msg = &channeldpb.SubscribedToChannelMessage{
	// 	ConnId:     uint32(ctx.Connection.id),
	// 	SubOptions: &ch.subscribedConnections[c.id].options,
	// }
	c.Send(ctx)

	c.Logger().Debug("sent SUB_TO_CHANNEL", zap.Uint32("channelId", ctx.ChannelId), zap.Uint32("connToSub", uint32(connToSub.Id())))
}

func (c *Connection) sendUnsubscribed(ctx MessageContext, ch *Channel, connToUnsub *Connection, stubId uint32) {
	ctx.ChannelId = uint32(ch.id)
	ctx.StubId = stubId
	ctx.MsgType = channeldpb.MessageType_UNSUB_FROM_CHANNEL
	ctx.Msg = &channeldpb.UnsubscribedFromChannelResultMessage{
		ConnId:      uint32(connToUnsub.id),
		ConnType:    connToUnsub.connectionType,
		ChannelType: ch.channelType,
	}
	c.Send(ctx)
}

func (ch *Channel) AddConnectionSubscribedNotification(connId ConnectionId) {

}
