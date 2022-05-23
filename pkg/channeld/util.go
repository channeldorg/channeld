package channeld

import (
	"sync"
)

func GetNextId(m *map[uint]interface{}, start uint, min uint, max uint) (uint, bool) {
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

func GetNextIdSync(m *sync.Map, start uint, min uint, max uint) (uint, bool) {
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
