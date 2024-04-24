package unreal

import (
	"github.com/metaworking/channeld/pkg/channeld"
	"github.com/metaworking/channeld/pkg/channeldpb"
	"github.com/metaworking/channeld/pkg/common"
	"github.com/metaworking/channeld/pkg/unrealpb"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func InitMessageHandlers() {
	channeld.RegisterMessageHandler(uint32(unrealpb.MessageType_SPAWN), &channeldpb.ServerForwardMessage{}, handleUnrealSpawnObject)
	channeld.RegisterMessageHandler(uint32(unrealpb.MessageType_DESTROY), &channeldpb.ServerForwardMessage{}, handleUnrealDestroyObject)
}

// Executed in the spatial channels or the GLOBAL channel (no-spatial scenario)
func handleUnrealSpawnObject(ctx channeld.MessageContext) {
	// server -> channeld -> client
	msg, ok := ctx.Msg.(*channeldpb.ServerForwardMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a ServerForwardMessage, will not be handled.")
		return
	}

	spawnMsg := &unrealpb.SpawnObjectMessage{}
	err := proto.Unmarshal(msg.Payload, spawnMsg)
	if err != nil {
		ctx.Connection.Logger().Error("failed to unmarshal SpawnObjectMessage")
		return
	}

	if spawnMsg.Obj == nil {
		ctx.Connection.Logger().Error("SpawnObjectMessage doesn't have the 'Obj' field")
		return
	}

	if spawnMsg.Obj.NetGUID == nil || *spawnMsg.Obj.NetGUID == 0 {
		ctx.Connection.Logger().Error("invalid NetGUID in SpawnObjectMessage")
		return
	}

	/*
		if len(spawnMsg.Obj.Context) == 0 {
			ctx.Connection.Logger().Warn("empty context in SpawnObjectMessage", zap.Uint32("netId", *spawnMsg.Obj.NetGUID))
		}
	*/

	// Update the message's spatial channelId based on the actor's location
	oldChId := *spawnMsg.ChannelId
	if spawnMsg.Location != nil {
		// Swap the Y and Z as UE uses the Z-Up rule but channeld uses the Y-up rule.
		spatialChId, err := channeld.GetSpatialController().GetChannelId(common.SpatialInfo{
			X: float64(*spawnMsg.Location.X),
			Y: float64(*spawnMsg.Location.Z),
			Z: float64(*spawnMsg.Location.Y),
		})
		if err != nil {
			ctx.Connection.Logger().Warn("failed to GetChannelId", zap.Error(err),
				zap.Float32("x", *spawnMsg.Location.X),
				zap.Float32("y", *spawnMsg.Location.Y),
				zap.Float32("z", *spawnMsg.Location.Z))
			return
		}
		*spawnMsg.ChannelId = uint32(spatialChId)
		if *spawnMsg.ChannelId != oldChId {
			newPayload, err := proto.Marshal(spawnMsg)
			if err == nil {
				msg.Payload = newPayload
				// Update the channel and let the new channel handle the message. Otherwise race conditions may happen.
				ctx.Channel = channeld.GetChannel(spatialChId)
				if ctx.Channel != nil {
					ctx.Channel.Execute(func(ch *channeld.Channel) {
						addSpatialEntity(ch, spawnMsg.Obj)
					})
					ctx.Channel.PutMessageContext(ctx, channeld.HandleServerToClientUserMessage)
				} else {
					ctx.Connection.Logger().Error("failed to handle the ServerForwardMessage as the new spatial channel doesn't exist", zap.Uint32("newChId", *spawnMsg.ChannelId))
				}
			} else {
				ctx.Connection.Logger().Error("failed to marshal the new payload")
			}
		} else {
			// ChannelId is not updated; handle the forward message in current channel.
			addSpatialEntity(ctx.Channel, spawnMsg.Obj)
			channeld.HandleServerToClientUserMessage(ctx)
		}
	} else {
		addSpatialEntity(ctx.Channel, spawnMsg.Obj)
		channeld.HandleServerToClientUserMessage(ctx)
	}

	/*
		defer allSpawnedObjLock.Unlock()
		allSpawnedObjLock.Lock()
		allSpawnedObj[*spawnMsg.Obj.NetGUID] = spawnMsg.Obj
		channeld.RootLogger().Debug("stored UnrealObjectRef from spawn message",
			zap.Uint32("netId", *spawnMsg.Obj.NetGUID),
			zap.Uint32("oldChId", oldChId),
			zap.Uint32("newChId", *spawnMsg.ChannelId),
		)
	*/

	// Entity channel should already be created by the spatial server.
	entityChannel := channeld.GetChannel(common.ChannelId(*spawnMsg.Obj.NetGUID))
	if entityChannel == nil {
		return
	}

	// Set the objRef of the entity channel's data
	entityChannel.Execute(func(ch *channeld.Channel) {
		if entityData, ok := ch.GetDataMessage().(UnrealObjectEntityData); ok {
			entityData.SetObjRef(spawnMsg.Obj)
			ch.Logger().Debug("set entity data's objRef")
		}
	})
}

// Entity channel data that contains an UnrealObjectRef should implement this interface.
type UnrealObjectEntityData interface {
	SetObjRef(objRef *unrealpb.UnrealObjectRef)
}

func addSpatialEntity(ch *channeld.Channel, objRef *unrealpb.UnrealObjectRef) {
	if ch.Type() != channeldpb.ChannelType_SPATIAL {
		return
	}

	if ch.GetDataMessage() == nil {
		return
	}

	spatialChannelData, ok := ch.GetDataMessage().(*unrealpb.SpatialChannelData)
	if !ok {
		ch.Logger().Warn("channel data is not a SpatialChannelData",
			zap.String("dataType", string(ch.GetDataMessage().ProtoReflect().Descriptor().FullName())))
		return
	}

	entityState := &unrealpb.SpatialEntityState{ObjRef: &unrealpb.UnrealObjectRef{}}
	proto.Merge(entityState.ObjRef, objRef)
	spatialChannelData.Entities[*objRef.NetGUID] = entityState
	ch.Logger().Debug("added spatial entity", zap.Uint32("netId", *objRef.NetGUID))
}

func removeSpatialEntity(ch *channeld.Channel, netId uint32) {
	if ch.Type() != channeldpb.ChannelType_SPATIAL {
		return
	}

	if ch.GetDataMessage() == nil {
		return
	}

	spatialChannelData, ok := ch.GetDataMessage().(*unrealpb.SpatialChannelData)
	if !ok {
		ch.Logger().Warn("channel data is not a SpatialChannelData",
			zap.String("dataType", string(ch.GetDataMessage().ProtoReflect().Descriptor().FullName())))
		return
	}

	delete(spatialChannelData.Entities, netId)
	ch.Logger().Debug("removed spatial entity", zap.Uint32("netId", netId))
}

func handleUnrealDestroyObject(ctx channeld.MessageContext) {
	// server -> channeld -> client
	msg, ok := ctx.Msg.(*channeldpb.ServerForwardMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a ServerForwardMessage, will not be handled.")
		return
	}

	destroyMsg := &unrealpb.DestroyObjectMessage{}
	err := proto.Unmarshal(msg.Payload, destroyMsg)
	if err != nil {
		ctx.Connection.Logger().Error("failed to unmarshal DestroyObjectMessage")
		return
	}

	removeSpatialEntity(ctx.Channel, destroyMsg.NetId)
	// Send/broadcast the message
	channeld.HandleServerToClientUserMessage(ctx)

	entityCh := channeld.GetChannel(common.ChannelId(destroyMsg.NetId))
	if entityCh != nil {
		entityCh.Logger().Info("removing entity channel from unrealpb.DestroyObjectMessage")
		channeld.RemoveChannel(entityCh)
	}
}
