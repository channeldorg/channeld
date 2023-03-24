package unreal

import (
	"sync"

	"github.com/metaworking/channeld/pkg/channeld"
	"github.com/metaworking/channeld/pkg/channeldpb"
	"github.com/metaworking/channeld/pkg/common"
	"github.com/metaworking/channeld/pkg/unrealpb"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// Stores all UObjects ever spawned in server and sent to client.
// Only in this way we can send the UnrealObjectRef as the handover data.
var allSpawnedObj map[uint32]*unrealpb.UnrealObjectRef = make(map[uint32]*unrealpb.UnrealObjectRef)
var allSpawnedObjLock sync.RWMutex

func InitMessageHandlers() {
	channeld.RegisterMessageHandler(uint32(unrealpb.MessageType_SPAWN), &channeldpb.ServerForwardMessage{}, handleUnrealSpawnObject)
	channeld.RegisterMessageHandler(uint32(unrealpb.MessageType_HANDOVER_CONTEXT), &unrealpb.GetHandoverContextResultMessage{}, handleHandoverContextResult)
	channeld.RegisterMessageHandler(uint32(unrealpb.MessageType_GET_UNREAL_OBJECT_REF), &unrealpb.GetUnrealObjectRefMessage{}, handleGetUnrealObjectRef)

	channeld.Event_GlobalChannelUnpossessed.Listen(func(struct{}) {
		// Global server exits. Clear up all the cache.
		allSpawnedObjLock.Lock()
		defer allSpawnedObjLock.Unlock()
		allSpawnedObj = make(map[uint32]*unrealpb.UnrealObjectRef)
		resetHandoverDataProviders()
	})
}

func handleGetUnrealObjectRef(ctx channeld.MessageContext) {
	msg, ok := ctx.Msg.(*unrealpb.GetUnrealObjectRefMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a GetUnrealObjectRefMessage, will not be handled.")
		return
	}

	resultMsg := &unrealpb.GetUnrealObjectRefResultMessage{}
	resultMsg.ObjRef = make([]*unrealpb.UnrealObjectRef, 0)

	defer allSpawnedObjLock.RLocker().Unlock()
	allSpawnedObjLock.RLock()
	for _, netId := range msg.NetGUID {
		objRef, exists := allSpawnedObj[netId]
		if exists {
			resultMsg.ObjRef = append(resultMsg.ObjRef, objRef)
		}
	}
	ctx.Msg = resultMsg
	ctx.Connection.Send(ctx)
}

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
					ctx.Channel.PutMessageContext(ctx, channeld.HandleServerToClientUserMessage)
				} else {
					ctx.Connection.Logger().Error("failed to handle the ServerForwardMessage as the new spatial channel doesn't exist", zap.Uint32("newChId", *spawnMsg.ChannelId))
				}
			} else {
				ctx.Connection.Logger().Error("failed to marshal the new payload")
			}
		} else {
			// ChannelId is not updated; handle the forward message in current channel.
			channeld.HandleServerToClientUserMessage(ctx)
		}
	} else {
		channeld.HandleServerToClientUserMessage(ctx)
	}

	defer allSpawnedObjLock.Unlock()
	allSpawnedObjLock.Lock()
	allSpawnedObj[*spawnMsg.Obj.NetGUID] = spawnMsg.Obj
	channeld.RootLogger().Debug("stored UnrealObjectRef from spawn message",
		zap.Uint32("netId", *spawnMsg.Obj.NetGUID),
		zap.Uint32("oldChId", oldChId),
		zap.Uint32("newChId", *spawnMsg.ChannelId),
	)
}

// Runs in the source spatial channel
func handleHandoverContextResult(ctx channeld.MessageContext) {
	msg, ok := ctx.Msg.(*unrealpb.GetHandoverContextResultMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a GetHandoverContextResultMessage, will not be handled.")
		return
	}

	provider, exists := getHandoverDataProvider(msg.NetId, msg.SrcChannelId)
	if !exists {
		ctx.Connection.Logger().Error("could not find the handover data provider", zap.Uint32("netId", msg.NetId))
		return
	}

	defer removeHandoverDataProvider(msg.NetId, msg.SrcChannelId)

	// No context - no handover will happen
	if len(msg.Context) == 0 {
		provider <- nil
		return
	}

	fullChannelData := ctx.Channel.GetDataMessage()

	if fullChannelData == nil {
		ctx.Channel.Logger().Error("channel data message is nil")
		provider <- nil
		return
	}

	handoverChannelData := fullChannelData.ProtoReflect().New().Interface()

	collector, ok := handoverChannelData.(ChannelDataCollector)
	if ok {
		for _, handoverCtx := range msg.Context {
			if handoverCtx.Obj == nil || handoverCtx.Obj.NetGUID == nil || *handoverCtx.Obj.NetGUID == 0 {
				ctx.Connection.Logger().Error("corrupted handover context", zap.Uint32("netId", msg.NetId))
				continue
			}
			// Make sure the object is fully exported, so the destination server can spawn it properly.
			if handoverCtx.Obj.NetGUIDBunch == nil {
				allSpawnedObjLock.RLock()
				objRef, exists := allSpawnedObj[*handoverCtx.Obj.NetGUID]
				allSpawnedObjLock.RLocker().Unlock()
				if exists {
					handoverCtx.Obj = objRef
				} else {
					ctx.Connection.Logger().Warn("handover obj is not fully exported yet", zap.Uint32("netId", msg.NetId))
				}
			}
			// Should be goroutine-safe, as the handler is running in the source channel that owns fullChannelData.
			collector.CollectStates(*handoverCtx.Obj.NetGUID, fullChannelData)
		}
	} else {
		ctx.Connection.Logger().Warn("channel data message is not a ChannelDataCollector, the states of the context objects will not be included in the handover data", zap.Uint32("netId", msg.NetId))
	}

	anyData, err := anypb.New(handoverChannelData)
	if err != nil {
		ctx.Connection.Logger().Error("failed to marshal handover data", zap.Error(err), zap.Uint32("netId", msg.NetId))
		provider <- nil
		return
	}

	// Don't forget to merge the handover channel data to the dst channel, otherwise it may miss some states.
	dstChannel := channeld.GetChannel(common.ChannelId(msg.DstChannelId))
	if dstChannel != nil {
		dstChannelDataMsg := dstChannel.GetDataMessage()
		if dstChannelDataMsg == nil {
			// Set the data directly
			dstChannel.Data().OnUpdate(handoverChannelData, dstChannel.GetTime(), 0, nil)
		} else {
			/* Calling Merge() causes concurrent map read and map write as the handler is running in the source channel.
			mergeable, ok := dstChannelDataMsg.(channeld.MergeableChannelData)
			if ok {
				// Should we let the dst channel fan out the update?
				// For now we don't. It's easier for the UE servers to control the sequence of spawning and update.
				mergeable.Merge(handoverChannelData, nil, nil)
			} else {
				proto.Merge(dstChannelDataMsg, handoverChannelData)
			}
			*/
			updateMsg := &channeldpb.ChannelDataUpdateMessage{
				Data: anyData,
			}
			dstChannel.PutMessageInternal(channeldpb.MessageType_CHANNEL_DATA_UPDATE, updateMsg)
		}
	}

	handoverData := &unrealpb.HandoverData{
		Context: msg.Context,
	}

	srcChannel := channeld.GetChannel(common.ChannelId(msg.SrcChannelId))
	/* In order to spawn objects with full states in the client, we need to provider the channel data for all handover.
	// Only provide channel data for cross-server handover
	*/
	if srcChannel != nil && dstChannel != nil /*&& !srcChannel.IsSameOwner(dstChannel)*/ {
		handoverData.ChannelData = anyData
	}

	provider <- handoverData
}
