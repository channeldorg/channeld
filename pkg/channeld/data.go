package channeld

import (
	"container/list"
	"fmt"

	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"channeld.clewcat.com/channeld/pkg/common"
	"github.com/indiest/fmutils"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

type ChannelData struct {
	mergeOptions *channeldpb.ChannelDataMergeOptions
	msg          common.ChannelDataMessage
	//updateMsg       common.ChannelDataMessage
	updateMsgBuffer     *list.List
	maxFanOutIntervalMs uint32
}

type RemovableMapField interface {
	GetRemoved() bool
}

type fanOutConnection struct {
	conn           ConnectionInChannel
	hadFirstFanOut bool
	lastFanOutTime ChannelTime
}

type updateMsgBufferElement struct {
	updateMsg   common.ChannelDataMessage
	arrivalTime ChannelTime
}

const (
	MaxUpdateMsgBufferSize = 512
)

var channelDataTypeRegistery = make(map[channeldpb.ChannelType]proto.Message)

// Register a Protobuf message template as the channel data of a specific channel type.
// This is needed when channeld doesn't know the package of the message is in,
// as well as creating a ChannelData using ReflectChannelData()
func RegisterChannelDataType(channelType channeldpb.ChannelType, msgTemplate proto.Message) {
	channelDataTypeRegistery[channelType] = msgTemplate
}

func ReflectChannelData(channelType channeldpb.ChannelType, mergeOptions *channeldpb.ChannelDataMergeOptions) (*ChannelData, error) {
	/*
		channelTypeName := channelType.String()
		dataTypeName := fmt.Sprintf("channeld.%sChannelDataMessage",
			strcase.ToCamel(strings.ToLower(channelTypeName)))
		dataType, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(dataTypeName))
		if err != nil {
			return nil, fmt.Errorf("failed to create data for channel type %s: %w", channelTypeName, err)
		}
	*/
	dataType, exists := channelDataTypeRegistery[channelType]
	if !exists {
		return nil, fmt.Errorf("failed to create data for channel type %s", channelType.String())
	}

	return &ChannelData{
		msg:             dataType.ProtoReflect().New().Interface(),
		mergeOptions:    mergeOptions,
		updateMsgBuffer: list.New(),
	}, nil
}

func (ch *Channel) InitData(dataMsg Message, mergeOptions *channeldpb.ChannelDataMergeOptions) {
	ch.data = &ChannelData{
		msg:             dataMsg,
		updateMsgBuffer: list.New(),
		mergeOptions:    mergeOptions,
	}
}

func (ch *Channel) Data() *ChannelData {
	return ch.data
}

func (ch *Channel) SetDataUpdateConnId(connId ConnectionId) {
	ch.latestDataUpdateConnId = connId
}

func (d *ChannelData) OnUpdate(updateMsg Message, t ChannelTime, spatialNotifier common.SpatialInfoChangedNotifier) {
	if d.msg == nil {
		d.msg = updateMsg
	} else {
		mergeWithOptions(d.msg, updateMsg, d.mergeOptions, spatialNotifier)
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

	focp := ch.fanOutQueue.Front()

	for foci := 0; foci < ch.fanOutQueue.Len(); foci++ {
		foc := focp.Value.(*fanOutConnection)
		conn := foc.conn
		if conn == nil || conn.IsClosing() {
			tmp := focp.Next()
			ch.fanOutQueue.Remove(focp)
			focp = tmp
			continue
		}
		cs := ch.subscribedConnections[conn]
		if cs == nil {
			focp = focp.Next()
			continue
		}

		/*
			----------------------------------------------------
			   ^                       ^                    ^
			   |------FanOutDelay------|---FanOutInterval---|
			   subTime                 firstFanOutTime      secondFanOutTime
		*/
		nextFanOutTime := foc.lastFanOutTime.AddMs(cs.options.FanOutIntervalMs)
		if t >= nextFanOutTime {

			var lastUpdateTime ChannelTime
			bufp := ch.data.updateMsgBuffer.Front()
			var accumulatedUpdateMsg common.ChannelDataMessage = nil

			//if foc.lastFanOutTime <= cs.subTime {
			if !foc.hadFirstFanOut {
				// Send the whole data for the first time
				ch.fanOutDataUpdate(conn, cs, ch.data.msg)
				foc.hadFirstFanOut = true
				// Use a hacky way to prevent the first update msg being fanned out twice (only happens when the channel doesn't have init data)
				t++
				// rootLogger.Info("conn first fan out",
				// 	zap.Uint32("connId", uint32(foc.conn.Id())),
				// 	zap.Int64("channelTime", int64(t)),
				// 	zap.Int64("nextFanOutTime", int64(nextFanOutTime)),
				// )
			} else if bufp != nil {
				if foc.lastFanOutTime >= lastUpdateTime {
					lastUpdateTime = foc.lastFanOutTime
				}

				//for be := bufp.Value.(*updateMsgBufferElement); bufp != nil && be.arrivalTime >= lastUpdateTime && be.arrivalTime <= nextFanOutTime; bufp = bufp.Next() {
				for bufi := 0; bufi < ch.data.updateMsgBuffer.Len(); bufi++ {
					be := bufp.Value.(*updateMsgBufferElement)
					if be.arrivalTime >= lastUpdateTime && be.arrivalTime <= nextFanOutTime {
						if accumulatedUpdateMsg == nil {
							accumulatedUpdateMsg = proto.Clone(be.updateMsg)
						} else {
							mergeWithOptions(accumulatedUpdateMsg, be.updateMsg, ch.data.mergeOptions, nil)
						}
						lastUpdateTime = be.arrivalTime
					}
					bufp = bufp.Next()
				}

				if accumulatedUpdateMsg != nil {
					ch.fanOutDataUpdate(conn, cs, accumulatedUpdateMsg)
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

func (ch *Channel) fanOutDataUpdate(conn ConnectionInChannel, cs *ChannelSubscription, updateMsg common.ChannelDataMessage) {
	fmutils.Filter(updateMsg, cs.options.DataFieldMasks)
	any, err := anypb.New(updateMsg)
	if err != nil {
		ch.Logger().Error("failed to marshal channel update data", zap.Error(err))
		return
	}
	conn.Send(MessageContext{
		MsgType:    channeldpb.MessageType_CHANNEL_DATA_UPDATE,
		Msg:        &channeldpb.ChannelDataUpdateMessage{Data: any},
		Connection: nil,
		Channel:    ch,
		Broadcast:  0,
		StubId:     0,
		ChannelId:  uint32(ch.id),
	})
	// cs.lastFanOutTime = time.Now()
	// cs.fanOutDataMsg = nil
}

// Implement this interface to manually merge the channel data. In most cases it can be MUCH more efficient than the default reflection-based merge.
type MergeableChannelData interface {
	Message
	Merge(src Message, options *channeldpb.ChannelDataMergeOptions, spatialNotifier common.SpatialInfoChangedNotifier) error
}

func mergeWithOptions(dst Message, src Message, options *channeldpb.ChannelDataMergeOptions, spatialNotifier common.SpatialInfoChangedNotifier) {
	mergeable, ok := dst.(MergeableChannelData)
	if ok {
		if options == nil {
			options = &channeldpb.ChannelDataMergeOptions{
				ShouldReplaceList:            false,
				ListSizeLimit:                0,
				TruncateTop:                  false,
				ShouldCheckRemovableMapField: true,
			}
		}
		if err := mergeable.Merge(src, options, spatialNotifier); err != nil {
			rootLogger.Error("custom merge error", zap.Error(err),
				zap.String("dstType", string(dst.ProtoReflect().Descriptor().FullName().Name())),
				zap.String("srcType", string(src.ProtoReflect().Descriptor().FullName().Name())),
			)
		}
	} else {
		reflectMerge(dst, src, options)
	}
}

// Use protoreflect to merge. No need to write custom merge code but less efficient.
func reflectMerge(dst Message, src Message, options *channeldpb.ChannelDataMergeOptions) {
	proto.Merge(dst, src)

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
