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
)

// Stores all UObjects ever spawned in server and sent to client.
// Only in this way we can send the UnrealObjectRef as the handover data.
var allSpawnedObj map[uint32]*unrealpb.UnrealObjectRef = make(map[uint32]*unrealpb.UnrealObjectRef)
var allSpawnedObjLock sync.RWMutex

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

// Implement [channeld.MergeableChannelData]
func (dst *TestRepChannelData) Merge(src proto.Message, options *channeldpb.ChannelDataMergeOptions, spatialNotifier common.SpatialInfoChangedNotifier) error {
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
							func() proto.Message {
								defer allSpawnedObjLock.RLocker().Unlock()
								allSpawnedObjLock.RLock()
								return &unrealpb.HandoverData{
									Obj:          allSpawnedObj[netId],
									ClientConnId: oldActorState.OwningConnId,
								}
							},
						)
					}
				}
			}
		}
	}

	// The maps can be nil after InitData().
	if dst.ActorStates == nil {
		dst.ActorStates = make(map[uint32]*unrealpb.ActorState)
	}
	if dst.PawnStates == nil {
		dst.PawnStates = make(map[uint32]*unrealpb.PawnState)
	}
	if dst.CharacterStates == nil {
		dst.CharacterStates = make(map[uint32]*unrealpb.CharacterState)
	}
	if dst.PlayerStates == nil {
		dst.PlayerStates = make(map[uint32]*unrealpb.PlayerState)
	}
	if dst.ControllerStates == nil {
		dst.ControllerStates = make(map[uint32]*unrealpb.ControllerState)
	}
	if dst.PlayerControllerStates == nil {
		dst.PlayerControllerStates = make(map[uint32]*unrealpb.PlayerControllerState)
	}
	if dst.ActorComponentStates == nil {
		dst.ActorComponentStates = make(map[uint32]*unrealpb.ActorComponentState)
	}
	if dst.SceneComponentStates == nil {
		dst.SceneComponentStates = make(map[uint32]*unrealpb.SceneComponentState)
	}

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
