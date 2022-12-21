package tpspb

import (
	"errors"
	"sync"

	"channeld.clewcat.com/channeld/pkg/channeld"
	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"channeld.clewcat.com/channeld/pkg/common"
	"channeld.clewcat.com/channeld/pkg/unrealpb"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// Stores all UObjects ever spawned in server and sent to client.
// Only in this way we can send the UnrealObjectRef as the handover data.
var allSpawnedObj map[uint32]*unrealpb.UnrealObjectRef = make(map[uint32]*unrealpb.UnrealObjectRef)
var allSpawnedObjLock sync.RWMutex
var handoverDataProviders map[uint32]chan common.Message = make(map[uint32]chan common.Message)

func HandleUnrealSpawnObject(ctx channeld.MessageContext) {
	defer channeld.HandleServerToClientUserMessage(ctx)

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

	// Update the message's spaital channelId based on the actor's location
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
			}
		}
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

func initStates(data *TestRepChannelData) {
	if data.ActorStates == nil {
		data.ActorStates = make(map[uint32]*unrealpb.ActorState)
	}
	if data.PawnStates == nil {
		data.PawnStates = make(map[uint32]*unrealpb.PawnState)
	}
	if data.CharacterStates == nil {
		data.CharacterStates = make(map[uint32]*unrealpb.CharacterState)
	}
	if data.PlayerStates == nil {
		data.PlayerStates = make(map[uint32]*unrealpb.PlayerState)
	}
	if data.ControllerStates == nil {
		data.ControllerStates = make(map[uint32]*unrealpb.ControllerState)
	}
	if data.PlayerControllerStates == nil {
		data.PlayerControllerStates = make(map[uint32]*unrealpb.PlayerControllerState)
	}
	if data.ActorComponentStates == nil {
		data.ActorComponentStates = make(map[uint32]*unrealpb.ActorComponentState)
	}
	if data.SceneComponentStates == nil {
		data.SceneComponentStates = make(map[uint32]*unrealpb.SceneComponentState)
	}
}

func collectStates(netId uint32, from *TestRepChannelData, to *TestRepChannelData) {
	initStates(to)
	actorState, exists := from.ActorStates[netId]
	if exists {
		to.ActorStates[netId] = actorState
	}
	pawnState, exists := from.PawnStates[netId]
	if exists {
		to.PawnStates[netId] = pawnState
	}
	characterState, exists := from.CharacterStates[netId]
	if exists {
		to.CharacterStates[netId] = characterState
	}
	playerState, exists := from.PlayerStates[netId]
	if exists {
		to.PlayerStates[netId] = playerState
	}
	controllerState, exists := from.ControllerStates[netId]
	if exists {
		to.ControllerStates[netId] = controllerState
	}
	playerControllerStates, exists := from.PlayerControllerStates[netId]
	if exists {
		to.PlayerControllerStates[netId] = playerControllerStates
	}
}

func HandleHandoverContextResult(ctx channeld.MessageContext) {
	msg, ok := ctx.Msg.(*unrealpb.GetHandoverContextResultMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a GetHandoverContextResultMessage, will not be handled.")
		return
	}

	provider, exists := handoverDataProviders[msg.NetId]
	if !exists {
		ctx.Connection.Logger().Error("could not find the handover data provider", zap.Uint32("netId", msg.NetId))
		return
	}

	defer delete(handoverDataProviders, msg.NetId)

	// No context - no handover will happen
	if len(msg.Context) == 0 {
		provider <- nil
		return
	}

	if ctx.Channel.GetDataMessage() == nil {
		ctx.Channel.Logger().Error("channel data message is nil")
		return
	}
	fullChannelData := ctx.Channel.GetDataMessage().(*TestRepChannelData)

	handoverChannelData := &TestRepChannelData{}
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
		collectStates(*handoverCtx.Obj.NetGUID, fullChannelData, handoverChannelData)
	}

	// Don't forget to merge the handover channel data to the dst channel, otherwise it may miss some states.
	dstChannel := channeld.GetChannel(common.ChannelId(msg.DstChannelId))
	if dstChannel != nil {
		dstChannelDataMsg := dstChannel.GetDataMessage()
		if dstChannelDataMsg == nil {
			dstChannel.Data().OnUpdate(handoverChannelData, dstChannel.GetTime(), nil)
		} else {
			// Should we let the dst channel fan out the update?
			// For now we don't. It's easier for the UE servers to control the sequence of spawning and update.
			dstChannelDataMsg.(*TestRepChannelData).Merge(handoverChannelData, nil, nil)
		}
	}

	anyData, err := anypb.New(handoverChannelData)
	if err != nil {
		ctx.Connection.Logger().Error("failed to marshal handover data", zap.Error(err), zap.Uint32("netId", msg.NetId))
		return
	}

	provider <- &unrealpb.HandoverData{
		Context:     msg.Context,
		ChannelData: anyData,
	}
}

// Implement [channeld.MergeableChannelData]
func (dst *TestRepChannelData) Merge(src common.ChannelDataMessage, options *channeldpb.ChannelDataMergeOptions, spatialNotifier common.SpatialInfoChangedNotifier) error {
	srcData, ok := src.(*TestRepChannelData)
	if !ok {
		return errors.New("src is not a TestRepChannelData")
	}

	if spatialNotifier != nil {
		// src = the upcoming update, dst = existing channel data
		for netId, newActorState := range srcData.ActorStates {
			oldActorState, exists := dst.ActorStates[netId]
			if exists {
				if newActorState.ReplicatedMovement != nil && newActorState.ReplicatedMovement.Location != nil &&
					oldActorState.ReplicatedMovement != nil && oldActorState.ReplicatedMovement.Location != nil {
					oldLoc := oldActorState.ReplicatedMovement.Location
					newLoc := newActorState.ReplicatedMovement.Location
					var newX, newY float32
					if newLoc.X != nil {
						newX = *newLoc.X
					} else {
						newX = *oldLoc.X
					}
					if newLoc.Y != nil {
						newY = *newLoc.Y
					} else {
						newY = *oldLoc.Y
					}
					if newX != *oldLoc.X || newY != *oldLoc.Y {
						spatialNotifier.Notify(
							// Swap the Y and Z as UE uses the Z-Up rule but channeld uses the Y-up rule.
							common.SpatialInfo{
								X: float64(*oldLoc.X),
								Z: float64(*oldLoc.Y)},
							common.SpatialInfo{
								X: float64(newX),
								Z: float64(newY)},
							func(srcChannelId common.ChannelId, dstChannelId common.ChannelId, handoverData chan common.Message) {
								// Handover happens within the spatial server - no need to ask the handover context.
								if channeld.GetChannel(srcChannelId).IsSameOwner(channeld.GetChannel(dstChannelId)) {
									defer allSpawnedObjLock.RLocker().Unlock()
									allSpawnedObjLock.RLock()
									handoverData <- &unrealpb.HandoverData{
										Context: []*unrealpb.HandoverContext{
											{
												Obj:          allSpawnedObj[netId],
												ClientConnId: oldActorState.OwningConnId,
											},
										},
									}
								} else {
									handoverDataProviders[netId] = handoverData
									channeld.GetChannel(srcChannelId).SendToOwner(uint32(unrealpb.MessageType_HANDOVER_CONTEXT), &unrealpb.GetHandoverContextMessage{
										NetId:        netId,
										SrcChannelId: uint32(srcChannelId),
										DstChannelId: uint32(dstChannelId),
									})
									channeld.RootLogger().Info("getting handover context from src server", zap.Uint32("srcChannelId", uint32(srcChannelId)))
								}
							},
						)
					}
				}
			}
		}
	}

	// The maps can be nil after InitData().
	initStates(dst)

	// channeld.ReflectMerge(dst, src, options)

	if srcData.GameState != nil {
		proto.Merge(dst.GameState, srcData.GameState)
	}

	for netId, newActorState := range srcData.ActorStates {
		// Remove the states from the maps
		if newActorState.Removed {
			delete(dst.ActorStates, netId)
			delete(dst.PawnStates, netId)
			delete(dst.CharacterStates, netId)
			delete(dst.PlayerStates, netId)
			delete(dst.ControllerStates, netId)
			delete(dst.PlayerControllerStates, netId)
			continue
		} else {
			oldActorState, exists := dst.ActorStates[netId]
			if exists {
				proto.Merge(oldActorState, newActorState)
			} else {
				dst.ActorStates[netId] = newActorState
			}
		}
	}

	for netId, newPawnState := range srcData.PawnStates {
		oldPawnState, exists := dst.PawnStates[netId]
		if exists {
			proto.Merge(oldPawnState, newPawnState)
		} else {
			dst.PawnStates[netId] = newPawnState
		}
	}

	for netId, newCharacterState := range srcData.CharacterStates {
		oldCharacterState, exists := dst.CharacterStates[netId]
		if exists {
			proto.Merge(oldCharacterState, newCharacterState)
		} else {
			dst.CharacterStates[netId] = newCharacterState
		}
	}

	for netId, newPlayerState := range srcData.PlayerStates {
		oldPlayerState, exists := dst.PlayerStates[netId]
		if exists {
			proto.Merge(oldPlayerState, newPlayerState)
		} else {
			dst.PlayerStates[netId] = newPlayerState
		}
	}

	for netId, newControllerState := range srcData.ControllerStates {
		oldControllerState, exists := dst.ControllerStates[netId]
		if exists {
			proto.Merge(oldControllerState, newControllerState)
		} else {
			dst.ControllerStates[netId] = newControllerState
		}
	}

	for netId, newPlayerControllerState := range srcData.PlayerControllerStates {
		oldPlayerControllerState, exists := dst.PlayerControllerStates[netId]
		if exists {
			proto.Merge(oldPlayerControllerState, newPlayerControllerState)
		} else {
			dst.PlayerControllerStates[netId] = newPlayerControllerState
		}
	}

	for netId, newActorCompState := range srcData.ActorComponentStates {
		if newActorCompState.Removed {
			delete(dst.ActorComponentStates, netId)
			delete(dst.SceneComponentStates, netId)
		} else {
			oldActorCompState, exists := dst.ActorComponentStates[netId]
			if exists {
				proto.Merge(oldActorCompState, newActorCompState)
			} else {
				dst.ActorComponentStates[netId] = newActorCompState
			}
		}
	}

	for netId, newSceneCompState := range srcData.SceneComponentStates {
		oldSceneCompState, exists := dst.SceneComponentStates[netId]
		if exists {
			proto.Merge(oldSceneCompState, newSceneCompState)
		} else {
			dst.SceneComponentStates[netId] = newSceneCompState
		}
	}

	return nil
}
