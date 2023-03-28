package channeld

import (
	"container/list"
	"errors"
	"fmt"
	"hash/maphash"
	"math"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/metaworking/channeld/pkg/channeldpb"
	"github.com/metaworking/channeld/pkg/common"
	"github.com/puzpuzpuz/xsync/v2"
	"go.uber.org/zap"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type ChannelState uint8

const (
	ChannelState_INIT     ChannelState = 0
	ChannelState_OPEN     ChannelState = 1
	ChannelState_HANDOVER ChannelState = 2
)

// ChannelTime is the relative time since the channel created.
type ChannelTime int64 // time.Duration

func (t ChannelTime) AddMs(ms uint32) ChannelTime {
	return t + ChannelTime(time.Duration(ms)*time.Millisecond)
}

func (t ChannelTime) OffsetMs(ms int32) ChannelTime {
	return t + ChannelTime(time.Duration(ms)*time.Millisecond)
}

type channelMessage struct {
	ctx     MessageContext
	handler MessageHandlerFunc
}

// Use this interface instead of Connection for protecting the connection from unsafe writing in the channel goroutine.
type ConnectionInChannel interface {
	Id() ConnectionId
	GetConnectionType() channeldpb.ConnectionType
	OnAuthenticated(pit string)
	HasAuthorityOver(ch *Channel) bool
	Close()
	IsClosing() bool
	Send(ctx MessageContext)
	SubscribeToChannel(ch *Channel, options *channeldpb.ChannelSubscriptionOptions) *ChannelSubscription
	UnsubscribeFromChannel(ch *Channel) (*channeldpb.ChannelSubscriptionOptions, error)
	sendSubscribed(ctx MessageContext, ch *Channel, connToSub ConnectionInChannel, stubId uint32, subOptions *channeldpb.ChannelSubscriptionOptions)
	sendUnsubscribed(ctx MessageContext, ch *Channel, connToUnsub *Connection, stubId uint32)
	Logger() *Logger
	RemoteAddr() net.Addr
}

type Channel struct {
	id                    common.ChannelId
	channelType           channeldpb.ChannelType
	state                 ChannelState
	ownerConnection       ConnectionInChannel
	subscribedConnections map[ConnectionInChannel]*ChannelSubscription
	// Lock for reading all the subscribed connection outside the channel
	connectionsLock sync.RWMutex
	// Read-only property, e.g. name
	metadata string
	data     *ChannelData
	// The ID of the client connection that causes the latest ChannelDataUpdate
	latestDataUpdateConnId ConnectionId
	spatialNotifier        common.SpatialInfoChangedNotifier
	inMsgQueue             chan channelMessage
	fanOutQueue            *list.List
	// Time since channel created
	startTime             time.Time
	tickInterval          time.Duration
	tickFrames            int
	enableClientBroadcast bool
	logger                *Logger
	removing              int32
}

const (
	GlobalChannelId common.ChannelId = 0
)

var nextChannelId common.ChannelId
var nextSpatialChannelId common.ChannelId
var nextEntityChannelId common.ChannelId

// Cache the status so we don't have to check all the index in the sync map, until a channel is removed.
var nonSpatialChannelFull bool = false
var spatialChannelFull bool = false
var entityChannelFull bool = false

var allChannels *xsync.MapOf[common.ChannelId, *Channel]
var globalChannel *Channel

func InitChannels() {
	if allChannels != nil {
		return
	}

	allChannels = xsync.NewTypedMapOf[common.ChannelId, *Channel](func(s maphash.Seed, chId common.ChannelId) uint64 {
		return uint64(chId)
	})

	nextChannelId = 0
	nextSpatialChannelId = GlobalSettings.SpatialChannelIdStart
	nextEntityChannelId = GlobalSettings.EntityChannelIdStart
	var err error
	globalChannel, err = CreateChannel(channeldpb.ChannelType_GLOBAL, nil)
	if err != nil {
		rootLogger.Panic("Failed to create global channel", zap.Error(err))
	}

	for chType, settings := range GlobalSettings.ChannelSettings {
		if settings.DataMsgFullName == "" {
			continue
		}

		msgType, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(settings.DataMsgFullName))
		if err != nil {
			rootLogger.Error("failed to find message type for channel data",
				zap.String("channelType", chType.String()),
				zap.String("msgFullName", settings.DataMsgFullName),
				zap.Error(err),
			)
			continue
		}

		RegisterChannelDataType(chType, msgType.New().Interface())
	}
}

func GetChannel(id common.ChannelId) *Channel {
	ch, ok := allChannels.Load(id)
	if ok {
		return ch
	} else {
		return nil
	}
}

var ErrNonSpatialChannelFull = errors.New("non-spatial channels are full")
var ErrSpatialChannelFull = errors.New("spatial channels are full")
var ErrEntityChannelFull = errors.New("entity channels are full")

func createChannelWithId(channelId common.ChannelId, t channeldpb.ChannelType, owner ConnectionInChannel) *Channel {
	ch := &Channel{
		id:                    channelId,
		channelType:           t,
		ownerConnection:       owner,
		subscribedConnections: make(map[ConnectionInChannel]*ChannelSubscription),
		connectionsLock:       sync.RWMutex{},
		/* Channel data is not created by default. See handleCreateChannel().
		data:                  ReflectChannelData(t, nil),
		*/
		inMsgQueue:   make(chan channelMessage, 1024),
		fanOutQueue:  list.New(),
		startTime:    time.Now(),
		tickInterval: time.Duration(GlobalSettings.GetChannelSettings(t).TickIntervalMs) * time.Millisecond,
		tickFrames:   0,
		logger: &Logger{rootLogger.With(
			zap.String("channelType", t.String()),
			zap.Uint32("channelId", uint32(channelId)),
		)},
		removing: 0,
	}

	if ch.channelType == channeldpb.ChannelType_SPATIAL {
		ch.spatialNotifier = spatialController
	}

	if ch.HasOwner() {
		ch.state = ChannelState_OPEN
	} else {
		ch.state = ChannelState_INIT
	}

	allChannels.Store(ch.id, ch)
	go ch.Tick()

	channelNum.WithLabelValues(ch.channelType.String()).Inc()

	Event_ChannelCreated.Broadcast(ch)
	return ch
}

// Go-routine safe - should only be called in the GLOBAL channel
func CreateChannel(t channeldpb.ChannelType, owner ConnectionInChannel) (*Channel, error) {
	if t == channeldpb.ChannelType_GLOBAL && globalChannel != nil {
		return nil, errors.New("failed to create GLOBAL channel as it already exists")
	}

	var channelId common.ChannelId
	var ok bool
	if t == channeldpb.ChannelType_SPATIAL {
		if spatialChannelFull {
			return nil, ErrSpatialChannelFull
		}
		channelId, ok = GetNextIdTyped[common.ChannelId, *Channel](allChannels, nextSpatialChannelId, GlobalSettings.SpatialChannelIdStart, GlobalSettings.EntityChannelIdStart-1)
		if ok {
			nextSpatialChannelId = channelId
		} else {
			spatialChannelFull = true
			return nil, ErrSpatialChannelFull
		}
	} else if t == channeldpb.ChannelType_ENTITY {
		if entityChannelFull {
			return nil, ErrEntityChannelFull
		}
		channelId, ok = GetNextIdTyped[common.ChannelId, *Channel](allChannels, nextEntityChannelId, GlobalSettings.EntityChannelIdStart, math.MaxUint32)
		if ok {
			nextEntityChannelId = channelId
		} else {
			entityChannelFull = true
			return nil, ErrEntityChannelFull
		}
	} else {
		if nonSpatialChannelFull {
			return nil, ErrNonSpatialChannelFull
		}
		channelId, ok = GetNextIdTyped[common.ChannelId, *Channel](allChannels, nextChannelId, 1, GlobalSettings.SpatialChannelIdStart-1)
		if ok {
			nextChannelId = channelId
		} else {
			nonSpatialChannelFull = true
			return nil, ErrNonSpatialChannelFull
		}
	}

	return createChannelWithId(channelId, t, owner), nil
}

func RemoveChannel(ch *Channel) {
	atomic.AddInt32(&ch.removing, 1)
	close(ch.inMsgQueue)
	allChannels.Delete(ch.id)
	// Reset the channel full status cache
	if ch.channelType == channeldpb.ChannelType_SPATIAL {
		spatialChannelFull = false
		nextSpatialChannelId = ch.id
	} else if ch.channelType == channeldpb.ChannelType_ENTITY {
		entityChannelFull = false
		nextEntityChannelId = ch.id
	} else {
		nonSpatialChannelFull = false
		nextChannelId = ch.id
	}

	channelNum.WithLabelValues(ch.channelType.String()).Dec()

	Event_ChannelRemoved.Broadcast(ch.id)
}

func (ch *Channel) IsRemoving() bool {
	return ch.removing > 0
}

func (ch *Channel) PutMessage(msg common.Message, handler MessageHandlerFunc, conn *Connection, pack *channeldpb.MessagePack) {
	if ch.IsRemoving() {
		return
	}
	ch.inMsgQueue <- channelMessage{ctx: MessageContext{
		MsgType:     channeldpb.MessageType(pack.MsgType),
		Msg:         msg,
		Connection:  conn,
		Channel:     ch,
		Broadcast:   pack.Broadcast,
		StubId:      pack.StubId,
		ChannelId:   pack.ChannelId,
		arrivalTime: ch.GetTime(),
	}, handler: handler}
}

func (ch *Channel) PutMessageContext(ctx MessageContext, handler MessageHandlerFunc) {
	if ch.IsRemoving() {
		return
	}
	ch.inMsgQueue <- channelMessage{ctx: ctx, handler: handler}
}

// Put the message into the channel's message queue. This method is used internally to make sure the message is handled in the channel's goroutine, to avoid race condition.
//
// For the MessageContext, the Connection is set as the channel's ownerConnection, and the ChannelId is set as the channel's id.
func (ch *Channel) PutMessageInternal(msgType channeldpb.MessageType, msg common.Message) {
	if ch.IsRemoving() {
		return
	}

	entry, exists := MessageMap[msgType]
	if !exists {
		ch.logger.Error("can't find message handler", zap.String("msgType", msgType.String()))
		return
	}

	ch.inMsgQueue <- channelMessage{ctx: MessageContext{
		MsgType:     msgType,
		Msg:         msg,
		Connection:  ch.ownerConnection,
		Channel:     ch,
		Broadcast:   0,
		StubId:      0,
		ChannelId:   uint32(ch.id),
		arrivalTime: ch.GetTime(),
	}, handler: entry.handler}
}

func (ch *Channel) GetTime() ChannelTime {
	return ChannelTime(time.Since(ch.startTime))
}

func (ch *Channel) Tick() {
	for {
		if ch.IsRemoving() {
			return
		}

		// Run the code of SpatialController only in GLOBAL channel, to avoid any race condition.
		if ch.channelType == channeldpb.ChannelType_GLOBAL && spatialController != nil {
			spatialController.Tick()
		}

		tickStart := time.Now()
		ch.tickFrames++

		ch.tickMessages(tickStart)

		ch.tickData(ch.GetTime())

		ch.tickConnections()

		tickDuration := time.Since(tickStart)
		channelTickDuration.WithLabelValues(ch.channelType.String()).Set(float64(tickDuration) / float64(time.Millisecond))

		time.Sleep(ch.tickInterval - tickDuration)
	}
}

func (ch *Channel) tickMessages(tickStart time.Time) {
	for len(ch.inMsgQueue) > 0 {
		cm := <-ch.inMsgQueue
		if cm.ctx.Connection == nil {
			ch.Logger().Warn("drops message as the sender is lost", zap.Uint32("msgType", uint32(cm.ctx.MsgType)))
			continue
		}
		cm.handler(cm.ctx)
		if ch.tickInterval > 0 && time.Since(tickStart) >= ch.tickInterval {
			ch.Logger().Warn("spent too long handling messages, will delay the left to the next tick",
				zap.Duration("duration", time.Since(tickStart)),
				zap.Int("remaining", len(ch.inMsgQueue)),
			)
			break
		}
	}
}

func (ch *Channel) tickConnections() {
	defer func() {
		ch.connectionsLock.RUnlock()
	}()
	ch.connectionsLock.RLock()

	for conn := range ch.subscribedConnections {
		if conn.IsClosing() {
			// Unsub the connection from the channel
			delete(ch.subscribedConnections, conn)
			conn.Logger().Info("removed subscription of a disconnected endpoint", zap.Uint32("channelId", uint32(ch.id)))
			if ownerConn, ok := ch.ownerConnection.(*Connection); ok && conn != nil {
				if ownerConn == conn {
					// Reset the owner if it's removed
					ch.ownerConnection = nil
					if ch.channelType == channeldpb.ChannelType_GLOBAL {
						Event_GlobalChannelUnpossessed.Broadcast(struct{}{})
					}
					conn.Logger().Info("found removed ownner connection of channel", zap.Uint32("channelId", uint32(ch.id)))
					if GlobalSettings.GetChannelSettings(ch.channelType).RemoveChannelAfterOwnerRemoved {
						atomic.AddInt32(&ch.removing, 1)
						/* Let the GLOBAL channel handles the channel remove
						// Send RemoveChannelMessage to all subscribed connections
						ch.Broadcast(MessageContext{
							MsgType: channeldpb.MessageType_REMOVE_CHANNEL,
							Msg: &channeldpb.RemoveChannelMessage{
								ChannelId: uint32(ch.id),
							},
							Broadcast: uint32(channeldpb.BroadcastType_ALL_BUT_OWNER),
							StubId:    0,
							ChannelId: uint32(ch.id),
						})
						RemoveChannel(ch)
						*/
						globalChannel.PutMessage(&channeldpb.RemoveChannelMessage{
							ChannelId: uint32(ch.id),
						}, handleRemoveChannel, nil, &channeldpb.MessagePack{
							Broadcast: 0,
							StubId:    0,
							ChannelId: uint32(GlobalChannelId),
						})

						ch.Logger().Info("removing channel after the owner is removed")
						return
					}
				} else if conn != nil {
					ch.ownerConnection.sendUnsubscribed(MessageContext{}, ch, conn.(*Connection), 0)
				}
			}
		}
	}
}

func (ch *Channel) Broadcast(ctx MessageContext) {
	defer func() {
		ch.connectionsLock.RUnlock()
	}()
	ch.connectionsLock.RLock()

	for conn := range ch.subscribedConnections {
		//c := GetConnection(connId)
		if conn == nil {
			continue
		}
		if channeldpb.BroadcastType_ALL_BUT_SENDER.Check(ctx.Broadcast) && conn == ctx.Connection {
			continue
		}
		if channeldpb.BroadcastType_ALL_BUT_OWNER.Check(ctx.Broadcast) && conn == ch.ownerConnection {
			continue
		}
		if channeldpb.BroadcastType_ALL_BUT_CLIENT.Check(ctx.Broadcast) && conn.GetConnectionType() == channeldpb.ConnectionType_CLIENT {
			continue
		}
		if channeldpb.BroadcastType_ALL_BUT_SERVER.Check(ctx.Broadcast) && conn.GetConnectionType() == channeldpb.ConnectionType_SERVER {
			continue
		}
		conn.Send(ctx)
	}
}

// Goroutine-safe read of the subscribed connections
func (ch *Channel) GetAllConnections() map[ConnectionInChannel]struct{} {
	defer func() {
		ch.connectionsLock.RUnlock()
	}()
	ch.connectionsLock.RLock()

	conns := make(map[ConnectionInChannel]struct{})
	for conn := range ch.subscribedConnections {
		conns[conn] = struct{}{}
	}
	return conns
}

// Return true if the connection can 1)remove; 2)sub/unsub another connection to/from; the channel.
func (c *Connection) HasAuthorityOver(ch *Channel) bool {
	// The global owner has authority over everything.
	if globalChannel.ownerConnection == c {
		return true
	}
	if ch.ownerConnection == c {
		return true
	}
	return false
}

func (ch *Channel) String() string {
	return fmt.Sprintf("Channel(%s %d)", ch.channelType.String(), ch.id)
}

func (ch *Channel) Logger() *Logger {
	return ch.logger
}

func (ch *Channel) HasOwner() bool {
	conn, ok := ch.ownerConnection.(*Connection)
	return ok && conn != nil && !conn.IsClosing()
}

func (chA *Channel) IsSameOwner(chB *Channel) bool {
	return chA.HasOwner() && chB.HasOwner() && chA.ownerConnection == chB.ownerConnection
}

func (ch *Channel) SendToOwner(msgType uint32, msg common.Message) {
	if !ch.HasOwner() {
		ch.Logger().Warn("channel has no owner to send message", zap.Uint32("msgType", msgType))
		return
	}
	ch.ownerConnection.Send(MessageContext{
		MsgType:   channeldpb.MessageType(msgType),
		Msg:       msg,
		ChannelId: uint32(ch.id),
		Broadcast: 0,
		StubId:    0,
	})
}

// Implementation for ConnectionInChannel interface
func (c *Connection) IsNil() bool {
	return c == nil
}
