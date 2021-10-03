package channeld

import (
	"container/list"
	"fmt"
	"log"
	"time"

	"clewcat.com/channeld/proto"
)

/* Use the definitions in channeld.proto instead
type ChannelType uint8

const (
	WORLD   ChannelType = 1
	PRIVATE ChannelType = 2
	LEVEL   ChannelType = 3
	REGION  ChannelType = 4
)
*/

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
	msg     Message
	handler MessageHandlerFunc
	conn    *Connection
}

type Channel struct {
	id                    ChannelId
	channelType           proto.ChannelType
	state                 ChannelState
	ownerConnection       *Connection
	subscribedConnections map[ConnectionId]*ChannelSubscription
	data                  *ChannelData
	inMsgQueue            chan channelMessage
	fanOutQueue           *list.List
	startTime             time.Time // Time since channel created
	tickInterval          time.Duration
	tickFrames            int
	removing              chan bool
}

const (
	GlobalChannelId     ChannelId     = 0
	DefaultTickInterval time.Duration = time.Millisecond * 50
)

var nextChannelId ChannelId = GlobalChannelId
var allChannels map[ChannelId]*Channel
var globalChannel *Channel

func InitChannels() {
	allChannels = make(map[ChannelId]*Channel, 1024)
	globalChannel = CreateChannel(proto.ChannelType_GLOBAL, nil)
	allChannels[GlobalChannelId] = globalChannel
}

func GetChannel(id ChannelId) *Channel {
	return allChannels[id]
}

func CreateChannel(t proto.ChannelType, owner *Connection) *Channel {
	if t == proto.ChannelType_GLOBAL && globalChannel != nil {
		log.Panicln("Failed to create WORLD channel as it already exists.")
	}

	ch := &Channel{
		id:                    nextChannelId,
		channelType:           t,
		ownerConnection:       owner,
		subscribedConnections: make(map[ConnectionId]*ChannelSubscription),
		data:                  NewChannelData(t, nil),
		inMsgQueue:            make(chan channelMessage, 1024),
		fanOutQueue:           list.New(),
		startTime:             time.Now(),
		tickInterval:          DefaultTickInterval,
		tickFrames:            0,
		removing:              make(chan bool),
	}
	if owner == nil {
		ch.state = INIT
	} else {
		ch.state = OPEN
	}
	allChannels[nextChannelId] = ch
	nextChannelId += 1
	go ch.Tick()
	return ch
}

func RemoveChannel(ch *Channel) {
	close(ch.inMsgQueue)
	ch.removing <- true
	delete(allChannels, ch.id)
}

func (ch *Channel) String() string {
	return fmt.Sprintf("Channel(%s %d)", ch.channelType.Descriptor().Name(), ch.id)
}

func (ch *Channel) Data() *ChannelData {
	return ch.data
}

func (ch *Channel) PutMessage(msg Message, handler MessageHandlerFunc, conn *Connection) {
	ch.inMsgQueue <- channelMessage{msg, handler, conn}
}

func (ch *Channel) GetTime() ChannelTime {
	return ChannelTime(time.Since(ch.startTime))
}

func (ch *Channel) Tick() {
	for {
		if <-ch.removing {
			return
		}

		tickStart := time.Now()
		ch.tickFrames++

		for len(ch.inMsgQueue) > 0 {
			cm := <-ch.inMsgQueue
			cm.handler(cm.msg, cm.conn, ch)
			if ch.tickInterval > 0 && time.Since(tickStart) >= ch.tickInterval {
				log.Printf("%s spent %dms handling messages, will delay the left ones(%d) to the next tick.", ch, time.Millisecond*time.Since(tickStart), len(ch.inMsgQueue))
				break
			}
		}
		ch.tickData(ch.GetTime())

		time.Sleep(ch.tickInterval - time.Since(tickStart))
	}
}

func (ch *Channel) Broadcast(msgType proto.MessageType, msg Message) {

}

func (ch *Channel) BroadcastSub(connId ConnectionId) {

}
