package channeld

import (
	"container/list"
	"fmt"
	"strings"

	"channeld.clewcat.com/channeld/proto"
	"github.com/iancoleman/strcase"
	"github.com/indiest/fmutils"
	"go.uber.org/zap"
	protobuf "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/anypb"
)

type ChannelDataMessage = Message //protoreflect.Message

type ChannelData struct {
	mergeOptions *proto.ChannelDataMergeOptions
	msg          ChannelDataMessage
	//updateMsg       ChannelDataMessage
	updateMsgBuffer     *list.List
	maxFanOutIntervalMs uint32
}

type RemovableMapField interface {
	GetRemoved() bool
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

func ReflectChannelData(channelType proto.ChannelType, mergeOptions *proto.ChannelDataMergeOptions) (*ChannelData, error) {
	channelTypeName := channelType.String()
	dataTypeName := fmt.Sprintf("channeld.%sChannelDataMessage",
		strcase.ToCamel(strings.ToLower(channelTypeName)))
	dataType, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(dataTypeName))
	if err != nil {
		return nil, fmt.Errorf("failed to create data for channel type %s: %w", channelTypeName, err)
	}

	return &ChannelData{
		msg:             dataType.New().Interface(),
		mergeOptions:    mergeOptions,
		updateMsgBuffer: list.New(),
	}, nil
}

func (ch *Channel) InitData(dataMsg Message, mergeOptions *proto.ChannelDataMergeOptions) {
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
	if d.msg == nil {
		d.msg = updateMsg
	} else {
		mergeWithOptions(d.msg, updateMsg, d.mergeOptions)
	}

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
		if c == nil || c.IsRemoving() {
			tmp := focp.Next()
			ch.fanOutQueue.Remove(focp)
			focp = tmp
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
		ch.Logger().Error("failed to marshal channel update data", zap.Error(err))
		return
	}
	c.Send(MessageContext{
		MsgType:    proto.MessageType_CHANNEL_DATA_UPDATE,
		Msg:        &proto.ChannelDataUpdateMessage{Data: any},
		Connection: nil,
		Channel:    ch,
		Broadcast:  proto.BroadcastType_NO_BROADCAST,
		StubId:     0,
		ChannelId:  uint32(ch.id),
	})
	// cs.lastFanOutTime = time.Now()
	// cs.fanOutDataMsg = nil
}

/* Experiment: extending protobuf to make the merge more efficient
func postMergeMapKV(dst, src protoreflect.Map, k protoreflect.MapKey, v protoreflect.Value, fd protoreflect.FieldDescriptor) bool {
	removable, ok := v.Message().Interface().(removableMapField)
	if ok && removable.GetRemoved() {
		dst.Clear(k)
	}
	return true
}
*/

type MergeableChannelData interface {
	Message
	Merge(src Message, options *proto.ChannelDataMergeOptions) error
}

func mergeWithOptions(dst Message, src Message, options *proto.ChannelDataMergeOptions) {
	mergeable, ok := dst.(MergeableChannelData)
	if ok {
		if options == nil {
			options = &proto.ChannelDataMergeOptions{
				ShouldReplaceList:            false,
				ListSizeLimit:                0,
				TruncateTop:                  false,
				ShouldCheckRemovableMapField: true,
			}
		}
		if err := mergeable.Merge(src, options); err != nil {
			logger.Error("custom merge error", zap.Error(err),
				zap.String("dstType", string(dst.ProtoReflect().Descriptor().FullName().Name())),
				zap.String("srcType", string(src.ProtoReflect().Descriptor().FullName().Name())),
			)
		}
	} else {
		reflectMerge(dst, src, options)
	}
}

// Use protoreflect to merge. No need to write custom merge code but less efficient.
func reflectMerge(dst Message, src Message, options *proto.ChannelDataMergeOptions) {
	// if options == nil {
	protobuf.Merge(dst, src)
	// } else {
	// 	protobuf.MergeWithOptions(dst, src, protobuf.MergeOptions{PostMergeMapKV: postMergeMapKV})
	// }

	if options != nil {
		//logger.Debug("merged with options", zap.Any("src", src), zap.Any("dst", dst))
		defer func() {
			recover()
		}()

		dst.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			if fd.IsList() {
				if options.ShouldReplaceList {
					dst.ProtoReflect().Set(fd, src.ProtoReflect().Get(fd))
				}
				list := v.List()
				offset := list.Len() - int(options.ListSizeLimit)
				if options.ListSizeLimit > 0 && offset > 0 {
					if options.TruncateTop {
						for i := 0; i < int(options.ListSizeLimit); i++ {
							list.Set(i, list.Get(i+offset))
						}
					}
					list.Truncate(int(options.ListSizeLimit))
				}
			} else if fd.IsMap() {
				if options.ShouldCheckRemovableMapField {
					dstMap := v.Map()
					dstMap.Range(func(mk protoreflect.MapKey, mv protoreflect.Value) bool {
						removable, ok := mv.Message().Interface().(RemovableMapField)
						if ok && removable.GetRemoved() {
							dstMap.Clear(mk)
						}
						return true
					})
				}
			}
			return true
		})
	}
}
