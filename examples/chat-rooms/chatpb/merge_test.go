package chatpb

import (
	"fmt"
	"testing"
	"time"

	"github.com/channeldorg/channeld/pkg/channeld"
	"github.com/channeldorg/channeld/pkg/channeldpb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

type Message = proto.Message //protoreflect.ProtoMessage

func genTestingMsgs(count int, senderTime int64, content string) []*ChatMessage {
	msgs := make([]*ChatMessage, count)
	for i := 0; i < count; i++ {
		msgs[i] = &ChatMessage{
			Sender:   "c0",
			SendTime: senderTime,
			Content:  fmt.Sprintf("%s-%d", content, i+1),
		}
	}
	return msgs
}

func TestMergeByTimeSpanLimit(t *testing.T) {
	channeld.InitLogs()

	testChannel, _ := channeld.CreateChannel(channeldpb.ChannelType_TEST, nil)

	listSizeLimit := 100
	channelData := &ChatChannelData{ChatMessages: make([]*ChatMessage, 0)}
	testChannel.InitData(
		channelData,
		&channeldpb.ChannelDataMergeOptions{
			ListSizeLimit: uint32(listSizeLimit),
			TruncateTop:   true,
		},
	)

	channelStartTime := channeld.ChannelTime(time.Now().Unix())

	TimeSpanLimit = time.Millisecond * 300
	sendInterval := time.Millisecond * 100

	// ---------------- step 1 ----------------
	startTime := time.Now()
	MsgNum := 100
	updateData := &ChatChannelData{ChatMessages: genTestingMsgs(MsgNum, startTime.UnixMilli(), "s1")}
	testChannel.Data().OnUpdate(updateData, channelStartTime, 0, nil)
	// 0 -> 100
	assert.Equal(t, 100, len(channelData.ChatMessages))
	time.Sleep(sendInterval - time.Since(startTime))
	// ---------------- step 1 ----------------

	// ---------------- step 2 ----------------
	startTime = time.Now()
	MsgNum = 10
	updateData = &ChatChannelData{ChatMessages: genTestingMsgs(MsgNum, startTime.UnixMilli(), "s2")}
	testChannel.Data().OnUpdate(updateData, channelStartTime, 0, nil)
	// 100 -> 110
	assert.Equal(t, 110, len(channelData.ChatMessages))
	time.Sleep(sendInterval - time.Since(startTime))
	// ---------------- step 2 ----------------

	// ---------------- step 3 ----------------
	startTime = time.Now()
	MsgNum = 200
	updateData = &ChatChannelData{ChatMessages: genTestingMsgs(MsgNum, startTime.UnixMilli(), "s3")}
	testChannel.Data().OnUpdate(updateData, channelStartTime, 0, nil)
	// 110 -> 310
	assert.Equal(t, 310, len(channelData.ChatMessages))
	time.Sleep(sendInterval - time.Since(startTime))
	// ---------------- step 3 ----------------

	// ---------------- step 4 ----------------
	startTime = time.Now()
	MsgNum = 5
	updateData = &ChatChannelData{ChatMessages: genTestingMsgs(MsgNum, startTime.UnixMilli(), "s4")}
	testChannel.Data().OnUpdate(updateData, channelStartTime, 0, nil)
	// 310 -> 215
	// - s1:100
	// + s4:5
	assert.Equal(t, 215, len(channelData.ChatMessages))
	time.Sleep(sendInterval - time.Since(startTime))
	// ---------------- step 4 ----------------

	// ---------------- step 5 ----------------
	startTime = time.Now()
	MsgNum = 0
	updateData = &ChatChannelData{ChatMessages: genTestingMsgs(MsgNum, startTime.UnixMilli(), "s5")}
	testChannel.Data().OnUpdate(updateData, channelStartTime, 0, nil)
	// 215 -> 205
	// - s2:10
	assert.Equal(t, 205, len(channelData.ChatMessages))
	time.Sleep(sendInterval - time.Since(startTime))
	// ---------------- step 5 ----------------

	// ---------------- step 6 ----------------
	startTime = time.Now()
	MsgNum = 0
	updateData = &ChatChannelData{ChatMessages: genTestingMsgs(MsgNum, startTime.UnixMilli(), "s6")}
	testChannel.Data().OnUpdate(updateData, channelStartTime, 0, nil)
	// 205 -> 100
	// - s3:105
	assert.Equal(t, 100, len(channelData.ChatMessages))
	time.Sleep(sendInterval - time.Since(startTime))
	// ---------------- step 6 ----------------

	// ---------------- step 7 ----------------
	startTime = time.Now()
	MsgNum = 120
	updateData = &ChatChannelData{ChatMessages: genTestingMsgs(MsgNum, startTime.UnixMilli(), "s7")}
	testChannel.Data().OnUpdate(updateData, channelStartTime, 0, nil)
	// 100 -> 120
	// - s3:95
	// - s4:5
	// + s7:120
	assert.Equal(t, 120, len(channelData.ChatMessages))
	time.Sleep(sendInterval - time.Since(startTime))
	// ---------------- step 7 ----------------

	// ---------------- step 8 ----------------
	startTime = time.Now()
	MsgNum = 0
	updateData = &ChatChannelData{ChatMessages: genTestingMsgs(MsgNum, startTime.UnixMilli(), "s8")}
	testChannel.Data().OnUpdate(updateData, channelStartTime, 0, nil)
	// 120 -> 120
	assert.Equal(t, 120, len(channelData.ChatMessages))
	time.Sleep(sendInterval - time.Since(startTime))
	// ---------------- step 8 ----------------

	// ---------------- step 9 ----------------
	startTime = time.Now()
	MsgNum = 0
	updateData = &ChatChannelData{ChatMessages: genTestingMsgs(MsgNum, startTime.UnixMilli(), "s9")}
	testChannel.Data().OnUpdate(updateData, channelStartTime, 0, nil)
	// 120 -> 120
	assert.Equal(t, 120, len(channelData.ChatMessages))
	time.Sleep(sendInterval - time.Since(startTime))
	// ---------------- step 9 ----------------

	// ---------------- step 10 ----------------
	startTime = time.Now()
	MsgNum = 0
	updateData = &ChatChannelData{ChatMessages: genTestingMsgs(MsgNum, startTime.UnixMilli(), "s10")}
	testChannel.Data().OnUpdate(updateData, channelStartTime, 0, nil)
	// 120 -> 100
	// - s7:20
	assert.Equal(t, 100, len(channelData.ChatMessages))
	time.Sleep(sendInterval - time.Since(startTime))
	// ---------------- step 10 ----------------
}

func TestMergeWithoutTimeSpanLimit(t *testing.T) {
	channeld.InitLogs()

	testChannel, _ := channeld.CreateChannel(channeldpb.ChannelType_TEST, nil)

	listSizeLimit := 100
	channelData := &ChatChannelData{ChatMessages: make([]*ChatMessage, 0)}
	testChannel.InitData(
		channelData,
		&channeldpb.ChannelDataMergeOptions{
			ListSizeLimit: uint32(listSizeLimit),
			TruncateTop:   true,
		},
	)

	channelStartTime := channeld.ChannelTime(time.Now().Unix())

	TimeSpanLimit = 0
	sendInterval := time.Millisecond * 100

	// ---------------- step 1 ----------------
	startTime := time.Now()
	MsgNum := 100
	updateData := &ChatChannelData{ChatMessages: genTestingMsgs(MsgNum, startTime.UnixMilli(), "s1")}
	testChannel.Data().OnUpdate(updateData, channelStartTime, 0, nil)
	// 0 -> 100
	assert.Equal(t, 100, len(channelData.ChatMessages))
	time.Sleep(sendInterval - time.Since(startTime))
	// ---------------- step 1 ----------------

	// ---------------- step 2 ----------------
	startTime = time.Now()
	MsgNum = 10
	updateData = &ChatChannelData{ChatMessages: genTestingMsgs(MsgNum, startTime.UnixMilli(), "s2")}
	testChannel.Data().OnUpdate(updateData, channelStartTime, 0, nil)
	assert.Equal(t, 100, len(channelData.ChatMessages))
	time.Sleep(sendInterval - time.Since(startTime))
	// ---------------- step 2 ----------------

	// ---------------- step 3 ----------------
	startTime = time.Now()
	MsgNum = 200
	updateData = &ChatChannelData{ChatMessages: genTestingMsgs(MsgNum, startTime.UnixMilli(), "s3")}
	testChannel.Data().OnUpdate(updateData, channelStartTime, 0, nil)
	assert.Equal(t, 100, len(channelData.ChatMessages))
	time.Sleep(sendInterval - time.Since(startTime))
	// ---------------- step 3 ----------------
}

func TestMergeByLongTimeSpanLimit(t *testing.T) {
	channeld.InitLogs()

	testChannel, _ := channeld.CreateChannel(channeldpb.ChannelType_TEST, nil)

	listSizeLimit := 100
	channelData := &ChatChannelData{ChatMessages: make([]*ChatMessage, 0)}
	testChannel.InitData(
		channelData,
		&channeldpb.ChannelDataMergeOptions{
			ListSizeLimit: uint32(listSizeLimit),
			TruncateTop:   true,
		},
	)

	channelStartTime := channeld.ChannelTime(time.Now().Unix())

	TimeSpanLimit = time.Hour
	sendInterval := time.Millisecond * 100

	// ---------------- step 1 ----------------
	startTime := time.Now()
	MsgNum := 100
	updateData := &ChatChannelData{ChatMessages: genTestingMsgs(MsgNum, startTime.UnixMilli(), "s1")}
	testChannel.Data().OnUpdate(updateData, channelStartTime, 0, nil)
	// 0 -> 100
	assert.Equal(t, 100, len(channelData.ChatMessages))
	time.Sleep(sendInterval - time.Since(startTime))
	// ---------------- step 1 ----------------

	// ---------------- step 2 ----------------
	startTime = time.Now()
	MsgNum = 10
	updateData = &ChatChannelData{ChatMessages: genTestingMsgs(MsgNum, startTime.UnixMilli(), "s2")}
	testChannel.Data().OnUpdate(updateData, channelStartTime, 0, nil)
	assert.Equal(t, 110, len(channelData.ChatMessages))
	time.Sleep(sendInterval - time.Since(startTime))
	// ---------------- step 2 ----------------

	// ---------------- step 3 ----------------
	startTime = time.Now()
	MsgNum = 200
	updateData = &ChatChannelData{ChatMessages: genTestingMsgs(MsgNum, startTime.UnixMilli(), "s3")}
	testChannel.Data().OnUpdate(updateData, channelStartTime, 0, nil)
	assert.Equal(t, 310, len(channelData.ChatMessages))
	time.Sleep(sendInterval - time.Since(startTime))
	// ---------------- step 3 ----------------
}
