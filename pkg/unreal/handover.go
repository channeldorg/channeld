package unreal

import (
	"sync"

	"channeld.clewcat.com/channeld/pkg/common"
)

// Channel data messages need to implement this interface to be able to collect the states for handover
type ChannelDataCollector interface {
	common.Message
	CollectStates(netId uint32, src common.Message) error
}

// Stores stubs for providing the handover data. The stub will be removed when the source server answers the handover context.
var handoverDataProviders map[uint64]chan common.Message = make(map[uint64]chan common.Message)
var handoverDataProvidersLock sync.RWMutex

func getHandoverStub(netId uint32, srcChannelId uint32) uint64 {
	return uint64(srcChannelId)<<32 | uint64(netId)
}

func GetHandoverDataProvider(netId uint32, srcChannelId uint32) (chan common.Message, bool) {
	handoverDataProvidersLock.RLock()
	defer handoverDataProvidersLock.RUnlock()
	provider, exists := handoverDataProviders[getHandoverStub(netId, srcChannelId)]
	return provider, exists
}

func AddHandoverDataProvider(netId uint32, srcChannelId uint32, provider chan common.Message) {
	handoverDataProvidersLock.Lock()
	defer handoverDataProvidersLock.Unlock()
	handoverDataProviders[getHandoverStub(netId, srcChannelId)] = provider
}

func RemoveHandoverDataProvider(netId uint32, srcChannelId uint32) {
	handoverDataProvidersLock.Lock()
	defer handoverDataProvidersLock.Unlock()
	delete(handoverDataProviders, getHandoverStub(netId, srcChannelId))
}

func ResetHandoverDataProviders() {
	handoverDataProvidersLock.Lock()
	defer handoverDataProvidersLock.Unlock()
	handoverDataProviders = make(map[uint64]chan common.Message)
}
