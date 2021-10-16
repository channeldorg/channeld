package channeld

import (
	"container/list"
	"fmt"
	"log"
	"strings"

	"channeld.clewcat.com/channeld/proto"
	"github.com/iancoleman/strcase"
	"github.com/indiest/fmutils"
	protobuf "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/anypb"
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
	msg          ChannelDataMessage
	//updateMsg       ChannelDataMessage
	updateMsgBuffer     *list.List
	maxFanOutIntervalMs uint32
}

type fanOutConnection struct {
	connId         ConnectionId
	lastFanOutTime ChannelTime
}

type updateMsgBufferElement struct {
	updateMsg   ChannelDataMessage
	arrivalTime ChannelTime
}

const (
	MaxUpdateMsgBufferSize = 512
)

func ReflectChannelData(channelType proto.ChannelType, mergeOptions *DataMergeOptions) *ChannelData {
	channelTypeName := channelType.String()
	dataTypeName := fmt.Sprintf("channeld.%sChannelDataMessage",
		strcase.ToCamel(strings.ToLower(channelTypeName)))
	dataType, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(dataTypeName))
	if err != nil {
		log.Panicln("Failed to create data for channel type", channelTypeName)
	}
	/*
		msgTypeName := "CHANNEL_DATA_" + strings.ToUpper(channelTypeName)
		msgType, exists := proto.MessageType_value[msgTypeName]
		if !exists {
			log.Panicln("Can't find data update message type by name", msgTypeName)
		}
	*/
	return &ChannelData{
		//msgType:         proto.MessageType(msgType),
		msg:             dataType.New().Interface(),
		mergeOptions:    mergeOptions,
		updateMsgBuffer: list.New(),
	}
}

func (ch *Channel) InitData(dataMsg Message, mergeOptions *DataMergeOptions) {
	ch.data = &ChannelData{
		msg:             dataMsg,
		updateMsgBuffer: list.New(),
		mergeOptions:    mergeOptions,
	}
}

func (ch *Channel) Data() *ChannelData {
	return ch.data
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
		updateMsg:   updateMsg,
		arrivalTime: t,
	})
	if d.updateMsgBuffer.Len() > MaxUpdateMsgBufferSize {
		oldest := d.updateMsgBuffer.Front()
		// Remove the oldest update message if it should has been fanned-out
		if oldest.Value.(*updateMsgBufferElement).arrivalTime.AddMs(d.maxFanOutIntervalMs) < t {
			d.updateMsgBuffer.Remove(oldest)
		}
	}
}

func (ch *Channel) tickData(t ChannelTime) {
	if ch.data == nil || ch.data.msg == nil {
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
	focp := ch.fanOutQueue.Front()

	for foci := 0; foci < ch.fanOutQueue.Len(); foci++ {
		foc := focp.Value.(*fanOutConnection)
		c := GetConnection(foc.connId)
		if c == nil {
			focp = focp.Next()
			continue
		}
		cs := ch.subscribedConnections[foc.connId]
		if cs == nil {
			focp = focp.Next()
			continue
		}

		nextFanOutTime := foc.lastFanOutTime.AddMs(cs.options.FanOutIntervalMs)
		if t >= nextFanOutTime {

			var lastUpdateTime ChannelTime
			bufp := ch.data.updateMsgBuffer.Front()
			var accumulatedUpdateMsg ChannelDataMessage = nil

			if foc.lastFanOutTime == 0 {
				// Send the whole data for the first time
				ch.fanOutDataUpdate(c, cs, ch.data.msg)
			} else if bufp != nil {
				if foc.lastFanOutTime >= lastUpdateTime {
					lastUpdateTime = foc.lastFanOutTime
				}

				//for be := bufp.Value.(*updateMsgBufferElement); bufp != nil && be.arrivalTime >= lastUpdateTime && be.arrivalTime <= nextFanOutTime; bufp = bufp.Next() {
				for bufi := 0; bufi < ch.data.updateMsgBuffer.Len(); bufi++ {
					be := bufp.Value.(*updateMsgBufferElement)
					if be.arrivalTime >= lastUpdateTime && be.arrivalTime <= nextFanOutTime {
						if accumulatedUpdateMsg == nil {
							accumulatedUpdateMsg = protobuf.Clone(be.updateMsg)
						} else {
							mergeWithOptions(accumulatedUpdateMsg, be.updateMsg, ch.data.mergeOptions)
						}
						lastUpdateTime = be.arrivalTime
					}
					bufp = bufp.Next()
				}

				if accumulatedUpdateMsg != nil {
					ch.fanOutDataUpdate(c, cs, accumulatedUpdateMsg)
				}
			}

			foc.lastFanOutTime = t

			temp := focp.Next()
			// Move the fanned-out connection to the back of the queue
			for be := ch.fanOutQueue.Back(); be != nil; be = be.Prev() {
				if be.Value.(*fanOutConnection).lastFanOutTime <= foc.lastFanOutTime {
					ch.fanOutQueue.MoveAfter(focp, be)
					focp = temp
					break
				}
			}
		} else {
			focp = focp.Next()
		}
	}
}

func (ch *Channel) fanOutDataUpdate(c *Connection, cs *ChannelSubscription, updateMsg ChannelDataMessage) {
	fmutils.Filter(updateMsg, cs.options.DataFieldMasks)
	any, err := anypb.New(updateMsg)
	if err != nil {
		log.Panicln(err)
	}
	c.Send(ch.id, proto.MessageType_CHANNEL_DATA_UPDATE, &proto.ChannelDataUpdateMessage{
		Data: any,
	})
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
