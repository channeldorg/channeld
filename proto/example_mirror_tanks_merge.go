package proto

import (
	"errors"

	protobuf "google.golang.org/protobuf/proto"
)

func (dst *TankGameChannelData) Merge(src protobuf.Message, options *ChannelDataMergeOptions) error {
	srcMsg, ok := src.(*TankGameChannelData)
	if !ok {
		return errors.New("src is not a TankGameChannelData")
	}

	if dst.TransformStates == nil {
		dst.TransformStates = make(map[uint32]*TransformState)
	}

	for k, v := range srcMsg.TransformStates {
		if v.Removed {
			delete(dst.TransformStates, k)
		} else {
			trans, exists := dst.TransformStates[k]
			if exists {
				if v.Position != nil {
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

	return nil
}
