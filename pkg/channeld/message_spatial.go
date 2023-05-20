package channeld

import (
	"github.com/metaworking/channeld/pkg/channeldpb"
	"github.com/metaworking/channeld/pkg/common"
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

	clientConn := GetConnection(ConnectionId(msg.ConnId))
	if clientConn == nil {
		ctx.Connection.Logger().Error("cannot find client connection to update spatial interest", zap.Uint32("clientConnId", msg.ConnId))
		return
	}

	spatialChIds, err := spatialController.QueryChannelIds(msg.Query)
	if err != nil {
		ctx.Connection.Logger().Error("error querying spatial channel ids", zap.Error(err))
		return
	}

	channelsToSub := make(map[common.ChannelId]*channeldpb.ChannelSubscriptionOptions)
	for chId, dist := range spatialChIds {
		dampSettings := getSpatialDampingSettings(dist)
		if dampSettings == nil {
			channelsToSub[chId] = &channeldpb.ChannelSubscriptionOptions{
				// DataAccess:       Pointer(channeldpb.ChannelDataAccess_NO_ACCESS),
				FanOutIntervalMs: proto.Uint32(GlobalSettings.GetChannelSettings(channeldpb.ChannelType_SPATIAL).DefaultFanOutIntervalMs),
			}
		} else {
			channelsToSub[chId] = &channeldpb.ChannelSubscriptionOptions{
				// DataAccess:       Pointer(channeldpb.ChannelDataAccess_NO_ACCESS),
				FanOutIntervalMs: proto.Uint32(dampSettings.FanOutIntervalMs),
				DataFieldMasks:   dampSettings.DataFieldMasks,
			}
		}
	}

	existingsSubs := make(map[common.ChannelId]*channeldpb.ChannelSubscriptionOptions)
	clientConn.spatialSubscriptions.Range(func(chId common.ChannelId, subOptions *channeldpb.ChannelSubscriptionOptions) bool {
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
			ConnId: msg.ConnId,
		}
		ctx.Connection = clientConn
		/* Make sure the unsub message is handled in the channel's goroutine
		handleUnsubFromChannel(ctx)
		*/
		ctx.Channel.PutMessageContext(ctx, handleUnsubFromChannel)
	}

	for chId, subOptions := range channelsToSub {
		if ctx.Channel = GetChannel(chId); ctx.Channel == nil {
			continue
		}
		ctx.MsgType = channeldpb.MessageType_SUB_TO_CHANNEL
		ctx.Msg = &channeldpb.SubscribedToChannelMessage{
			ConnId:     msg.ConnId,
			SubOptions: subOptions,
		}
		ctx.Connection = clientConn
		/* Make sure the sub message is handled in the channel's goroutine
		handleSubToChannel(ctx)
		*/
		ctx.Channel.PutMessageContext(ctx, handleSubToChannel)
	}
}
