package unrealpb

import (
	"errors"

	"github.com/metaworking/channeld/pkg/channeldpb"
	"github.com/metaworking/channeld/pkg/common"
)

// Implement [channeld.HandoverDataWithPayload]
func (data *HandoverData) ClearPayload() {
	data.ChannelData = nil
}

// Implement [channeld.ChannelDataInitializer]
func (data *SpatialChannelData) Init() error {
	data.Entities = make(map[uint32]*SpatialEntityState)
	return nil
}

// Implement [channeld.MergeableChannelData]
func (dst *SpatialChannelData) Merge(src common.ChannelDataMessage, options *channeldpb.ChannelDataMergeOptions, spatialNotifier common.SpatialInfoChangedNotifier) error {
	srcData, ok := src.(*SpatialChannelData)
	if !ok {
		return errors.New("src is not a SpatialChannelData")
	}

	for id, entity := range srcData.Entities {
		if _, exists := dst.Entities[id]; !exists {
			dst.Entities[id] = entity
		}
	}

	return nil
}
