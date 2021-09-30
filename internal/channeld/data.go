package channeld

import (
	"container/list"
	"fmt"
	"log"
	"strings"
	"time"

	"clewcat.com/channeld/proto"
	"github.com/iancoleman/strcase"
	"github.com/mennanov/fmutils"
	protobuf "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type ChannelDataMessage = Message //protoreflect.Message

type ChannelData struct {
	msgType         proto.MessageType
	msg             ChannelDataMessage
	updateMsg       ChannelDataMessage
	updateMsgBuffer *list.List
}

type FanOutConnection struct {
	connId         ConnectionId
	lastFanOutTime time.Time
}

type UpdateMsgBufferElement struct {
	updateMsg  ChannelDataMessage
	updateTime time.Time
}

const (
	MaxUpdateMsgBufferSize = 512
)

func NewChannelData(channelType proto.ChannelType) *ChannelData {
	channelTypeName := channelType.String()
	dataTypeName := fmt.Sprintf("channeld.%sChannelDataMessage",
		strcase.ToCamel(strings.ToLower(channelTypeName)))
	dataType, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(dataTypeName))
	if err != nil {
		log.Panicln("Failed to create data for channel type", channelTypeName)
	}
	msgTypeName := "CHANNEL_DATA_" + strings.ToUpper(channelTypeName)
	msgType, exists := proto.MessageType_value[msgTypeName]
	if !exists {
		log.Panicln("Can't find data update message type by name", msgTypeName)
	}
	return &ChannelData{
		msgType:         proto.MessageType(msgType),
		msg:             dataType.New().Interface(),
		updateMsgBuffer: list.New(),
	}
}

func (d *ChannelData) OnUpdate(updateMsg Message) {
	protobuf.Merge(d.msg, updateMsg)
	if d.updateMsg == nil {
		d.updateMsg = updateMsg
	} else {
		protobuf.Merge(d.updateMsg, updateMsg)
	}

	d.updateMsgBuffer.PushBack(&UpdateMsgBufferElement{
		updateMsg:  updateMsg,
		updateTime: time.Now(),
	})
	if d.updateMsgBuffer.Len() > MaxUpdateMsgBufferSize {
		d.updateMsgBuffer.Remove(d.updateMsgBuffer.Front())
	}
}

func (ch *Channel) tickData() {
	if ch.Data().msg == nil {
		return
	}
	/*
		for connId, cs := range ch.subscribedConnections {
			if cs.options.FanOutInterval <= 0 || time.Since(cs.lastFanOutTime) >= cs.options.FanOutInterval {
				c := GetConnection(connId)
				if c == nil {
					continue
				}
				ch.FanOutDataUpdate(c, cs)
			}
		}
	*/
	bufp := ch.Data().updateMsgBuffer.Front()
	var accumulatedUpdateMsg ChannelDataMessage = nil
	var lastUpdateTime time.Time

	for fe := ch.fanOutQueue.Front(); fe != nil; fe = fe.Next() {
		foc := fe.Value.(*FanOutConnection)
		c := GetConnection(foc.connId)
		if c == nil {
			continue
		}
		cs := ch.subscribedConnections[foc.connId]
		if cs == nil {
			continue
		}

		nextFanOutTime := foc.lastFanOutTime.Add(cs.options.FanOutInterval)
		if time.Now().After(nextFanOutTime) {
			if foc.lastFanOutTime.IsZero() {
				// Send the whole data for the first time
				ch.FanOutDataUpdate(c, cs, ch.Data().msg)
			} else if bufp != nil {
				if foc.lastFanOutTime.After(lastUpdateTime) {
					lastUpdateTime = foc.lastFanOutTime
				}

				for be := bufp.Value.(*UpdateMsgBufferElement); be.updateTime.After(lastUpdateTime) && be.updateTime.Before(nextFanOutTime); bufp = bufp.Next() {
					if accumulatedUpdateMsg == nil {
						accumulatedUpdateMsg = protobuf.Clone(be.updateMsg)
					} else {
						protobuf.Merge(accumulatedUpdateMsg, be.updateMsg)
					}
					lastUpdateTime = be.updateTime
				}

				if accumulatedUpdateMsg != nil {
					ch.FanOutDataUpdate(c, cs, accumulatedUpdateMsg)
				}
			}

			foc.lastFanOutTime = time.Now()

			temp := fe
			fe = fe.Next()
			// Move the fanned-out connection to the back of the queue
			for be := ch.fanOutQueue.Back(); be != nil; be = be.Prev() {
				if be.Value.(*FanOutConnection).lastFanOutTime.Before(foc.lastFanOutTime) {
					ch.fanOutQueue.MoveAfter(temp, be)
				}
			}
			fe = fe.Prev()
		}
	}
}

func (ch *Channel) FanOutDataUpdate(c *Connection, cs *ChannelSubscription, updateMsg ChannelDataMessage) {
	fmutils.Filter(updateMsg, cs.options.DataFieldMasks)
	c.SendWithChannel(ch.id, ch.Data().msgType, updateMsg)
	c.Flush()
	// cs.lastFanOutTime = time.Now()
	// cs.fanOutDataMsg = nil
}
