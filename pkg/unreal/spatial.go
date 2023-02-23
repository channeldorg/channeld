package unreal

import "channeld.clewcat.com/channeld/pkg/common"

type ChannelDataCollector interface {
	common.Message
	CollectStates(netId uint32, src common.Message) error
}

// Stores stubs for providing the handover data. The stub will be removed when the source server answers the handover context.
var HandoverDataProviders map[uint64]chan common.Message = make(map[uint64]chan common.Message)

func GetHandoverStub(netId uint32, srcChannelId uint32) uint64 {
	return uint64(srcChannelId)<<32 | uint64(netId)
}
