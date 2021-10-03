package channeld

import (
	"container/list"
	"fmt"
	"log"
	"strings"

	"clewcat.com/channeld/proto"
	"github.com/iancoleman/strcase"
	"github.com/indiest/fmutils"
	protobuf "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type ChannelDataMessage = Message //protoreflect.Message

type DataMergeOptions struct {
	// By default, Protobuf appends the src list to the dst list. Setting this option to true will replace the dst list with the src list.
	ShouldReplaceRepeated bool
	// By default, Protobuf ignores the nil value when merging the map. Setting this option to true will delete the key-value pair when merging.
	ShouldDeleteNilMapValue bool
	// If the value is greater than 0, truncate the top elements of the list when oversized.
	ListSizeLimit int
}

type ChannelData struct {
	mergeOptions *DataMergeOptions
	msgType      proto.MessageType
	msg          ChannelDataMessage
	//updateMsg       ChannelDataMessage
	updateMsgBuffer *list.List
}

type fanOutConnection struct {
	connId         ConnectionId
	lastFanOutTime ChannelTime
}

type updateMsgBufferElement struct {
	updateMsg  ChannelDataMessage
	updateTime ChannelTime
}

const (
	MaxUpdateMsgBufferSize = 512
)

func NewChannelData(channelType proto.ChannelType, mergeOptions *DataMergeOptions) *ChannelData {
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

func (d *ChannelData) OnUpdate(updateMsg Message, t ChannelTime) {
	mergeWithOptions(d.msg, updateMsg, d.mergeOptions)
	/*
		if d.updateMsg == nil {
			d.updateMsg = updateMsg
		} else {
			mergeWithOptions(d.updateMsg, updateMsg, d.mergeOptions)
		}
	*/

	d.updateMsgBuffer.PushBack(&updateMsgBufferElement{
		updateMsg:  updateMsg,
		updateTime: t,
	})
	if d.updateMsgBuffer.Len() > MaxUpdateMsgBufferSize {
		d.updateMsgBuffer.Remove(d.updateMsgBuffer.Front())
	}
}

func (ch *Channel) tickData(t ChannelTime) {
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
	var lastUpdateTime ChannelTime
	fe := ch.fanOutQueue.Front()

	for i := 0; i < ch.fanOutQueue.Len(); i++ {
		foc := fe.Value.(*fanOutConnection)
		c := GetConnection(foc.connId)
		if c == nil {
			fe = fe.Next()
			continue
		}
		cs := ch.subscribedConnections[foc.connId]
		if cs == nil {
			fe = fe.Next()
			continue
		}

		nextFanOutTime := foc.lastFanOutTime.AddMs(cs.options.FanOutIntervalMs)
		if t >= nextFanOutTime {
			if foc.lastFanOutTime == 0 {
				// Send the whole data for the first time
				ch.fanOutDataUpdate(c, cs, ch.Data().msg)
			} else if bufp != nil {
				if foc.lastFanOutTime >= lastUpdateTime {
					lastUpdateTime = foc.lastFanOutTime
				}

				for be := bufp.Value.(*updateMsgBufferElement); bufp != nil && be.updateTime >= lastUpdateTime && be.updateTime <= nextFanOutTime; bufp = bufp.Next() {
					if accumulatedUpdateMsg == nil {
						accumulatedUpdateMsg = protobuf.Clone(be.updateMsg)
					} else {
						mergeWithOptions(accumulatedUpdateMsg, be.updateMsg, ch.data.mergeOptions)
					}
					lastUpdateTime = be.updateTime
				}

				if accumulatedUpdateMsg != nil {
					ch.fanOutDataUpdate(c, cs, accumulatedUpdateMsg)
				}
			}

			foc.lastFanOutTime = t

			temp := fe.Next()
			// Move the fanned-out connection to the back of the queue
			for be := ch.fanOutQueue.Back(); be != nil; be = be.Prev() {
				if be.Value.(*fanOutConnection).lastFanOutTime <= foc.lastFanOutTime {
					ch.fanOutQueue.MoveAfter(fe, be)
					fe = temp
					break
				}
			}
		} else {
			fe = fe.Next()
		}
	}
}

func (ch *Channel) fanOutDataUpdate(c *Connection, cs *ChannelSubscription, updateMsg ChannelDataMessage) {
	fmutils.Filter(updateMsg, cs.options.DataFieldMasks)
	c.SendWithChannel(ch.id, ch.Data().msgType, updateMsg)
	// cs.lastFanOutTime = time.Now()
	// cs.fanOutDataMsg = nil
}

func mergeWithOptions(dst Message, src Message, options *DataMergeOptions) {
	protobuf.Merge(dst, src)
	if options != nil {
		dst.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			if fd.IsList() {
				if options.ShouldReplaceRepeated {
					dst.ProtoReflect().Set(fd, src.ProtoReflect().Get(fd))
				}
				list := v.List()
				offset := list.Len() - options.ListSizeLimit
				if options.ListSizeLimit > 0 && offset > 0 {
					for i := 0; i < options.ListSizeLimit; i++ {
						list.Set(i, list.Get(i+offset))
					}
					list.Truncate(options.ListSizeLimit)
				}
			} else if fd.IsMap() {
				if options.ShouldDeleteNilMapValue {
					dstMap := v.Map()
					srcMap := src.ProtoReflect().Get(fd).Map()
					srcMap.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
						if !v.Message().IsValid() {
							dstMap.Clear(k)
						}
						return true
					})
				}
			}
			return true
		})
	}
}
