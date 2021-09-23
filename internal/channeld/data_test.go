package channeld

import (
	"testing"

	"clewcat.com/channeld/proto"
	"github.com/stretchr/testify/assert"
	protobuf "google.golang.org/protobuf/proto"
)

func TestFanOutChannelData(t *testing.T) {

}

func TestUpdateChannelData(t *testing.T) {

}

func TestNewChannelData(t *testing.T) {

}

func TestProtobufMapMerge(t *testing.T) {
	testMsg := &proto.TestMapMessage{Kv: make(map[uint32]string)}
	testMsg.Kv[1] = "a"
	testMsg.Kv[2] = "b"
	testMsg.Kv[3] = "c"

	updateMsg := &proto.TestMapMessage{Kv: make(map[uint32]string)}
	updateMsg.Kv[2] = "bb"
	updateMsg.Kv[4] = "dd"

	protobuf.Merge(testMsg, updateMsg)

	assert.Equal(t, "a", testMsg.Kv[1])
	assert.Equal(t, "bb", testMsg.Kv[2])
	assert.Equal(t, "c", testMsg.Kv[3])
	assert.Equal(t, "dd", testMsg.Kv[3])
}
