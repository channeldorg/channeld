package channeld

import (
	"testing"

	"clewcat.com/channeld/proto"
	"github.com/mennanov/fmutils"
	"github.com/stretchr/testify/assert"
	protobuf "google.golang.org/protobuf/proto"
)

func TestFanOutChannelData(t *testing.T) {

}

func TestUpdateChannelData(t *testing.T) {

}

func TestDataFieldMasks(t *testing.T) {
	nestedMsg := &proto.TestFieldMaskMessage_NestedMessage{
		P1: 1,
		P2: 2,
	}
	testMsg := &proto.TestFieldMaskMessage{
		Name: "test",
		Msg:  nestedMsg,
		List: []*proto.TestFieldMaskMessage_NestedMessage{nestedMsg},
		Kv: map[int64]*proto.TestFieldMaskMessage_NestedMessage{
			10: nestedMsg,
		},
	}

	filteredMsg1 := protobuf.Clone(testMsg)
	fmutils.Filter(filteredMsg1, []string{"name"})
	t.Log(filteredMsg1.(*proto.TestFieldMaskMessage).String())

	filteredMsg2 := protobuf.Clone(testMsg)
	fmutils.Filter(filteredMsg2, []string{"msg.p1"})
	t.Log(filteredMsg2.(*proto.TestFieldMaskMessage).String())

	filteredMsg3 := protobuf.Clone(testMsg)
	fmutils.Filter(filteredMsg3, []string{"list.p2"})
	t.Log(filteredMsg3.(*proto.TestFieldMaskMessage).String())

	filteredMsg4 := protobuf.Clone(testMsg)
	fmutils.Filter(filteredMsg4, []string{"kv"})
	t.Log(filteredMsg4.(*proto.TestFieldMaskMessage).String())
}

func TestNewChannelData(t *testing.T) {
	globalData := NewChannelData(proto.ChannelType_GLOBAL)
	assert.NotNil(t, globalData)
	assert.Equal(t, proto.MessageType_CHANNEL_DATA_GLOBAL, globalData.msgType)
	assert.IsType(t, &proto.GlobalChannelDataMessage{}, globalData.msg)
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
