package unrealpb

import (
	"errors"

	"github.com/metaworking/channeld/pkg/channeld"
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

	for netId, entity := range srcData.Entities {
		if entity.Removed {
			delete(dst.Entities, netId)

			entityCh := channeld.GetChannel(common.ChannelId(netId))
			if entityCh != nil {
				entityCh.Logger().Info("removing entity channel from SpatialChannelData.Merge()")
				channeld.RemoveChannel(entityCh)
			}
		} else {
			// Do not merge the SpatialEntityState if it already exists in the channel data
			if _, exists := dst.Entities[netId]; !exists {
				dst.Entities[netId] = entity
			}
		}
	}

	return nil
}

// Entity channel data should implement this interface
type EntityChannelDataWithObjRef interface {
	GetObjRef() *UnrealObjectRef
}

// Implement [channeld.SpatialChannelDataUpdater]
func (dst *SpatialChannelData) AddEntity(entityId channeld.EntityId, msg common.Message) error {
	entityData, ok := msg.(EntityChannelDataWithObjRef)
	if !ok {
		return errors.New("msg is doesn't have GetObjRef()")
	}

	dst.Entities[uint32(entityId)] = &SpatialEntityState{
		ObjRef: entityData.GetObjRef(),
	}

	return nil
}

func (dst *SpatialChannelData) RemoveEntity(entityId channeld.EntityId) error {
	delete(dst.Entities, uint32(entityId))
	return nil
}
