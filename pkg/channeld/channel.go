package channeld

import (
	"container/list"
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"channeld.clewcat.com/channeld/pkg/common"
	"go.uber.org/zap"
)

type ChannelState uint8

const (
	INIT     ChannelState = 0
	OPEN     ChannelState = 1
	HANDOVER ChannelState = 2
)

// Each channel uses a goroutine and we can have at most millions of goroutines at the same time.
// So we won't use 64-bit channel ID unless we use a distributed architecture for channeld itself.
type ChannelId uint32

// ChannelTime is the relative time since the channel created.
type ChannelTime int64 // time.Duration

func (t ChannelTime) AddMs(ms uint32) ChannelTime {
	return t + ChannelTime(ms*uint32(time.Millisecond))
}

func (t ChannelTime) OffsetMs(ms int32) ChannelTime {
	return t + ChannelTime(ms*int32(time.Millisecond))
}

type channelMessage struct {
	ctx     MessageContext
	handler MessageHandlerFunc
}

// Use this interface instead of Connection for protecting the connection from unsafe writing in the channel goroutine.
type ConnectionInChannel interface {
	Id() ConnectionId
	GetConnectionType() channeldpb.ConnectionType
	OnAuthenticated()
	HasAuthorityOver(ch *Channel) bool
	IsRemoving() bool
	Send(ctx MessageContext)
	SubscribeToChannel(ch *Channel, options *channeldpb.ChannelSubscriptionOptions) *ChannelSubscription
	UnsubscribeFromChannel(ch *Channel) (*channeldpb.ChannelSubscriptionOptions, error)
	sendSubscribed(ctx MessageContext, ch *Channel, connToSub ConnectionInChannel, stubId uint32, subOptions *channeldpb.ChannelSubscriptionOptions)
	sendUnsubscribed(ctx MessageContext, ch *Channel, connToUnsub *Connection, stubId uint32)
	Logger() *Logger
}

type Channel struct {
	id                    ChannelId
	channelType           channeldpb.ChannelType
	state                 ChannelState
	ownerConnection       ConnectionInChannel
	subscribedConnections map[ConnectionInChannel]*ChannelSubscription
	connectionsLock       sync.RWMutex
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
	GlobalChannelId ChannelId = 0
)

var nextChannelId ChannelId
var nextSpatialChannelId ChannelId

// Cache the status so we don't have to check all the index in the sync map, until a channel is removed.
var nonSpatialchannelFull bool = false
var spatialChannelFull bool = false

var allChannels sync.Map //map[ChannelId]*Channel
var globalChannel *Channel

func InitChannels() {
	nextChannelId = 0
	nextSpatialChannelId = GlobalSettings.SpatialChannelIdStart
	globalChannel, _ = CreateChannel(channeldpb.ChannelType_GLOBAL, nil)
}

func GetChannel(id ChannelId) *Channel {
	ch, ok := allChannels.Load(id)
	if ok {
		return ch.(*Channel)
	} else {
		return nil
	}
}

var ErrNonSpatialChannelFull = errors.New("non-spatial channels are full")
var ErrSpatialChannelFull = errors.New("spatial channels are full")

func createChannelWithId(channelId ChannelId, t channeldpb.ChannelType, owner ConnectionInChannel) *Channel {
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

	// TODO: check state if tick()
	if ch.HasOwner() {
		ch.state = OPEN
	} else {
		ch.state = INIT
	}

	allChannels.Store(ch.id, ch)
	go ch.Tick()

	channelNum.WithLabelValues(ch.channelType.String()).Inc()

	return ch
}

func CreateChannel(t channeldpb.ChannelType, owner ConnectionInChannel) (*Channel, error) {
	if t == channeldpb.ChannelType_GLOBAL && globalChannel != nil {
		return nil, errors.New("failed to create WORLD channel as it already exists")
	}

	var channelId ChannelId
	var ok bool
	if t != channeldpb.ChannelType_SPATIAL {
		if nonSpatialchannelFull {
			return nil, ErrNonSpatialChannelFull
		}
		channelId, ok = GetNextIdSync(&allChannels, nextChannelId, 1, GlobalSettings.SpatialChannelIdStart-1)
		if ok {
			nextChannelId = channelId
		} else {
			nonSpatialchannelFull = true
			return nil, ErrNonSpatialChannelFull
		}
	} else {
		if spatialChannelFull {
			return nil, ErrSpatialChannelFull
		}
		channelId, ok = GetNextIdSync(&allChannels, nextSpatialChannelId, GlobalSettings.SpatialChannelIdStart, math.MaxUint32)
		if ok {
			nextSpatialChannelId = channelId
		} else {
			spatialChannelFull = true
			return nil, ErrSpatialChannelFull
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
	} else {
		nonSpatialchannelFull = false
		nextChannelId = ch.id
	}

	channelNum.WithLabelValues(ch.channelType.String()).Dec()
}

func (ch *Channel) IsRemoving() bool {
	return ch.removing > 0
}

func (ch *Channel) PutMessage(msg Message, handler MessageHandlerFunc, conn *Connection, pack *channeldpb.MessagePack) {
	if ch.IsRemoving() {
		return
	}
	ch.inMsgQueue <- channelMessage{ctx: MessageContext{
		MsgType:    channeldpb.MessageType(pack.MsgType),
		Msg:        msg,
		Connection: conn,
		Channel:    ch,
		Broadcast:  pack.Broadcast,
		StubId:     pack.StubId,
		ChannelId:  pack.ChannelId,
	}, handler: handler}
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

	// Tick connections
	/*
		if ch.ownerConnection != nil {
			if ch.ownerConnection.IsRemoving() {
				ch.ownerConnection = nil
			}
		}
	*/
	for conn := range ch.subscribedConnections {
		if conn.IsRemoving() {
			// Unsub the connection from the channel
			delete(ch.subscribedConnections, conn)
			conn.Logger().Info("removed subscription of a disconnected endpoint", zap.Uint32("channelId", uint32(ch.id)))
			if ownerConn, ok := ch.ownerConnection.(*Connection); ok && conn != nil {
				if ownerConn == conn {
					// Reset the owner if it's removed
					ch.ownerConnection = nil
					conn.Logger().Info("found removed ownner connection of channel", zap.Uint32("channelId", uint32(ch.id)))
					if GlobalSettings.GetChannelSettings(ch.channelType).RemoveChannelAfterOwnerRemoved {
						// TODO: send RemoveChannelMessage to all subscribed connections
						RemoveChannel(ch)
						ch.Logger().Info("removed channel after the owner is removed")
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
	return ok && conn != nil && !conn.IsRemoving()
}

// Implementation for ConnectionInChannel interface
func (c *Connection) IsNil() bool {
	return c == nil
}
