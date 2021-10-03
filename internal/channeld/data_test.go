package channeld

import (
	"container/list"
	"net"
	"testing"
	"time"

	"clewcat.com/channeld/proto"
	"github.com/indiest/fmutils"
	"github.com/stretchr/testify/assert"
	protobuf "google.golang.org/protobuf/proto"
)

func TestFanOutChannelData(t *testing.T) {
	// See the test case in [the design doc](doc/design.md#fan-out)
	serverConn, clientConn := net.Pipe()
	InitConnections(3, "../../config/server_conn_fsm.json", "../../config/client_conn_fsm.json")
	InitChannels()
	// We need to manually tick. Set the interval to a very large value.
	globalChannel.tickInterval = time.Hour
	dataMsg := globalChannel.Data().msg.(*proto.GlobalChannelDataMessage)
	dataMsg.Title = "a"

	sc := AddConnection(serverConn, SERVER)
	cc1 := AddConnection(clientConn, CLIENT)
	cc2 := AddConnection(clientConn, CLIENT)
	//ch := CreateChannel(proto.ChannelType_GLOBAL, sc)
	sc.SubscribeToChannel(globalChannel, nil)
	cc1.SubscribeToChannel(globalChannel, &proto.ChannelSubscriptionOptions{
		FanOutIntervalMs: 50,
	})

	channelStartTime := ChannelTime(100 * int64(time.Millisecond))
	// F0 = the whole data
	globalChannel.tickData(channelStartTime)
	assert.Equal(t, 1, len(cc1.sendQueue))
	assert.Equal(t, 0, len(cc2.sendQueue))

	cc2.SubscribeToChannel(globalChannel, &proto.ChannelSubscriptionOptions{
		FanOutIntervalMs: 100,
	})
	// F1 = no data, F7 = the whole data
	globalChannel.tickData(channelStartTime.AddMs(50))
	assert.Equal(t, 1, len(cc1.sendQueue))
	assert.Equal(t, 1, len(cc2.sendQueue))

	// U1 arrives
	u1 := &proto.GlobalChannelDataMessage{Title: "b"}
	globalChannel.Data().OnUpdate(u1, channelStartTime.AddMs(60))

	// F2 = U1
	globalChannel.tickData(channelStartTime.AddMs(100))
	assert.Equal(t, 2, len(cc1.sendQueue))
}

func TestListMoveElement(t *testing.T) {
	list := list.New()
	list.PushBack("a")
	list.PushBack("b")
	list.PushBack("c")
	assert.Equal(t, "a", list.Front().Value)

	e := list.Front().Next()
	assert.Equal(t, "b", e.Value)
	temp := e.Prev()
	list.MoveToBack(e)
	e = temp.Next()
	assert.Equal(t, "c", e.Value)
}

func TestDataMergeOptions(t *testing.T) {
	dstMsg := &proto.TestMergeMessage{
		List: []string{"a", "b", "c"},
		Kv: map[int64]*proto.TestMergeMessage_StringWrapper{
			1: &proto.TestMergeMessage_StringWrapper{Content: "aa"},
			2: &proto.TestMergeMessage_StringWrapper{Content: "bb"},
		},
	}

	srcMsg := &proto.TestMergeMessage{
		List: []string{"d", "e"},
		Kv: map[int64]*proto.TestMergeMessage_StringWrapper{
			1: nil,
			2: &proto.TestMergeMessage_StringWrapper{Content: "bbb"},
		},
	}

	mergedMsg1 := protobuf.Clone(dstMsg).(*proto.TestMergeMessage)
	mergeOptions1 := &DataMergeOptions{
		ShouldReplaceRepeated: true,
	}
	mergeWithOptions(mergedMsg1, srcMsg, mergeOptions1)
	assert.Equal(t, 2, len(mergedMsg1.List))
	assert.Equal(t, "e", mergedMsg1.List[1])

	mergedMsg2 := protobuf.Clone(dstMsg).(*proto.TestMergeMessage)
	mergeOptions2 := &DataMergeOptions{
		ListSizeLimit: 4,
	}
	mergeWithOptions(mergedMsg2, srcMsg, mergeOptions2)
	assert.Equal(t, 4, len(mergedMsg2.List))
	assert.Equal(t, "b", mergedMsg2.List[0])

	mergedMsg3 := protobuf.Clone(dstMsg).(*proto.TestMergeMessage)
	mergeOptions3 := &DataMergeOptions{
		ShouldDeleteNilMapValue: true,
	}
	mergeWithOptions(mergedMsg3, srcMsg, mergeOptions3)
	assert.Equal(t, 1, len(mergedMsg3.Kv))
	assert.Nil(t, mergedMsg3.Kv[1])
	assert.Equal(t, "bbb", mergedMsg3.Kv[2].Content)
}

func TestNewChannelData(t *testing.T) {
	globalData := NewChannelData(proto.ChannelType_GLOBAL, nil)
	assert.NotNil(t, globalData)
	assert.Equal(t, proto.MessageType_CHANNEL_DATA_GLOBAL, globalData.msgType)
	assert.IsType(t, &proto.GlobalChannelDataMessage{}, globalData.msg)
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
		Kv1: map[int64]*proto.TestFieldMaskMessage_NestedMessage{
			10: nestedMsg,
		},
		Kv2: map[int64]string{
			100: "hello",
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
	fmutils.Filter(filteredMsg4, []string{"kv1.p1", "kv1.p2", "kv1.p3"})
	t.Log(filteredMsg4.(*proto.TestFieldMaskMessage).String())
	fmutils.Prune(filteredMsg4, []string{"kv1.p1"})
	t.Log(filteredMsg4.(*proto.TestFieldMaskMessage).String())

	filteredMsg5 := protobuf.Clone(testMsg)
	fmutils.Filter(filteredMsg5, []string{"kv2.a"})
	t.Log(filteredMsg5.(*proto.TestFieldMaskMessage).String())
}

func TestProtobufMapMerge(t *testing.T) {
	testMsg := &proto.TestMapMessage{
		Kv:  make(map[uint32]string),
		Kv2: make(map[uint32]*proto.TestMapMessage_StringWrapper),
	}
	testMsg.Kv[1] = "a"
	testMsg.Kv[2] = "b"
	testMsg.Kv[3] = "c"
	testMsg.Kv[4] = "d"

	testMsg.Kv2[1] = &proto.TestMapMessage_StringWrapper{Content: "a"}
	testMsg.Kv2[2] = &proto.TestMapMessage_StringWrapper{Content: "b"}

	updateMsg := &proto.TestMapMessage{
		Kv:  make(map[uint32]string),
		Kv2: make(map[uint32]*proto.TestMapMessage_StringWrapper),
	}
	updateMsg.Kv[2] = "bb"
	updateMsg.Kv[3] = ""
	updateMsg.Kv[4] = "dd"

	updateMsg.Kv2[1] = nil

	protobuf.Merge(testMsg, updateMsg)

	assert.Equal(t, "a", testMsg.Kv[1])
	assert.Equal(t, "bb", testMsg.Kv[2])
	assert.Equal(t, "", testMsg.Kv[3])
	assert.Equal(t, "dd", testMsg.Kv[4])

	/* By default, protobuf ignores the nil value
	assert.Equal(t, nil, testMsg.Kv2[1])
	*/
	assert.NotEqual(t, nil, testMsg.Kv2[1])
	assert.Equal(t, "b", testMsg.Kv2[2].Content)

}
