package tpspb

import (
	"errors"

	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"channeld.clewcat.com/channeld/pkg/common"
	"google.golang.org/protobuf/proto"
)

// Implement [channeld.MergeableChannelData]
func (dst *TestRepChannelData) Merge(src proto.Message, options *channeldpb.ChannelDataMergeOptions, spatialNotifier common.SpatialInfoChangedNotifier) error {
	srcMsg, ok := src.(*TestRepChannelData)
	if !ok {
		return errors.New("src is not a TestRepChannelData")
	}

	if dst.SceneComponentStates == nil {
		dst.SceneComponentStates = make(map[uint32]*channeldpb.SceneComponentState)
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

	return nil
}
