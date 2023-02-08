package channeld

import (
	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"channeld.clewcat.com/channeld/pkg/common"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type SpatialDampingSettings struct {
	MaxDistance      uint
	FanOutIntervalMs uint32
	DataFieldMasks   []string
}

var spatialDampingSettings []*SpatialDampingSettings = []*SpatialDampingSettings{
	{
		MaxDistance:      0,
		FanOutIntervalMs: 20,
	},
	{
		MaxDistance:      1,
		FanOutIntervalMs: 50,
	},
	{
		MaxDistance:      2,
		FanOutIntervalMs: 100,
	},
}

func getSpatialDampingSettings(dist uint) *SpatialDampingSettings {
	for _, s := range spatialDampingSettings {
		if dist <= s.MaxDistance {
			return s
		}
	}
	return nil
}

func handleUpdateSpatialInterest(ctx MessageContext) {
	msg, ok := ctx.Msg.(*channeldpb.UpdateSpatialInterestMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a UpdateSpatialInterestMessage, will not be handled.")
		return
	}

	if spatialController == nil {
		ctx.Connection.Logger().Error("cannot update spatial interest as the spatial controller does not exist")
		return
	}

	conn, ok := ctx.Connection.(*Connection)
	if !ok {
		ctx.Connection.Logger().Error("ctx.Connection is not a Connection, will not be handled.")
		return
	}

	spatialChIds, err := spatialController.QueryChannelIds(msg.Query)
	if err != nil {
		ctx.Connection.Logger().Error("error getting spatial channel ids", zap.Error(err))
		return
	}

	channelsToSub := make(map[common.ChannelId]*channeldpb.ChannelSubscriptionOptions)
	for chId, dist := range spatialChIds {
		dampSettings := getSpatialDampingSettings(dist)
		if dampSettings == nil {
			channelsToSub[chId] = &channeldpb.ChannelSubscriptionOptions{
				FanOutIntervalMs: proto.Uint32(GlobalSettings.GetChannelSettings(channeldpb.ChannelType_SPATIAL).DefaultFanOutIntervalMs),
			}
		} else {
			channelsToSub[chId] = &channeldpb.ChannelSubscriptionOptions{
				FanOutIntervalMs: proto.Uint32(dampSettings.FanOutIntervalMs),
				DataFieldMasks:   dampSettings.DataFieldMasks,
			}
		}
	}

	existingsSubs := make(map[common.ChannelId]*channeldpb.ChannelSubscriptionOptions)
	conn.spatialSubscriptions.Range(func(key, value interface{}) bool {
		chId := key.(common.ChannelId)
		subOptions := value.(*channeldpb.ChannelSubscriptionOptions)
		existingsSubs[chId] = subOptions
		return true
	})

	channelsToUnsub := Difference(existingsSubs, channelsToSub)

	for chId := range channelsToUnsub {
		if ctx.Channel = GetChannel(chId); ctx.Channel == nil {
			continue
		}
		ctx.MsgType = channeldpb.MessageType_UNSUB_FROM_CHANNEL
		ctx.Msg = &channeldpb.UnsubscribedFromChannelMessage{
			ConnId: uint32(ctx.Connection.Id()),
		}
		handleUnsubFromChannel(ctx)
	}

	for chId, subOptions := range channelsToSub {
		if ctx.Channel = GetChannel(chId); ctx.Channel == nil {
			continue
		}
		ctx.MsgType = channeldpb.MessageType_SUB_TO_CHANNEL
		ctx.Msg = &channeldpb.SubscribedToChannelMessage{
			ConnId:     uint32(ctx.Connection.Id()),
			SubOptions: subOptions,
		}
		handleSubToChannel(ctx)
	}
}
