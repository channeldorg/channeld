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
		spatialChId, err := channeld.GetSpatialController().GetChannelId(common.SpatialInfo{
			X: float64(*spawnMsg.Location.X),
			Y: float64(*spawnMsg.Location.Y),
			Z: float64(*spawnMsg.Location.Z),
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
	srcMsg, ok := src.(*TestRepChannelData)

	if spatialNotifier != nil {
		if !ok {
			return errors.New("src is not a TestRepChannelData")
		}

		// src = the upcoming update, dst = existing channel data
		for netId, newActorState := range srcMsg.ActorStates {
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
							common.SpatialInfo{
								X: float64(*oldLoc.X),
								Z: float64(*oldLoc.Y)},
							common.SpatialInfo{
								X: float64(newX),
								Z: float64(newY)},
							func() proto.Message {
								defer allSpawnedObjLock.RLocker().Unlock()
								allSpawnedObjLock.RLock()
								return allSpawnedObj[netId]
							},
						)
					}
				}
			}
		}
	}

	/*

		if dst.SceneComponentStates == nil {
			dst.SceneComponentStates = make(map[uint32]*unrealpb.SceneComponentState)
		}

		for k, v := range srcMsg.SceneComponentStates {
			if v.Removed {
				delete(dst.SceneComponentStates, k)
				continue
			}

			trans, exists := dst.SceneComponentStates[k]
			if exists {
				if v.RelativeLocation != nil {
					trans.RelativeLocation = v.RelativeLocation
				}
				if v.RelativeRotation != nil {
					trans.RelativeRotation = v.RelativeRotation
				}
				if v.RelativeScale != nil {
					trans.RelativeScale = v.RelativeScale
				}
			} else {
				dst.SceneComponentStates[k] = v
			}
		}

		for k, v := range srcMsg.CharacterStates {
			if v.Removed {
				delete(dst.CharacterStates, k)
				continue
			}

			char, exists := dst.CharacterStates[k]
			if exists {
				if v.RootMotion != nil {
					if char.RootMotion == nil {
						char.RootMotion = v.RootMotion
					} else {
						// FIXME: manual copy properties instead of using reflection
						proto.Merge(char.RootMotion, v.RootMotion)
					}
				}
				if v.BasedMovement != nil {
					if char.BasedMovement == nil {
						char.BasedMovement = v.BasedMovement
					} else {
						proto.Merge(char.BasedMovement, v.BasedMovement)
					}
				}
				// if v.HasServerLastTransformUpdateTimeStamp() {
				if v.ProtoReflect().Has(v.ProtoReflect().Descriptor().Fields().ByNumber(4)) {
					char.ServerLastTransformUpdateTimeStamp = v.ServerLastTransformUpdateTimeStamp
				}
			}
		}
	*/

	// FIXME: manual copy properties instead of using reflection
	proto.Merge(dst, src)

	// Remove the states from the maps
	for netId, newActorState := range srcMsg.ActorStates {
		if newActorState.Removed {
			delete(dst.ActorStates, netId)
			delete(dst.PawnStates, netId)
			delete(dst.CharacterStates, netId)
			delete(dst.PlayerStates, netId)
			delete(dst.ControllerStates, netId)
			delete(dst.PlayerControllerStates, netId)
			continue
		}
	}

	return nil
}
