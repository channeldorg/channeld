package channeld

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetNextId(t *testing.T) {
	m := make(map[uint]interface{})
	var index uint = 1
	var ok bool

	index, _ = GetNextId(&m, index, 1, 3)
	assert.EqualValues(t, 1, index)
	m[index] = "aaa"

	index, _ = GetNextId(&m, index, 1, 3)
	assert.EqualValues(t, 2, index)
	m[index] = "bbb"

	index, _ = GetNextId(&m, index, 1, 3)
	assert.EqualValues(t, 3, index)
	m[index] = "ccc"

	_, ok = GetNextId(&m, index, 1, 3)
	assert.True(t, ok)
}

func BenchmarkGetNextId(b *testing.B) {
	m := make(map[uint]interface{})
	var index uint = 1
	var ok bool

	for i := 0; i < b.N; i++ {
		index, ok = GetNextId(&m, index, 1, 65535)
		if ok {
			m[index] = i
		} else {
			break
		}
	}
}

func TestGetNextIdSync(t *testing.T) {
	m := sync.Map{}
	var index uint = 1
	var ok bool

	index, _ = GetNextIdSync(&m, index, 1, 3)
	assert.EqualValues(t, 1, index)
	m.Store(index, "aaa")

	index, _ = GetNextIdSync(&m, index, 1, 3)
	assert.EqualValues(t, 2, index)
	m.Store(index, "bbb")

	index, _ = GetNextIdSync(&m, index, 1, 3)
	assert.EqualValues(t, 3, index)
	m.Store(index, "ccc")

	_, ok = GetNextIdSync(&m, index, 1, 3)
	assert.True(t, ok)

}

// 3x slower as of BenchmarkGetNextId
func BenchmarkGetNextIdSync(b *testing.B) {
	m := sync.Map{}
	var index uint = 1
	var ok bool

	for i := 0; i < b.N; i++ {
		index, ok = GetNextIdSync(&m, index, 1, 65535)
		if ok {
			m.Store(index, i)
		} else {
			break
		}
	}
}
