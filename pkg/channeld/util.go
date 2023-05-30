package channeld

import (
	"hash/maphash"
	"net"
	"strings"
	"sync"

	"github.com/metaworking/channeld/pkg/common"
)

type LocalhostAddr struct {
	NetworkName string
}

func (addr *LocalhostAddr) Network() string {
	return addr.NetworkName
}

func (addr *LocalhostAddr) Addr() string {
	return "localhost"
}

func GetIP(addr net.Addr) string {
	switch addr := addr.(type) {
	case *net.UDPAddr:
		return addr.IP.String()
	case *net.TCPAddr:
		return addr.IP.String()
	}
	return strings.Split(addr.String(), ":")[0]
}

func GetNextId(m *map[uint32]interface{}, start uint32, min uint32, max uint32) (uint32, bool) {
	for i := min; i <= max; i++ {
		if _, exists := (*m)[start]; !exists {
			return start, true
		}
		if start < max {
			start++
		} else {
			start = min
		}
	}

	return 0, false
}

func GetNextIdSync(m *sync.Map, start common.ChannelId, min common.ChannelId, max common.ChannelId) (common.ChannelId, bool) {
	for i := min; i <= max; i++ {
		if _, exists := m.Load(start); !exists {
			return start, true
		}
		if start < max {
			start++
		} else {
			start = min
		}
	}

	return 0, false
}

type UintId interface {
	common.ChannelId | ConnectionId | EntityId | uint32
}
type MapRead[K comparable, V any] interface {
	Load(key K) (value V, ok bool)
}

func GetNextIdTyped[K UintId, V any](m MapRead[K, V], start K, min K, max K) (K, bool) {
	for i := min; i <= max; i++ {
		if _, exists := m.Load(start); !exists {
			return start, true
		}
		if start < max {
			start++
		} else {
			start = min
		}
	}

	return 0, false
}

func UintIdHasher[T UintId]() func(maphash.Seed, T) uint64 {
	return func(_ maphash.Seed, id T) uint64 {
		return uint64(id)
	}
}

func HashString(s string) uint32 {
	hash := uint32(17)
	for c := range s {
		hash = hash*31 + uint32(c)
	}
	return hash
}

func Pointer[K any](val K) *K {
	return &val
}

// Returns a map of all the keys in thisMap that are not in otherMap
func Difference[K comparable, V any](thisMap map[K]V, otherMap map[K]V) map[K]V {
	result := make(map[K]V)
	for k, v := range thisMap {
		if _, exists := otherMap[k]; !exists {
			result[k] = v
		}
	}
	return result
}

func CopyArray[FROM UintId, TO UintId](arr []FROM) []TO {
	result := make([]TO, len(arr))
	for i, v := range arr {
		result[i] = TO(v)
	}
	return result
}
