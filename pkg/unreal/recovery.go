package unreal

import (
	"github.com/channeldorg/channeld/pkg/channeld"
	"github.com/channeldorg/channeld/pkg/common"
	"github.com/channeldorg/channeld/pkg/unrealpb"
)

// Global and Subworld channels maintain the spawned objects and send them to the recovering connection
type RecoverableChannelDataExtension struct {
	channeld.ChannelDataExtension

	spawnedObjs map[uint32]*unrealpb.UnrealObjectRef
}

func (ext *RecoverableChannelDataExtension) Init(ch *channeld.Channel) {
	ext.spawnedObjs = make(map[uint32]*unrealpb.UnrealObjectRef)
}

func (ext *RecoverableChannelDataExtension) GetRecoveryDataMessage() common.Message {
	return &unrealpb.ChannelRecoveryData{
		ObjRefs: ext.spawnedObjs,
	}
}

func onSpawnObject(ch *channeld.Channel, obj *unrealpb.UnrealObjectRef) {
	ext, ok := ch.Data().Extension().(*RecoverableChannelDataExtension)
	if !ok {
		return
	}
	ext.spawnedObjs[*obj.NetGUID] = obj
}

func onDestroyObject(ch *channeld.Channel, netId uint32) {
	ext, ok := ch.Data().Extension().(*RecoverableChannelDataExtension)
	if !ok {
		return
	}
	delete(ext.spawnedObjs, netId)
}
