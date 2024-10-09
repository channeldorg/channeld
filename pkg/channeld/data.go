package channeld

import (
	"container/list"
	"fmt"
	"reflect"

	"github.com/channeldorg/channeld/pkg/channeldpb"
	"github.com/channeldorg/channeld/pkg/common"
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
	accumulatedUpdateMsg common.ChannelDataMessage
	updateMsgBuffer      *list.List
	maxFanOutIntervalMs  uint32
	msgIndex             uint64
	extension            ChannelDataExtension
}

// Indicate that the channel data message should be initialized with default values.
type ChannelDataInitializer interface {
	common.Message
	Init() error
}

type RemovableMapField interface {
	GetRemoved() bool
}

type fanOutConnection struct {
	conn             ConnectionInChannel
	hadFirstFanOut   bool
	lastFanOutTime   ChannelTime
	lastMessageIndex uint64
}

type updateMsgBufferElement struct {
	updateMsg    common.ChannelDataMessage
	arrivalTime  ChannelTime
	senderConnId ConnectionId
	messageIndex uint64
}

const (
	MaxUpdateMsgBufferSize = 512
)

var channelDataTypeRegistery = make(map[channeldpb.ChannelType]proto.Message)

// Register a Protobuf message template as the channel data of a specific channel type.
// This is needed when channeld doesn't know the package of the message is in,
// as well as creating a ChannelData using ReflectChannelData()
func RegisterChannelDataType(channelType channeldpb.ChannelType, msgTemplate proto.Message) {
	msg, exists := channelDataTypeRegistery[channelType]

	if exists {
		if rootLogger != nil {
			rootLogger.Warn("channel data type already exists, won't be registered",
				zap.String("channelType", channelType.String()),
				zap.String("curMsgName", string(msg.ProtoReflect().Descriptor().FullName())),
				zap.String("newMsgName", string(msgTemplate.ProtoReflect().Descriptor().FullName())),
			)
		}
	} else {
		channelDataTypeRegistery[channelType] = msgTemplate

		if rootLogger != nil {
			rootLogger.Info("registered channel data type",
				zap.String("channelType", channelType.String()),
				zap.String("msgFullName", string(msgTemplate.ProtoReflect().Descriptor().FullName())),
			)
		}
	}

}

func ReflectChannelDataMessage(channelType channeldpb.ChannelType) (common.ChannelDataMessage, error) {
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
		return nil, fmt.Errorf("no channel data type registered for channel type %s", channelType.String())
	}

	return dataType.ProtoReflect().New().Interface(), nil
}

func (ch *Channel) InitData(dataMsg common.ChannelDataMessage, mergeOptions *channeldpb.ChannelDataMergeOptions) {
	ch.data = &ChannelData{
		msg:             dataMsg,
		updateMsgBuffer: list.New(),
		mergeOptions:    mergeOptions,
	}

	if dataMsg == nil {
		var err error
		ch.data.msg, err = ReflectChannelDataMessage(ch.channelType)
		if err != nil {
			ch.logger.Info("unable to create default channel data message; will use the first received message to set", zap.String("chType", ch.channelType.String()), zap.Error(err))
			initChannelDataExtension(ch)
			return
		}
		ch.data.accumulatedUpdateMsg = proto.Clone(ch.data.msg)
	}

	initializer, ok := ch.data.msg.(ChannelDataInitializer)
	if ok {
		if err := initializer.Init(); err != nil {
			ch.logger.Error("failed to initialize channel data message", zap.Error(err))
			return
		}
	}

	initChannelDataExtension(ch)
}

func (ch *Channel) Data() *ChannelData {
	return ch.data
}

// CAUTION: this function is not goroutine-safe. Read/write to the channel data message should be done in the the channel's goroutine.
func (ch *Channel) GetDataMessage() common.ChannelDataMessage {
	if ch.data == nil {
		return nil
	}
	return ch.data.msg
}

func (ch *Channel) SetDataUpdateConnId(connId ConnectionId) {
	ch.latestDataUpdateConnId = connId
}

func (d *ChannelData) OnUpdate(updateMsg common.ChannelDataMessage, t ChannelTime, senderConnId ConnectionId, spatialNotifier common.SpatialInfoChangedNotifier) {
	if d.msg == nil {
		d.msg = updateMsg
		rootLogger.Info("initialized channel data with update message",
			zap.Uint32("senderConnId", uint32(senderConnId)),
			zap.String("msgName", string(updateMsg.ProtoReflect().Descriptor().FullName())),
		)
	} else {
		mergeWithOptions(d.msg, updateMsg, d.mergeOptions, spatialNotifier)
	}
	d.msgIndex = d.msgIndex + 1
	d.updateMsgBuffer.PushBack(&updateMsgBufferElement{
		updateMsg:    updateMsg,
		arrivalTime:  t,
		senderConnId: senderConnId,
		messageIndex: d.msgIndex,
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

	for focp != nil {
		foc := focp.Value.(*fanOutConnection)
		conn := foc.conn
		if conn == nil || conn.IsClosing() {
			tmp := focp.Next()
			ch.fanOutQueue.Remove(focp)
			focp = tmp
			continue
		}
		// ch.subLock.RLock()
		cs := ch.subscribedConnections[conn]
		// ch.subLock.RUnlock()
		if cs == nil || *cs.options.DataAccess == channeldpb.ChannelDataAccess_NO_ACCESS {
			focp = focp.Next()
			continue
		}

		/*
			----------------------------------------------------
			   ^                       ^                    ^
			   |------FanOutDelay------|---FanOutInterval---|
			   subTime                 firstFanOutTime      secondFanOutTime
		*/
		nextFanOutTime := foc.lastFanOutTime.AddMs(*cs.options.FanOutIntervalMs)
		// latestFanoutTime := foc.lastFanOutTime
		if t >= nextFanOutTime {
			latestFanoutTime := nextFanOutTime
			var lastUpdateTime ChannelTime
			bufp := ch.data.updateMsgBuffer.Front()
			if ch.data.accumulatedUpdateMsg == nil {
				ch.data.accumulatedUpdateMsg = ch.data.msg.ProtoReflect().New().Interface()
			} else {
				proto.Reset(ch.data.accumulatedUpdateMsg)
			}
			hasEverMerged := false

			//if foc.lastFanOutTime <= cs.subTime {
			if !foc.hadFirstFanOut {
				// Send the whole data for the first time
				ch.fanOutDataUpdate(conn, cs, ch.data.msg)
				foc.hadFirstFanOut = true
				foc.lastMessageIndex = ch.data.msgIndex
				latestFanoutTime = t
			} else if bufp != nil {
				if foc.lastFanOutTime >= lastUpdateTime {
					lastUpdateTime = foc.lastFanOutTime
				}

				for bufi := 0; bufi < ch.data.updateMsgBuffer.Len(); bufi++ {
					be := bufp.Value.(*updateMsgBufferElement)
					/*
						ch.Logger().Trace("going through updateMsgBuffer",
							zap.Int("bufi", bufi),
							zap.Int64("lastUpdateTime", int64(lastUpdateTime)/1000),
							zap.Int64("arrivalTime", int64(be.arrivalTime)/1000),
							zap.Int64("nextFanOutTime", int64(nextFanOutTime)/1000),
							zap.Uint32("senderConnId", uint32(be.senderConnId)),
						)
					*/

					if be.senderConnId == conn.Id() && *cs.options.SkipSelfUpdateFanOut {
						bufp = bufp.Next()
						continue
					}

					if be.arrivalTime >= lastUpdateTime && be.arrivalTime <= nextFanOutTime {
						if !hasEverMerged {
							proto.Merge(ch.data.accumulatedUpdateMsg, be.updateMsg)
						} else {
							mergeWithOptions(ch.data.accumulatedUpdateMsg, be.updateMsg, ch.data.mergeOptions, nil)
						}
						hasEverMerged = true
						lastUpdateTime = be.arrivalTime
						foc.lastMessageIndex = be.messageIndex
					}

					/* TODO: remove the out-dated buffer element to decrease the iteration time
					if be.arrivalTime.AddMs(ch.data.maxFanOutIntervalMs*2) < t {
						ch.data.updateMsgBuffer.Remove(bufp)
					}
					*/

					bufp = bufp.Next()
				}

				if hasEverMerged {
					ch.fanOutDataUpdate(conn, cs, ch.data.accumulatedUpdateMsg)
				}
			}
			foc.lastFanOutTime = latestFanoutTime

			temp := focp.Prev()
			// Move the fanned-out connection to the back of the queue
			for be := ch.fanOutQueue.Back(); be != nil; be = be.Prev() {
				if be.Value.(*fanOutConnection).lastFanOutTime <= foc.lastFanOutTime {
					ch.fanOutQueue.MoveAfter(focp, be)
					if temp != nil {
						focp = temp.Next()
					} else {
						focp = ch.fanOutQueue.Front()
					}

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
	/*
		conn.Logger().Trace("fan out",
			zap.Int64("channelTime", int64(ch.GetTime())),
			zap.Int64("lastFanOutTime", int64(cs.fanOutElement.Value.(*fanOutConnection).lastFanOutTime)),
			zap.Stringer("updateMsg", updateMsg.(fmt.Stringer)),
		)
	*/
	// cs.lastFanOutTime = time.Now()
	// cs.fanOutDataMsg = nil
}

// Implement this interface to manually merge the channel data. In most cases it can be MUCH more efficient than the default reflection-based merge.
type MergeableChannelData interface {
	common.Message
	Merge(src common.ChannelDataMessage, options *channeldpb.ChannelDataMergeOptions, spatialNotifier common.SpatialInfoChangedNotifier) error
}

func mergeWithOptions(dst common.ChannelDataMessage, src common.ChannelDataMessage, options *channeldpb.ChannelDataMergeOptions, spatialNotifier common.SpatialInfoChangedNotifier) {
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
		ReflectMerge(dst, src, options)
	}
}

// Use protoreflect to merge. No need to write custom merge code but less efficient.
func ReflectMerge(dst common.ChannelDataMessage, src common.ChannelDataMessage, options *channeldpb.ChannelDataMergeOptions) {
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

type ChannelDataExtension interface {
	Init(ch *Channel)
	GetRecoveryDataMessage() common.Message
}

func initChannelDataExtension(ch *Channel) {
	extType, exists := channelDataExtensionRegistery[ch.channelType]
	if exists {
		ch.data.extension = reflect.New(extType).Interface().(ChannelDataExtension)
		ch.data.extension.Init(ch)
	}
}

func (d *ChannelData) Extension() ChannelDataExtension {
	return d.extension
}

var channelDataExtensionRegistery = make(map[channeldpb.ChannelType]reflect.Type)

func SetChannelDataExtension[T any, PT interface {
	*T
	ChannelDataExtension
}](chType channeldpb.ChannelType) {
	var zero [0]T
	extType := reflect.TypeOf(zero).Elem()
	channelDataExtensionRegistery[chType] = extType
}
