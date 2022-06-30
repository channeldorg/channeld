package channeld

import (
	"bufio"
	"encoding/binary"
	"testing"
	"time"

	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestHandleListChannels(t *testing.T) {
	InitLogs()
	InitChannels()
	c := addTestConnection(channeldpb.ConnectionType_SERVER)
	ch0 := globalChannel
	ch1, _ := CreateChannel(channeldpb.ChannelType_PRIVATE, c)
	ch2, _ := CreateChannel(channeldpb.ChannelType_SUBWORLD, c)
	ch3, _ := CreateChannel(channeldpb.ChannelType_SUBWORLD, c)
	ch4, _ := CreateChannel(channeldpb.ChannelType_TEST, c)
	ch5, _ := CreateChannel(channeldpb.ChannelType_TEST, c)
	ch6, _ := CreateChannel(channeldpb.ChannelType_TEST, c)

	ch1.metadata = "aaa"
	ch2.metadata = "bbb"
	ch3.metadata = "ccc"
	ch4.metadata = "aab"
	ch5.metadata = "bbc"
	ch6.metadata = "abc"

	handleListChannel(MessageContext{
		Msg:        &channeldpb.ListChannelMessage{},
		Connection: c,
		Channel:    ch0,
	})
	// No filter - all channels match
	assert.Equal(t, 7, len(c.latestMsg().(*channeldpb.ListChannelResultMessage).Channels))

	handleListChannel(MessageContext{
		Msg: &channeldpb.ListChannelMessage{
			TypeFilter: channeldpb.ChannelType_SUBWORLD,
		},
		Connection: c,
		Channel:    ch0,
	})
	// 2 matches: ch2 and ch3
	assert.Equal(t, 2, len(c.latestMsg().(*channeldpb.ListChannelResultMessage).Channels))

	handleListChannel(MessageContext{
		Msg: &channeldpb.ListChannelMessage{
			MetadataFilters: []string{"aa"},
		},
		Connection: c,
		Channel:    ch0,
	})
	// 2 matches: ch1 and ch4
	assert.Equal(t, 2, len(c.latestMsg().(*channeldpb.ListChannelResultMessage).Channels))

	handleListChannel(MessageContext{
		Msg: &channeldpb.ListChannelMessage{
			MetadataFilters: []string{"bb", "cc"},
		},
		Connection: c,
		Channel:    ch0,
	})
	// 3 matches: ch2, ch3, ch5
	assert.Equal(t, 3, len(c.latestMsg().(*channeldpb.ListChannelResultMessage).Channels))

	handleListChannel(MessageContext{
		Msg: &channeldpb.ListChannelMessage{
			TypeFilter:      channeldpb.ChannelType_TEST,
			MetadataFilters: []string{"a", "b", "c"},
		},
		Connection: c,
		Channel:    ch0,
	})
	// 3 matches: ch4, ch5, ch6
	assert.Equal(t, 3, len(c.latestMsg().(*channeldpb.ListChannelResultMessage).Channels))

	handleListChannel(MessageContext{
		Msg: &channeldpb.ListChannelMessage{
			MetadataFilters: []string{"z"},
		},
		Connection: c,
		Channel:    ch0,
	})
	// no match
	assert.Equal(t, 0, len(c.latestMsg().(*channeldpb.ListChannelResultMessage).Channels))
}

func TestMessageHandlers(t *testing.T) {
	for name, value := range channeldpb.MessageType_value {
		msgType := channeldpb.MessageType(value)
		if msgType == channeldpb.MessageType_INVALID {
			continue
		}
		if msgType >= channeldpb.MessageType_USER_SPACE_START {
			continue
		}
		assert.NotNil(t, MessageMap[msgType], "Missing handler func for message type %s", name)
	}
}

func TestMessageTypeConversion(t *testing.T) {
	var n uint32 = 1
	msgType1 := channeldpb.MessageType(n)
	assert.Equal(t, channeldpb.MessageType_AUTH, msgType1)
	msgType2 := channeldpb.MessageType(uint32(999))
	t.Log(msgType2)
	_, exists := channeldpb.MessageType_name[int32(msgType2)]
	assert.False(t, exists)
}

func BenchmarkProtobufMessagePack(b *testing.B) {
	mp := &channeldpb.MessagePack{
		Broadcast: uint32(channeldpb.BroadcastType_ALL),
		StubId:    0,
		MsgType:   8,
		//BodySize:  0,
		MsgBody: []byte{},
	}

	var size int = 0
	for i := 0; i < b.N; i++ {
		// randomize the channel id between [0, 100)
		mp.ChannelId = uint32(time.Now().Nanosecond() % 100)
		bytes, _ := proto.Marshal(mp)
		size += len(bytes)
	}
	b.Logf("Average packet size: %.2f", float64(size)/float64(b.N))
	// Result:
	// BenchmarkProtobufPacket-12    	9602565	       127.8 ns/op	       4 B/op	       1 allocs/op
	// Average packet size: 4.00
}

type mockWriter struct{}

func (w *mockWriter) Write(p []byte) (n int, err error) {
	return 0, nil
}
func BenchmarkRawMessagePack(b *testing.B) {
	mw := &mockWriter{}
	w := bufio.NewWriterSize(mw, 20)
	var size int = 0
	for i := 0; i < b.N; i++ {
		binary.Write(w, binary.BigEndian, uint32(0))
		binary.Write(w, binary.BigEndian, byte(1))
		binary.Write(w, binary.BigEndian, uint32(0))
		binary.Write(w, binary.BigEndian, uint32(8))
		binary.Write(w, binary.BigEndian, uint32(0))
		binary.Write(w, binary.BigEndian, []byte{})
		size += w.Buffered()
		w.Reset(mw)
	}
	b.Logf("Average buf size: %.2f", float64(size)/float64(b.N))
	// Result:
	// BenchmarkRawPacket-12    	 4974171	       239.3 ns/op	      44 B/op	       6 allocs/op
	// Average buf size: 17.00
}

func TestMessageCopy(t *testing.T) {
	msg := MessageMap[channeldpb.MessageType_CREATE_CHANNEL].msg
	msgCopy := proto.Clone(msg).(*channeldpb.CreateChannelMessage)
	assert.NotEqual(t, MessageMap[channeldpb.MessageType_CREATE_CHANNEL].msg, msgCopy)

	createChannelMsg := &channeldpb.CreateChannelMessage{}
	assert.IsType(t, msg, createChannelMsg)
	assert.IsType(t, msgCopy, createChannelMsg)

	msgCopy.ChannelType = channeldpb.ChannelType_GLOBAL
	proto.Reset(msg)
	assert.Equal(t, channeldpb.ChannelType_GLOBAL, msgCopy.ChannelType)
	proto.Reset(msgCopy)
	assert.Equal(t, channeldpb.ChannelType_UNKNOWN, msgCopy.ChannelType)
}
