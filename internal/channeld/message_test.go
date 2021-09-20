package channeld

import (
	"testing"

	"clewcat.com/channeld/proto"
	"github.com/stretchr/testify/assert"
	protobuf "google.golang.org/protobuf/proto"
)

func TestMessageHandlers(t *testing.T) {
	for name, value := range proto.MessageType_value {
		msgType := proto.MessageType(value)
		if msgType == proto.MessageType_INVALID {
			continue
		}
		assert.NotNil(t, MessageMap[msgType], "Missing handler func for message type %s", name)
	}
}

func TestMessageCopy(t *testing.T) {
	msg := MessageMap[proto.MessageType_CREATE_CHANNEL].msg
	msgCopy := protobuf.Clone(msg).(*proto.CreateChannelMessage)
	assert.NotEqual(t, MessageMap[proto.MessageType_CREATE_CHANNEL].msg, msgCopy)

	createChannelMsg := &proto.CreateChannelMessage{}
	assert.IsType(t, msg, createChannelMsg)
	assert.IsType(t, msgCopy, createChannelMsg)

	msgCopy.ChannelType = proto.ChannelType_GLOBAL
	protobuf.Reset(msg)
	assert.Equal(t, proto.ChannelType_GLOBAL, msgCopy.ChannelType)
	protobuf.Reset(msgCopy)
	assert.Equal(t, proto.ChannelType_UNKNOWN, msgCopy.ChannelType)
}
