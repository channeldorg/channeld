package replay

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"channeld.clewcat.com/channeld/pkg/client"
	"channeld.clewcat.com/channeld/pkg/replaypb"
	"google.golang.org/protobuf/proto"
)

type Duration time.Duration

type CaseConfig struct {
	ChanneldAddr     string                  `json:"channeldAddr"`
	ConnectionGroups []ConnectionGroupConfig `json:"connectionGroups"`
}

type ConnectionGroupConfig struct {
	CprFilePath              string   `json:"cprFilePath"`
	ConnectionNumber         int      `json:"connectionNumber"`         // Number of connections
	ConnectInterval          Duration `json:"connectInterval"`          // start connections interval
	RunningTime              Duration `json:"runningTime"`              // replay session running time
	SleepEndOfSession        Duration `json:"sleepEndOfSession"`        // sleep each end of session
	MaxTickInterval          Duration `json:"maxTickInterval"`          // channeld connection max tick time
	ActionIntervalMultiplier float64  `json:"actionIntervalMultiplier"` // used to adjust the packet offsettime (ActionIntervalMultiplier * ReplayPacket.Offsettime)
	WaitAuthSuccess          bool     `json:"waitAuthSuccess"`          // if true, replay loop will wait for auth success
	AuthOnlyOnce             bool     `json:"authOnlyOnce"`             // if true, only send auth message once in the entire replay
}

var DefaultConnGroupConfig = ConnectionGroupConfig{
	ConnectionNumber:         1,
	ActionIntervalMultiplier: 1,
	WaitAuthSuccess:          true,
	AuthOnlyOnce:             true,
}

type AlterChannelIdBeforeSendHandlerFunc func(channelId uint32, msgType channeldpb.MessageType, msgPack *channeldpb.MessagePack, c *client.ChanneldClient) (chId uint32, needToSend bool)

type BeforeSendMessageHandlerFunc func(msg proto.Message, msgPack *channeldpb.MessagePack, c *client.ChanneldClient) (needToSend bool)
type beforeSendMessageMapEntry struct {
	msgTemp           proto.Message
	beforeSendHandler BeforeSendMessageHandlerFunc
}

type MessageHandlerFunc func(c *client.ChanneldClient, channelId uint32, m proto.Message)
type messageMapEntry struct {
	msg      proto.Message
	handlers []MessageHandlerFunc
}

type NeedWaitMessageCallbackHandlerFunc func(msgType channeldpb.MessageType, msgPack *channeldpb.MessagePack, c *client.ChanneldClient) bool

type ReplayClient struct {
	CaseConfig                      CaseConfig
	ConnectionGroups                []ConnectionGroup
	alterChannelIdBeforeSendHandler AlterChannelIdBeforeSendHandlerFunc
	beforeSendMessageMap            map[channeldpb.MessageType]*beforeSendMessageMapEntry
	needWaitMessageCallbackHandler  NeedWaitMessageCallbackHandlerFunc
	messageMap                      map[channeldpb.MessageType]*messageMapEntry
}

type ConnectionGroup struct {
	config  ConnectionGroupConfig
	session *replaypb.ReplaySession
}

func CreateReplayClientByConfigFile(configPath string) (*ReplayClient, error) {
	rc := &ReplayClient{}
	err := rc.LoadCaseConfig(configPath)
	if err != nil {
		return nil, err
	}
	rc.beforeSendMessageMap = make(map[channeldpb.MessageType]*beforeSendMessageMapEntry)
	rc.messageMap = make(map[channeldpb.MessageType]*messageMapEntry)
	return rc, nil
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		*d = Duration(time.Duration(value))
		return nil
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(tmp)
		return nil
	default:
		return errors.New("invalid duration")
	}
}

func (rc *ReplayClient) LoadCaseConfig(path string) error {

	config, err := ioutil.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(config, &rc.CaseConfig); err != nil {
			return fmt.Errorf("failed to unmarshall case config: %v", err)
		}
	} else {
		return fmt.Errorf("failed to load case config: %v", err)
	}

	for _, c := range rc.CaseConfig.ConnectionGroups {
		session, err := ReadReplaySessionFile(c.CprFilePath)
		if err != nil {
			return fmt.Errorf("failed to read replay session file: %v", err)
		}
		rc.ConnectionGroups = append(rc.ConnectionGroups, ConnectionGroup{
			config:  c,
			session: session,
		})
	}
	return nil
}

func (rc *ReplayClient) SetAlterChannelIdBeforeSendHandler(handler AlterChannelIdBeforeSendHandlerFunc) {
	rc.alterChannelIdBeforeSendHandler = handler
}

func (rc *ReplayClient) SetBeforeSendMessageEntry(msgType channeldpb.MessageType, msgTemp proto.Message, handler BeforeSendMessageHandlerFunc) {
	rc.beforeSendMessageMap[msgType] = &beforeSendMessageMapEntry{
		msgTemp:           msgTemp,
		beforeSendHandler: handler,
	}
}

func (rc *ReplayClient) AddMessageHandler(msgType channeldpb.MessageType, handlers ...MessageHandlerFunc) {
	entry := rc.messageMap[msgType]
	if entry != nil {
		entry.handlers = append(entry.handlers, handlers...)
	} else {
		rc.messageMap[msgType] = &messageMapEntry{
			handlers: handlers,
		}
	}
}

func (rc *ReplayClient) SetMessageEntry(msgType uint32, msgTemp proto.Message, handlers ...MessageHandlerFunc) {
	rc.messageMap[channeldpb.MessageType(msgType)] = &messageMapEntry{
		msg:      msgTemp,
		handlers: handlers,
	}
}

func (rc *ReplayClient) SetNeedWaitMessageCallback(handler NeedWaitMessageCallbackHandlerFunc) {
	rc.needWaitMessageCallbackHandler = handler
}

func ReadReplaySessionFile(cprPath string) (*replaypb.ReplaySession, error) {
	data, err := os.ReadFile(cprPath)
	if err != nil {
		return nil, err
	}

	var rs replaypb.ReplaySession
	if err = proto.Unmarshal(data, &rs); err != nil {
		return nil, err
	}

	return &rs, nil
}

func (rc *ReplayClient) Run() {
	for _, cg := range rc.ConnectionGroups {
		cg.run(rc)
	}

}

func (cg *ConnectionGroup) run(rc *ReplayClient) {
	connNumber := cg.config.ConnectionNumber
	maxTickInterval := time.Duration(cg.config.MaxTickInterval)
	connInterval := time.Duration(cg.config.ConnectInterval)
	runningTime := time.Duration(cg.config.RunningTime)
	session := cg.session
	authOnlyOnce := cg.config.AuthOnlyOnce
	waitAuthSuccess := cg.config.WaitAuthSuccess
	sleepEndOfSession := time.Duration(cg.config.SleepEndOfSession)
	actionIntervalMultiplier := cg.config.ActionIntervalMultiplier

	sessionPacketNum := len(session.Packets)

	var wg sync.WaitGroup // wait all connections in the group to timeout
	wg.Add(connNumber)

	for ci := 0; ci < connNumber; ci++ {
		go func() {
			c, err := client.NewClient(rc.CaseConfig.ChanneldAddr)
			if err != nil {
				log.Println(err)
				return
			}

			// Register msg handlers
			for msgType, entry := range rc.messageMap {
				handlers := make([]client.MessageHandlerFunc, 0, len(entry.handlers))
				for _, handler := range entry.handlers {
					handlers = append(handlers, func(client *client.ChanneldClient, channelId uint32, m client.Message) {
						handler(client, channelId, m)
					})
				}
				err := c.AddMessageHandler(uint32(msgType), handlers...)
				if err != nil {
					c.SetMessageEntry(uint32(msgType), entry.msg, handlers...)
				}
			}

			go func() {
				for {
					if err := c.Receive(); err != nil {
						log.Println(err)
						return
					}
				}
			}()

			nextPacketIndex := 0        // replay packet index in session
			prePacketTime := time.Now() // used to determine whether the next packet has arrived at the sending time
			isFirstAuth := true

			var waitMessageCallback int32 // counter for messages that has not been callback

			for t := time.Now(); time.Since(t) < runningTime && c.IsConnected(); {
				tickStartTime := time.Now()

				if nextPacketIndex >= sessionPacketNum {
					// If end of the session, delay sleepEndOfSession and replay packet that start of the session
					nextPacketIndex = nextPacketIndex % sessionPacketNum
					prePacketTime = prePacketTime.Add(sleepEndOfSession)
				}

				replayPacket := session.Packets[nextPacketIndex]
				offsetTime := time.Duration(float64(replayPacket.OffsetTime) * actionIntervalMultiplier)

				// If no messages that has not been callback and the next packet has arrived at the sending time
				// try to send the packet
				if waitMessageCallback == 0 && time.Since(prePacketTime) >= offsetTime {

					prePacketTime = prePacketTime.Add(offsetTime)
					nextPacketIndex++

					// Try to send messages in packet
					for _, msgPack := range replayPacket.Packet.Messages {

						msgType := channeldpb.MessageType(msgPack.MsgType)

						if msgType == channeldpb.MessageType_AUTH {
							if isFirstAuth {
								isFirstAuth = false
							} else if authOnlyOnce {
								continue
							}
						}

						channelId := msgPack.ChannelId
						// Alter channelId if user set the alterChannelIdBeforeSendHandler
						if rc.alterChannelIdBeforeSendHandler != nil {
							newChId, needToSend := rc.alterChannelIdBeforeSendHandler(channelId, msgType, msgPack, c)
							if !needToSend {
								log.Printf("Connection(%v) pass message: { %v }", c.Id, msgPack.String())
								continue
							}
							channelId = newChId
						}

						var messageCallback func(client *client.ChanneldClient, channelId uint32, m client.Message) = nil
						needWaitMessageCallback := (rc.needWaitMessageCallbackHandler != nil && rc.needWaitMessageCallbackHandler(msgType, msgPack, c)) || (msgType == channeldpb.MessageType_AUTH && waitAuthSuccess)
						if needWaitMessageCallback {
							messageCallback = func(client *client.ChanneldClient, channelId uint32, m client.Message) {
								atomic.AddInt32(&waitMessageCallback, -1)
							}
						}

						// The handler for alter or abandon message before send message
						entry, ok := rc.beforeSendMessageMap[msgType]

						if !ok && entry == nil {
							// Send bytes directly
							log.Printf("Connection(%v) send message: { %v }", c.Id, msgPack.String())
							if needWaitMessageCallback {
								atomic.AddInt32(&waitMessageCallback, 1)
							}
							c.SendRaw(channelId, channeldpb.BroadcastType(msgPack.Broadcast), msgPack.MsgType, &msgPack.MsgBody, messageCallback)
						} else {
							// Copy message for uesr modify
							msg := proto.Clone(entry.msgTemp)
							err := proto.Unmarshal(msgPack.MsgBody, msg)
							if err != nil {
								log.Println(err)
								return
							}
							needToSend := entry.beforeSendHandler(msg, msgPack, c)
							if needToSend {
								log.Printf("Connection(%v) send message: { %v }", c.Id, msgPack.String())
								if needWaitMessageCallback {
									atomic.AddInt32(&waitMessageCallback, 1)
								}
								c.Send(channelId, channeldpb.BroadcastType(msgPack.Broadcast), msgPack.MsgType, msg, messageCallback)
							} else {
								log.Printf("Connection(%v) pass message: { %v }", c.Id, msgPack.String())
								continue
							}
						}
					}
				}
				c.Tick()
				time.Sleep(maxTickInterval - time.Since(tickStartTime))
			}
			c.Disconnect()
			time.Sleep(time.Millisecond * 100) // wait totally disconnect
			wg.Done()
		}()
		// Run next connection after connectionInterval
		time.Sleep(connInterval)
	}
	wg.Wait()
}
