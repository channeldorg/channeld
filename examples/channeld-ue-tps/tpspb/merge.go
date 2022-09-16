package tpspb

import (
	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"channeld.clewcat.com/channeld/pkg/common"
	"google.golang.org/protobuf/proto"
)

// Implement [channeld.MergeableChannelData]
func (dst *TestRepChannelData) Merge(src proto.Message, options *channeldpb.ChannelDataMergeOptions, spatialNotifier common.SpatialInfoChangedNotifier) error {
	// FIXME: manual copy properties instead of using reflection
	proto.Merge(dst, src)

	/*
		srcMsg, ok := src.(*TestRepChannelData)
		if !ok {
			return errors.New("src is not a TestRepChannelData")
		}

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

	return nil
}
