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

// Executed in the spatial channels
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
		ctxUnsub := MessageContext{ChannelId: ctx.ChannelId}
		if ctxUnsub.Channel = GetChannel(chId); ctxUnsub.Channel == nil {
			continue
		}
		ctxUnsub.MsgType = channeldpb.MessageType_UNSUB_FROM_CHANNEL
		ctxUnsub.Msg = &channeldpb.UnsubscribedFromChannelMessage{
			ConnId: msg.ConnId,
		}
		ctxUnsub.Connection = clientConn
		ctxUnsub.StubId = ctx.StubId

		// Make sure the unsub message is handled in the channel's goroutine
		if ctxUnsub.Channel == ctx.Channel {
			handleUnsubFromChannel(ctxUnsub)
		} else {
			ctxUnsub.Channel.PutMessageContext(ctxUnsub, handleUnsubFromChannel)
		}
	}

	for chId, subOptions := range channelsToSub {
		ctxSub := MessageContext{ChannelId: ctx.ChannelId}
		if ctxSub.Channel = GetChannel(chId); ctxSub.Channel == nil {
			continue
		}
		ctxSub.MsgType = channeldpb.MessageType_SUB_TO_CHANNEL
		ctxSub.Msg = &channeldpb.SubscribedToChannelMessage{
			ConnId:     msg.ConnId,
			SubOptions: subOptions,
		}
		ctxSub.Connection = clientConn

		// Make sure the sub message is handled in the channel's goroutine
		if ctxSub.Channel == ctx.Channel {
			handleSubToChannel(ctxSub)
		} else {
			ctxSub.Channel.PutMessageContext(ctxSub, handleSubToChannel)
		}
	}
}

func handleCreateSpatialChannel(ctx MessageContext, msg *channeldpb.CreateChannelMessage) {
	if ctx.Connection.GetConnectionType() != channeldpb.ConnectionType_SERVER {
		ctx.Connection.Logger().Error("illegal attemp to create Spatial channel from client connection")
		return
	}

	if spatialController == nil {
		ctx.Connection.Logger().Error("illegal attemp to create Spatial channel as there's no controller")
		return
	}

	channels, err := spatialController.CreateChannels(ctx)
	if err != nil {
		ctx.Connection.Logger().Error("failed to create Spatial channel", zap.Error(err))
		return
	}

	resultMsg := &channeldpb.CreateSpatialChannelsResultMessage{
		SpatialChannelId: make([]uint32, len(channels)),
		Metadata:         msg.Metadata,
		OwnerConnId:      uint32(ctx.Connection.Id()),
	}

	for i := range channels {
		resultMsg.SpatialChannelId[i] = uint32(channels[i].id)
	}

	ctx.MsgType = channeldpb.MessageType_CREATE_SPATIAL_CHANNEL
	ctx.Msg = resultMsg
	ctx.Connection.Send(ctx)
	// Also send the response to the GLOBAL channel owner.
	if ownerConn := globalChannel.GetOwner(); ownerConn != nil && ownerConn != ctx.Connection && !ownerConn.IsClosing() {
		ctx.StubId = 0
		ownerConn.Send(ctx)
	}

	for _, newChannel := range channels {
		// Subscribe to channel after creation
		cs, _ := ctx.Connection.SubscribeToChannel(newChannel, msg.SubOptions)
		if cs != nil {
			ctx.Connection.sendSubscribed(ctx, newChannel, ctx.Connection, 0, &cs.options)
		}
	}

	ctx.Connection.Logger().Info("created spatial channel(s)", zap.Uint32s("channelIds", resultMsg.SpatialChannelId))

	// Send the regions info upon the spatial channels creation
	regions, err := spatialController.GetRegions()
	if err != nil {
		ctx.Connection.Logger().Error("failed to send Spatial regions info upon the spatial channels creation",
			zap.Uint32s("channelIds", resultMsg.SpatialChannelId))
		return
	}
	ctx.MsgType = channeldpb.MessageType_SPATIAL_REGIONS_UPDATE
	ctx.Msg = &channeldpb.SpatialRegionsUpdateMessage{
		Regions: regions,
	}
	ctx.Connection.Send(ctx)
}

func handleCreateEntityChannel(ctx MessageContext) {
	// Only the global and spatial channels can create the entity channels
	if ctx.Channel != globalChannel && ctx.Channel.Type() != channeldpb.ChannelType_SPATIAL {
		ctx.Connection.Logger().Error("illegal attemp to create entity channel outside the GLOBAL or SPATIAL channels")
		return
	}

	msg, ok := ctx.Msg.(*channeldpb.CreateEntityChannelMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a CreateEntityChannelMessage, will not be handled.")
		return
	}

	entityChId := common.ChannelId(msg.EntityId)
	if entityChId < GlobalSettings.EntityChannelIdStart {
		ctx.Connection.Logger().Error("illegal attemp to create entity channel with invalid entityId", zap.Uint32("entityId", uint32(entityChId)))
		return
	}

	if entityCh := GetChannel(entityChId); entityCh != nil && !entityCh.IsRemoving() {
		// This could happen when the UE server is restarted but the channeld is not.
		ctx.Connection.Logger().Warn("illegal attemp to create entity channel with duplicated entityId", zap.Uint32("entityId", uint32(entityChId)))
		return
	}

	newChannel := createChannelWithId(entityChId, channeldpb.ChannelType_ENTITY, ctx.Connection)
	newChannel.Logger().Info("created entity channel",
		zap.Uint32("ownerConnId", uint32(newChannel.GetOwner().Id())),
	)

	newChannel.metadata = msg.Metadata
	if msg.Data != nil {
		dataMsg, err := msg.Data.UnmarshalNew()
		if err != nil {
			newChannel.Logger().Error("failed to unmarshal data message for the new channel", zap.Error(err))
		} else {
			newChannel.InitData(dataMsg, msg.MergeOptions)

			// If the entity channel is created from GLOBAL channel(master server), but its data contains spatial info,
			// we should set the owner of the entity channel to the spatial channel's.
			if ctx.Channel == globalChannel && spatialController != nil {
				dataMsgWithSpatialInfo, ok := dataMsg.(EntityChannelDataWithSpatialInfo)
				if ok {
					spatialInfo := dataMsgWithSpatialInfo.GetSpatialInfo()
					if spatialInfo != nil {
						spatialChId, err := spatialController.GetChannelId(*spatialInfo)
						if err != nil {
							ctx.Connection.Logger().Error("failed to set the entity channel owner to the spatial channel's", zap.Error(err))
						} else {
							spatialCh := GetChannel(spatialChId)
							if spatialCh == nil {
								newChannel.Logger().Error("failed to set the entity channel owner as the spatial channel does not exist",
									zap.Uint32("spatialChId", uint32(spatialChId)))
							} else {
								ownerConn := spatialCh.GetOwner()
								if ownerConn != nil && !ownerConn.IsClosing() {
									newChannel.SetOwner(ownerConn)
									newChannel.Logger().Info("set the entity channel owner to the spatial channel's",
										zap.Uint32("spatialChId", uint32(spatialChId)))

									Event_EntityChannelSpatiallyOwned.Broadcast(EntityChannelSpatiallyOwnedEventData{newChannel, spatialCh})

									// Sub-and-send happens at the end of this function

									// Set the messge context so that the CreateChannelResultMessage will be sent to the owner
									// instead of the message sender (the master server).
									ctx.Connection = ownerConn
									ctx.ChannelId = uint32(spatialChId)
								} else {
									newChannel.Logger().Warn("the entity's owning spatial channel does not have an owner connection",
										zap.Uint32("spatialChId", uint32(spatialChId)))
								}
							}
						}
					}
				}
			}
		}
	} else {
		// Channel data should always be initialized
		newChannel.InitData(nil, msg.MergeOptions)
	}

	ctx.Msg = &channeldpb.CreateChannelResultMessage{
		ChannelType: newChannel.channelType,
		Metadata:    newChannel.metadata,
		OwnerConnId: uint32(ctx.Connection.Id()),
		ChannelId:   uint32(newChannel.id),
	}
	ctx.Connection.Send(ctx)

	// Should we also send the result to the GLOBAL channel owner?

	if msg.IsWellKnown {
		// Subscribe ALL the connections to the entity channel
		allConnections.Range(func(_ ConnectionId, conn *Connection) bool {
			// Ignore the well-known entity for server
			if conn.GetConnectionType() == channeldpb.ConnectionType_SERVER {
				return true
			}
			/*
			 */

			// FIXME: different subOptions for different connection?
			cs, shouldSend := conn.SubscribeToChannel(newChannel, nil)
			if shouldSend {
				conn.sendSubscribed(MessageContext{}, newChannel, conn, 0, &cs.options)
				newChannel.Logger().Debug("subscribed existing connection for the well-known entity", zap.Uint32("connId", uint32(conn.Id())))
			}
			return true
		})

		// Add hook to subscribe the new connection to the entity channel
		Event_AuthComplete.ListenFor(newChannel, func(data AuthEventData) {
			// Ignore the well-known entity for server
			if data.Connection.GetConnectionType() == channeldpb.ConnectionType_SERVER {
				return
			}

			if data.AuthResult == channeldpb.AuthResultMessage_SUCCESSFUL {
				// FIXME: different subOptions for different connection?
				// Add some delay so the client won't have to spawn the entity immediately after the auth.
				subOptions := &channeldpb.ChannelSubscriptionOptions{FanOutDelayMs: Pointer(int32(1000))}
				cs, shouldSend := data.Connection.SubscribeToChannel(newChannel, subOptions)
				if shouldSend {
					data.Connection.sendSubscribed(MessageContext{}, newChannel, data.Connection, 0, &cs.options)
					newChannel.Logger().Debug("subscribed new connection for the well-known entity", zap.Uint32("connId", uint32(data.Connection.Id())))
				}
			}
		})
	}

	// Subscribe the owner to channel after creation
	cs, _ := ctx.Connection.SubscribeToChannel(newChannel, msg.SubOptions)
	if cs != nil {
		ctx.Connection.sendSubscribed(ctx, newChannel, ctx.Connection, 0, &cs.options)
	}

	/* We could sub all the connections in the spatial channel to the entity channel here,
	 * but channeld doesn't know the sub options for each connection. So each connection
	 * should subscribe to the entity channel by itself after received the Spawn message.
	 */
}

func handleQuerySpatialChannel(ctx MessageContext) {
	if ctx.Channel != globalChannel {
		ctx.Connection.Logger().Error("illegal attemp to query spatial channel outside the GLOBAL channel")
		return
	}

	msg, ok := ctx.Msg.(*channeldpb.QuerySpatialChannelMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a QuerySpatialChannelMessage, will not be handled.")
		return
	}

	if spatialController == nil {
		ctx.Connection.Logger().Error("cannot query spatial channel as the spatial controller does not exist")
		return
	}

	channelIds := make([]uint32, len(msg.SpatialInfo))
	for i, info := range msg.SpatialInfo {
		channelId, err := spatialController.GetChannelId(common.SpatialInfo{
			X: info.X,
			Y: info.Y,
			Z: info.Z,
		})
		if err != nil {
			ctx.Connection.Logger().Warn("failed to GetChannelId", zap.Error(err),
				zap.Float64("x", info.X), zap.Float64("y", info.Y), zap.Float64("z", info.Z))
		}
		channelIds[i] = uint32(channelId)
	}

	ctx.Msg = &channeldpb.QuerySpatialChannelResultMessage{
		ChannelId: channelIds,
	}
	ctx.Connection.Send(ctx)
}
