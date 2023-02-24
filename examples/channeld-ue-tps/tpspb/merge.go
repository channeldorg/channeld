package tpspb

import (
	"errors"

	"channeld.clewcat.com/channeld/pkg/channeld"
	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"channeld.clewcat.com/channeld/pkg/common"
	"channeld.clewcat.com/channeld/pkg/unreal"
	"channeld.clewcat.com/channeld/pkg/unrealpb"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// Implement [channeld.ChannelDataInitializer]
func (data *TestRepChannelData) Init() error {
	if data.GameState == nil {
		data.GameState = &unrealpb.GameStateBase{}
	}
	if data.TestGameState == nil {
		data.TestGameState = &TestRepGameState{}
	}
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
	if data.TestRepPlayerControllerStates == nil {
		data.TestRepPlayerControllerStates = make(map[uint32]*TestRepPlayerControllerState)
	}
	if data.TestNPCStates == nil {
		data.TestNPCStates = make(map[uint32]*TestNPCState)
	}
	if data.ActorComponentStates == nil {
		data.ActorComponentStates = make(map[uint32]*unrealpb.ActorComponentState)
	}
	if data.SceneComponentStates == nil {
		data.SceneComponentStates = make(map[uint32]*unrealpb.SceneComponentState)
	}

	return nil
}

// Implement [channeld.ChannelDataInitializer]
func (to *TestRepChannelData) CollectStates(netId uint32, src common.Message) error {
	from, ok := src.(*TestRepChannelData)
	if !ok {
		return errors.New("src is not a TestRepChannelData")
	}

	to.Init()
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
	testRepPlayerControllerStates, exists := from.TestRepPlayerControllerStates[netId]
	if exists {
		to.TestRepPlayerControllerStates[netId] = testRepPlayerControllerStates
	}
	testNPCStates, exists := from.TestNPCStates[netId]
	if exists {
		to.TestNPCStates[netId] = testNPCStates
	}

	return nil
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
					newLoc := newActorState.ReplicatedMovement.Location
					oldLoc := oldActorState.ReplicatedMovement.Location

					var newX, newY float32
					if newLoc.X != nil {
						newX = *newLoc.X
					} else {
						// Use GetX/Y() to avoid violation memory access!
						newX = oldLoc.GetX()
					}
					if newLoc.Y != nil {
						newY = *newLoc.Y
					} else {
						newY = oldLoc.GetY()
					}
					if newX != oldLoc.GetX() || newY != oldLoc.GetY() {
						spatialNotifier.Notify(
							// Swap the Y and Z as UE uses the Z-Up rule but channeld uses the Y-up rule.
							common.SpatialInfo{
								X: float64(oldLoc.GetX()),
								Z: float64(oldLoc.GetY())},
							common.SpatialInfo{
								X: float64(newX),
								Z: float64(newY)},
							func(srcChannelId common.ChannelId, dstChannelId common.ChannelId, handoverData chan common.Message) {
								/* No matter if it's cross-server, always ask for the handover context.
								// Handover happens within the spatial server - no need to ask for the handover context.
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
								} else*/{
									unreal.AddHandoverDataProvider(netId, uint32(srcChannelId), handoverData)
									channeld.GetChannel(srcChannelId).SendToOwner(uint32(unrealpb.MessageType_HANDOVER_CONTEXT), &unrealpb.GetHandoverContextMessage{
										NetId:        netId,
										SrcChannelId: uint32(srcChannelId),
										DstChannelId: uint32(dstChannelId),
									})
									channeld.RootLogger().Info("getting handover context from src server",
										zap.Uint32("srcChannelId", uint32(srcChannelId)),
										zap.Float32("oldX", oldLoc.GetX()), zap.Float32("oldY", oldLoc.GetY()),
										zap.Float32("newX", newX), zap.Float32("newY", newY),
									)
								}
							},
						)
					}
				}
			}
		}
	}

	/* The states should be initialized in Channel.InitData() -> TestRepChannelData.Init().
	// The maps can be nil after InitData().
	initStates(dst)
	*/

	// channeld.ReflectMerge(dst, src, options)

	if srcData.GameState != nil {
		proto.Merge(dst.GameState, srcData.GameState)
	}
	if srcData.TestGameState != nil {
		proto.Merge(dst.TestGameState, srcData.TestGameState)
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

	for netId, newTestRepPlayerControllerState := range srcData.TestRepPlayerControllerStates {
		oldTestRepPlayerControllerState, exists := dst.TestRepPlayerControllerStates[netId]
		if exists {
			proto.Merge(oldTestRepPlayerControllerState, newTestRepPlayerControllerState)
		} else {
			dst.TestRepPlayerControllerStates[netId] = newTestRepPlayerControllerState
		}
	}

	for netId, newTestNPCState := range srcData.TestNPCStates {
		oldTestNPCState, exists := dst.TestNPCStates[netId]
		if exists {
			proto.Merge(oldTestNPCState, newTestNPCState)
		} else {
			dst.TestNPCStates[netId] = newTestNPCState
		}
	}

	for netId, newSceneCompState := range srcData.SceneComponentStates {
		if newSceneCompState.Removed {
			delete(dst.SceneComponentStates, netId)
		} else {
			oldSceneCompState, exists := dst.SceneComponentStates[netId]
			if exists {
				proto.Merge(oldSceneCompState, newSceneCompState)
			} else {
				dst.SceneComponentStates[netId] = newSceneCompState
			}
		}
	}

	for netId, newActorCompState := range srcData.ActorComponentStates {
		if newActorCompState.Removed {
			delete(dst.ActorComponentStates, netId)
		} else {
			oldActorCompState, exists := dst.ActorComponentStates[netId]
			if exists {
				proto.Merge(oldActorCompState, newActorCompState)
			} else {
				dst.ActorComponentStates[netId] = newActorCompState
			}
		}
	}

	// Remove the actor and the corresponding states at last, in case any 'parent' state (e.g. CharacterState) is added to the dst above.
	for netId, newActorState := range srcData.ActorStates {
		if newActorState.Removed {
			delete(dst.ActorStates, netId)
			delete(dst.PawnStates, netId)
			delete(dst.CharacterStates, netId)
			delete(dst.PlayerStates, netId)
			delete(dst.ControllerStates, netId)
			delete(dst.PlayerControllerStates, netId)
			delete(dst.TestRepPlayerControllerStates, netId)
			delete(dst.TestNPCStates, netId)
			channeld.RootLogger().Debug("removed actor state", zap.Uint32("netId", netId))
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

	return nil
}
