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
	return t + ChannelTime(uint32(time.Millisecond)*ms)
}

type channelMessage struct {
	ctx     MessageContext
	handler MessageHandlerFunc
}

// Use this interface instead of Connection for protecting the connection from writing in the channel goroutine.
type ConnectionInChannel interface {
	Id() ConnectionId
	IsRemoving() bool
	Send(ctx MessageContext)
	sendSubscribed(ctx MessageContext, ch *Channel, connToSub *Connection, stubId uint32, subOptions *channeldpb.ChannelSubscriptionOptions)
	sendUnsubscribed(ctx MessageContext, ch *Channel, connToUnsub *Connection, stubId uint32)
	Logger() *zap.Logger
	IsNil() bool
}

type Channel struct {
	id                    ChannelId
	channelType           channeldpb.ChannelType
	state                 ChannelState
	ownerConnection       ConnectionInChannel
	subscribedConnections map[ConnectionInChannel]*ChannelSubscription
	metadata              string // Read-only property, e.g. name
	data                  *ChannelData
	spatialNotifier       SpatialInfoChangedNotifier
	inMsgQueue            chan channelMessage
	fanOutQueue           *list.List
	startTime             time.Time // Time since channel created
	tickInterval          time.Duration
	tickFrames            int
	enableClientBroadcast bool
	logger                *zap.Logger
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

func CreateChannel(t channeldpb.ChannelType, owner *Connection) (*Channel, error) {
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

	ch := &Channel{
		id:                    channelId,
		channelType:           t,
		ownerConnection:       owner,
		subscribedConnections: make(map[ConnectionInChannel]*ChannelSubscription),
		/* Channel data is not created by default. See handleCreateChannel().
		data:                  ReflectChannelData(t, nil),
		*/
		inMsgQueue:   make(chan channelMessage, 1024),
		fanOutQueue:  list.New(),
		startTime:    time.Now(),
		tickInterval: time.Duration(GlobalSettings.GetChannelSettings(t).TickIntervalMs) * time.Millisecond,
		tickFrames:   0,
		logger: logger.With(
			zap.String("channelType", t.String()),
			zap.Uint32("channelId", uint32(nextChannelId)),
		),
		removing: 0,
	}
	if ch.channelType == channeldpb.ChannelType_SPATIAL {
		ch.spatialNotifier = &StaticGrid2DSpatialController{
			channel:      ch,
			WorldOffsetX: -40,
			WorldOffsetZ: -40,
			GridWidth:    8,
			GridHeight:   8,
			GridCols:     10,
			GridRows:     10,
		}
	}

	if owner == nil {
		ch.state = INIT
	} else {
		ch.state = OPEN
	}

	allChannels.Store(ch.id, ch)
	go ch.Tick()

	channelNum.WithLabelValues(ch.channelType.String()).Inc()

	return ch, nil
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
				if ch.HasOwner() {
					if ch.ownerConnection == conn {
						// Reset the owner if it's removed
						ch.ownerConnection = nil
						conn.Logger().Info("found removed ownner connection of channel", zap.Uint32("channelId", uint32(ch.id)))
					} else if conn != nil {
						ch.ownerConnection.sendUnsubscribed(MessageContext{}, ch, conn.(*Connection), 0)
					}
				}
			}
		}

		tickStart := time.Now()
		ch.tickFrames++

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
		ch.tickData(ch.GetTime())

		tickDuration := time.Since(tickStart)
		channelTickDuration.WithLabelValues(ch.channelType.String()).Set(float64(tickDuration) / float64(time.Millisecond))

		time.Sleep(ch.tickInterval - tickDuration)
	}
}

func (ch *Channel) Broadcast(ctx MessageContext) {
	for conn := range ctx.Channel.subscribedConnections {
		//c := GetConnection(connId)
		if conn == nil {
			continue
		}
		if ctx.Broadcast == channeldpb.BroadcastType_ALL_BUT_SENDER && conn == ctx.Connection {
			continue
		}
		conn.Send(ctx)
	}
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

func (ch *Channel) Logger() *zap.Logger {
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
