package tankspb

import (
	"errors"

	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"channeld.clewcat.com/channeld/pkg/common"
	"google.golang.org/protobuf/proto"
)

// Implement [channeld.MergeableChannelData]
func (dst *TankGameChannelData) Merge(src common.ChannelDataMessage, options *channeldpb.ChannelDataMergeOptions, spatialNotifier common.SpatialInfoChangedNotifier) error {
	srcMsg, ok := src.(*TankGameChannelData)
	if !ok {
		return errors.New("src is not a TankGameChannelData")
	}

	if dst.TankStates == nil {
		dst.TankStates = make(map[uint32]*TankState)
	}

	for k, v := range srcMsg.TankStates {
		if v.Removed {
			delete(dst.TankStates, k)
		} else {
			tank, exits := dst.TankStates[k]
			if exits {
				tank.Health = v.Health
			} else {
				dst.TankStates[k] = v
			}
		}
	}

	if dst.TransformStates == nil {
		dst.TransformStates = make(map[uint32]*channeldpb.TransformState)
	}

	for k, v := range srcMsg.TransformStates {
		if v.Removed {
			delete(dst.TransformStates, k)
			continue
		}

		trans, exists := dst.TransformStates[k]
		if exists {
			if v.Position != nil {
				if trans.Position != nil && spatialNotifier != nil {
					if trans.Position.X != v.Position.X || trans.Position.Z != v.Position.Z {
						spatialNotifier.Notify(
							common.SpatialInfo{
								X: float64(trans.Position.X),
								Z: float64(trans.Position.Z)},
							common.SpatialInfo{
								X: float64(v.Position.X),
								Z: float64(v.Position.Z)},
							func() proto.Message {
								data := &TankGameChannelData{
									TransformStates: map[uint32]*channeldpb.TransformState{
										k: trans, //v,
									},
									TankStates: map[uint32]*TankState{},
								}

								if tankState, exists := dst.TankStates[k]; exists {
									data.TankStates[k] = tankState
								}

								return data
							},
						)
					}
				}
				trans.Position = v.Position
			}
			if v.Rotation != nil {
				trans.Rotation = v.Rotation
			}
			if v.Scale != nil {
				trans.Scale = v.Scale
			}
		} else {
			dst.TransformStates[k] = v
		}
	}

	return nil
}
