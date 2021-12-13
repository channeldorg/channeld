package channeld

import (
	"container/list"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"channeld.clewcat.com/channeld/proto"
	"go.uber.org/zap"
)

type ChannelState uint8

const (
	INIT     ChannelState = 0
	OPEN     ChannelState = 1
	HANDOVER ChannelState = 2
)

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

type Channel struct {
	id                    ChannelId
	channelType           proto.ChannelType
	state                 ChannelState
	ownerConnection       *Connection
	subscribedConnections map[ConnectionId]*ChannelSubscription
	metadata              string // Read-only property, e.g. name
	data                  *ChannelData
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
	GlobalChannelId     ChannelId     = 0
	DefaultTickInterval time.Duration = time.Millisecond * 10
)

var nextChannelId ChannelId = GlobalChannelId
var allChannels sync.Map //map[ChannelId]*Channel
var globalChannel *Channel

func InitChannels() {
	globalChannel, _ = CreateChannel(proto.ChannelType_GLOBAL, nil)
	allChannels.Store(GlobalChannelId, globalChannel)
}

func GetChannel(id ChannelId) *Channel {
	ch, ok := allChannels.Load(id)
	if ok {
		return ch.(*Channel)
	} else {
		return nil
	}
}

func CreateChannel(t proto.ChannelType, owner *Connection) (*Channel, error) {
	if t == proto.ChannelType_GLOBAL && globalChannel != nil {
		return nil, errors.New("failed to create WORLD channel as it already exists")
	}

	ch := &Channel{
		id:                    nextChannelId,
		channelType:           t,
		ownerConnection:       owner,
		subscribedConnections: make(map[ConnectionId]*ChannelSubscription),
		/* Channel data is not created by default. See handleCreateChannel().
		data:                  ReflectChannelData(t, nil),
		*/
		inMsgQueue:   make(chan channelMessage, 1024),
		fanOutQueue:  list.New(),
		startTime:    time.Now(),
		tickInterval: DefaultTickInterval,
		tickFrames:   0,
		logger: logger.With(
			zap.String("channelType", t.String()),
			zap.Uint32("channelId", uint32(nextChannelId)),
		),
		removing: 0,
	}
	if owner == nil {
		ch.state = INIT
	} else {
		ch.state = OPEN
	}
	allChannels.Store(nextChannelId, ch)
	nextChannelId += 1
	go ch.Tick()

	channelNum.WithLabelValues(ch.channelType.String()).Inc()

	return ch, nil
}

func RemoveChannel(ch *Channel) {
	atomic.AddInt32(&ch.removing, 1)
	close(ch.inMsgQueue)
	allChannels.Delete(ch.id)

	channelNum.WithLabelValues(ch.channelType.String()).Dec()
}

func (ch *Channel) IsRemoving() bool {
	return ch.removing > 0
}

func (ch *Channel) PutMessage(msg Message, handler MessageHandlerFunc, conn *Connection, pack *proto.MessagePack) {
	if ch.IsRemoving() {
		return
	}
	ch.inMsgQueue <- channelMessage{ctx: MessageContext{
		MsgType:    proto.MessageType(pack.MsgType),
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

		if ch.ownerConnection != nil {
			if ch.ownerConnection.IsRemoving() {
				ch.ownerConnection = nil
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

		time.Sleep(ch.tickInterval - time.Since(tickStart))
	}
}

func (ch *Channel) Broadcast(ctx MessageContext) {
	for connId := range ctx.Channel.subscribedConnections {
		c := GetConnection(connId)
		if c == nil {
			continue
		}
		if ctx.Broadcast == proto.BroadcastType_ALL_BUT_SENDER && c == ctx.Connection {
			continue
		}
		c.Send(ctx)
	}
}

func (ch *Channel) String() string {
	return fmt.Sprintf("Channel(%s %d)", ch.channelType.String(), ch.id)
}

func (ch *Channel) Logger() *zap.Logger {
	return ch.logger
}
