package tpspb

import (
	"errors"

	"github.com/metaworking/channeld/pkg/channeld"
	"github.com/metaworking/channeld/pkg/channeldpb"
	"github.com/metaworking/channeld/pkg/common"
	"github.com/metaworking/channeld/pkg/unreal"
	"github.com/metaworking/channeld/pkg/unrealpb"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

/*
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
*/

// Implement [channeld.MergeableChannelData]
func (dst *TestRepChannelData) Merge(src common.ChannelDataMessage, options *channeldpb.ChannelDataMergeOptions, spatialNotifier common.SpatialInfoChangedNotifier) error {
	srcData, ok := src.(*TestRepChannelData)
	if !ok {
		return errors.New("src is not a TestRepChannelData")
	}

	if srcData.GameState != nil {
		if dst.GameState == nil {
			dst.GameState = &unrealpb.GameStateBase{}
		}
		proto.Merge(dst.GameState, srcData.GameState)
	}

	if srcData.TestGameState != nil {
		if dst.TestGameState == nil {
			dst.TestGameState = &TestRepGameState{}
		}
		proto.Merge(dst.TestGameState, srcData.TestGameState)
	}

	for netId, newPawnState := range srcData.PawnStates {
		oldPawnState, exists := dst.PawnStates[netId]
		if exists {
			proto.Merge(oldPawnState, newPawnState)
		} else {
			if dst.PawnStates == nil {
				dst.PawnStates = make(map[uint32]*unrealpb.PawnState)
			}
			dst.PawnStates[netId] = newPawnState
		}
	}

	for netId, newCharacterState := range srcData.CharacterStates {
		oldCharacterState, exists := dst.CharacterStates[netId]
		if exists {
			proto.Merge(oldCharacterState, newCharacterState)
		} else {
			if dst.CharacterStates == nil {
				dst.CharacterStates = make(map[uint32]*unrealpb.CharacterState)
			}
			dst.CharacterStates[netId] = newCharacterState
		}
	}

	for netId, newPlayerState := range srcData.PlayerStates {
		oldPlayerState, exists := dst.PlayerStates[netId]
		if exists {
			proto.Merge(oldPlayerState, newPlayerState)
		} else {
			if dst.PlayerStates == nil {
				dst.PlayerStates = make(map[uint32]*unrealpb.PlayerState)
			}
			dst.PlayerStates[netId] = newPlayerState
		}
	}

	for netId, newControllerState := range srcData.ControllerStates {
		oldControllerState, exists := dst.ControllerStates[netId]
		if exists {
			proto.Merge(oldControllerState, newControllerState)
		} else {
			if dst.ControllerStates == nil {
				dst.ControllerStates = make(map[uint32]*unrealpb.ControllerState)
			}
			dst.ControllerStates[netId] = newControllerState
		}
	}

	for netId, newPlayerControllerState := range srcData.PlayerControllerStates {
		oldPlayerControllerState, exists := dst.PlayerControllerStates[netId]
		if exists {
			proto.Merge(oldPlayerControllerState, newPlayerControllerState)
		} else {
			if dst.PlayerControllerStates == nil {
				dst.PlayerControllerStates = make(map[uint32]*unrealpb.PlayerControllerState)
			}
			dst.PlayerControllerStates[netId] = newPlayerControllerState
		}
	}

	for netId, newTestRepPlayerControllerState := range srcData.TestRepPlayerControllerStates {
		oldTestRepPlayerControllerState, exists := dst.TestRepPlayerControllerStates[netId]
		if exists {
			proto.Merge(oldTestRepPlayerControllerState, newTestRepPlayerControllerState)
		} else {
			if dst.TestRepPlayerControllerStates == nil {
				dst.TestRepPlayerControllerStates = make(map[uint32]*TestRepPlayerControllerState)
			}
			dst.TestRepPlayerControllerStates[netId] = newTestRepPlayerControllerState
		}
	}

	for netId, newTestNPCState := range srcData.TestNPCStates {
		oldTestNPCState, exists := dst.TestNPCStates[netId]
		if exists {
			proto.Merge(oldTestNPCState, newTestNPCState)
		} else {
			if dst.TestNPCStates == nil {
				dst.TestNPCStates = make(map[uint32]*TestNPCState)
			}
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
				if dst.SceneComponentStates == nil {
					dst.SceneComponentStates = make(map[uint32]*unrealpb.SceneComponentState)
				}
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
				if dst.ActorComponentStates == nil {
					dst.ActorComponentStates = make(map[uint32]*unrealpb.ActorComponentState)
				}
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
				if dst.ActorStates == nil {
					dst.ActorStates = make(map[uint32]*unrealpb.ActorState)
				}
				dst.ActorStates[netId] = newActorState
			}
		}
	}

	return nil
}

// Implement [channeld.MergeableChannelData]
func (dstData *EntityChannelData) Merge(src common.ChannelDataMessage, options *channeldpb.ChannelDataMergeOptions, spatialNotifier common.SpatialInfoChangedNotifier) error {
	srcData, ok := src.(*EntityChannelData)
	if !ok {
		return errors.New("src is not a EntityChannelData")
	}

	hasHandover := false
	var oldInfo, newInfo *common.SpatialInfo
	if spatialNotifier != nil && dstData.ObjRef != nil {
		// src = the incoming update, dst = existing channel data
		if srcData.ActorState != nil && srcData.ActorState.ReplicatedMovement != nil && srcData.ActorState.ReplicatedMovement.Location != nil &&
			dstData.ActorState != nil && dstData.ActorState.ReplicatedMovement != nil && dstData.ActorState.ReplicatedMovement.Location != nil {
			hasHandover, oldInfo, newInfo = unreal.CheckEntityHandover(*dstData.ObjRef.NetGUID, srcData.ActorState.ReplicatedMovement.Location, dstData.ActorState.ReplicatedMovement.Location)
		}

		if !hasHandover && srcData.SceneComponentState != nil && srcData.SceneComponentState.RelativeLocation != nil &&
			dstData.SceneComponentState != nil && dstData.SceneComponentState.RelativeLocation != nil {
			hasHandover, oldInfo, newInfo = unreal.CheckEntityHandover(*dstData.ObjRef.NetGUID, srcData.SceneComponentState.RelativeLocation, dstData.SceneComponentState.RelativeLocation)
		}
	}

	/* Merging EntityChannelData directly may cause the accumulation of objRef.context,
	 * so we should remove the objRef from source first.
	 */
	srcData.ObjRef = nil
	proto.Merge(dstData, srcData)

	if hasHandover {
		spatialNotifier.Notify(*oldInfo, *newInfo,
			func(srcChannelId common.ChannelId, dstChannelId common.ChannelId, handoverData interface{}) {
				entityId, ok := handoverData.(*channeld.EntityId)
				if !ok {
					channeld.RootLogger().Error("handover data is not an entityId",
						zap.Uint32("srcChannelId", uint32(srcChannelId)),
						zap.Uint32("dstChannelId", uint32(dstChannelId)),
					)
					return
				}
				*entityId = channeld.EntityId(*dstData.ObjRef.NetGUID)

				/* We can't afford to wait in the message queue to send the handover message!
				// Back to the source spatial channel's goroutine
				srcChannel.Execute(func(ch *channeld.Channel) {

				})
				*/

				/*
					// Read the UnrealObjectRef in the source spatial channel (from the entity channel's goroutine)
					srcChannel := channeld.GetChannel(srcChannelId)
					if srcChannel == nil {
						handoverData <- nil
						return
					}

					if srcChannel.GetDataMessage() == nil {
						handoverData <- nil
						return
					}

					spatialChData, ok := srcChannel.GetDataMessage().(*unrealpb.SpatialChannelData)
					if !ok {
						handoverData <- nil
						return
					}

					entities := spatialChData.GetEntities()
					if entities == nil {
						handoverData <- nil
						return
					}

					// CAUTION: running outside the source spatial channel's goroutine may cause concurrent map read/write error!
					entity, exists := entities[netId]
					if !exists {
						handoverData <- nil
						return
					}

					handoverData <- &unrealpb.HandoverData{
						Context: []*unrealpb.HandoverContext{
							{
								Obj: entity.ObjRef,
								// ClientConnId: oldActorState.OwningConnId,
							},
						},
					}
				*/
			},
		)
	}

	return nil
}

// Implement [channeld.HandoverDataMerger]
func (entityData *EntityChannelData) MergeTo(msg common.Message, fullData bool) error {
	handoverData, ok := msg.(*unrealpb.SpatialChannelData)
	if !ok {
		return errors.New("msg is not a SpatialChannelData")
	}

	entityState := &unrealpb.SpatialEntityState{
		ObjRef: entityData.ObjRef,
	}

	if fullData {
		anyData, err := anypb.New(entityData)
		if err != nil {
			return err
		}
		entityState.EntityData = anyData
	}

	if handoverData.Entities == nil {
		handoverData.Entities = make(map[uint32]*unrealpb.SpatialEntityState)
	}
	handoverData.Entities[*entityData.ObjRef.NetGUID] = entityState

	return nil
}

// Implement [unreal.UnrealObjectEntityData]
func (entityData *EntityChannelData) SetObjRef(objRef *unrealpb.UnrealObjectRef) {
	entityData.ObjRef = objRef
}
