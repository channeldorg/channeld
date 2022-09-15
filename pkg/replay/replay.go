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

type MockClient struct {
	CprFilePath              string   `json:"cprFilePath"`
	Concurrent               int      `json:"concurrent"`
	ConnectInterval          Duration `json:"connectInterval"`
	RunningTime              Duration `json:"runningTime"`
	SleepEndOfSession        Duration `json:"sleepEndOfSession"`
	MaxTickInterval          Duration `json:"maxTickInterval"`
	ActionIntervalMultiplier float64  `json:"actionIntervalMultiplier"`
	WaitAuthSuccess          bool     `json:"waitAuthSuccess"`
	AuthOnlyOnce             bool     `json:"authOnlyOnce"`
	session                  *replaypb.ReplaySession
}

var DefaultMockClient = MockClient{
	Concurrent:               1,
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

type NeedWaitMessageCallbackFunc func(msgType channeldpb.MessageType, msgPack *channeldpb.MessagePack, c *client.ChanneldClient) bool

type ReplayClient struct {
	ChanneldAddr                    string       `json:"channeldAddr"`
	MockClients                     []MockClient `json:"mockClients"`
	alterChannelIdbeforeSendHandler AlterChannelIdBeforeSendHandlerFunc
	beforeSendMessageMap            map[channeldpb.MessageType]*beforeSendMessageMapEntry
	needWaitMessageCallback         NeedWaitMessageCallbackFunc
	messageMap                      map[channeldpb.MessageType]*messageMapEntry
}

func CreateReplayClientByConfigFile(configPath string) (*ReplayClient, error) {
	rc := &ReplayClient{}
	err := rc.LoadConfig(configPath)
	if err != nil {
		return nil, err
	}
	rc.beforeSendMessageMap = make(map[channeldpb.MessageType]*beforeSendMessageMapEntry)
	rc.messageMap = make(map[channeldpb.MessageType]*messageMapEntry)
	return rc, nil
}

func (c *MockClient) UnmarshalJSON(b []byte) error {
	type XMockClient MockClient
	xc := XMockClient(DefaultMockClient)
	if err := json.Unmarshal(b, &xc); err != nil {
		return err
	}

	*c = MockClient(xc)

	if s, err := ReadReplaySession(c.CprFilePath); err != nil {
		return err
	} else {
		c.session = s
	}

	return nil
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

func (rc *ReplayClient) LoadConfig(path string) error {

	config, err := ioutil.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(config, rc); err != nil {
			return fmt.Errorf("failed to unmarshall channel settings: %v", err)
		}
	} else {
		return fmt.Errorf("failed to read channel settings: %v", err)
	}
	return nil
}

func (rc *ReplayClient) SetAlterChannelIdBeforeSendHandler(handler AlterChannelIdBeforeSendHandlerFunc) {
	rc.alterChannelIdbeforeSendHandler = handler
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

func (rc *ReplayClient) SetNeedWaitMessageCallback(handler NeedWaitMessageCallbackFunc) {
	rc.needWaitMessageCallback = handler
}

func ReadReplaySession(cprPath string) (*replaypb.ReplaySession, error) {
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

func (rc *ReplayClient) RunReplay() {
	for _, mockClient := range rc.MockClients {
		mockClient.RunMockClient(rc)
	}

}

func (mc *MockClient) RunMockClient(rc *ReplayClient) {
	concurrent := mc.Concurrent
	maxTickInterval := time.Duration(mc.MaxTickInterval)
	connInterval := time.Duration(mc.ConnectInterval)
	runningTime := time.Duration(mc.RunningTime)
	session := mc.session
	authOnlyOnce := mc.AuthOnlyOnce
	waitAuthSuccess := mc.WaitAuthSuccess
	sleepEndOfSession := time.Duration(mc.SleepEndOfSession)

	packetsLen := len(session.Packets)

	var wg sync.WaitGroup // wait all mock clients timeout
	wg.Add(concurrent)

	for ci := 0; ci < concurrent; ci++ {
		go func() {
			c, err := client.NewClient(rc.ChanneldAddr)
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

			nextPacketsIndex := 0
			nextPacketTime := time.Now()
			firstAuth := true

			var waitMessageCallback int32 // counter for messages that has not been callback

			for t := time.Now(); time.Since(t) < runningTime && c.IsConnected(); {
				tickStartTime := time.Now()

				if nextPacketsIndex >= packetsLen {
					// If end of the session, delay sleepEndOfSession and replay packet that start of the session
					nextPacketsIndex = nextPacketsIndex % packetsLen
					nextPacketTime = nextPacketTime.Add(sleepEndOfSession)
				}

				replayPacket := session.Packets[nextPacketsIndex]
				offsetTime := time.Duration(replayPacket.OffsetTime)

				if waitMessageCallback == 0 && time.Since(nextPacketTime) >= offsetTime {

					nextPacketTime = nextPacketTime.Add(offsetTime)
					nextPacketsIndex++

					for _, msgPack := range replayPacket.Packet.Messages {

						msgType := channeldpb.MessageType(msgPack.MsgType)

						if msgType == channeldpb.MessageType_AUTH {
							if firstAuth {
								firstAuth = false
							} else if authOnlyOnce {
								continue
							}
						}

						channelId := msgPack.ChannelId
						if rc.alterChannelIdbeforeSendHandler != nil {
							newChId, needToSend := rc.alterChannelIdbeforeSendHandler(channelId, msgType, msgPack, c)
							if !needToSend {
								log.Printf("client: %v pass packet: %v", c.Id, replayPacket.Packet.String())
								continue
							}
							channelId = newChId
						}

						var receiveCallback func(client *client.ChanneldClient, channelId uint32, m client.Message) = nil
						needWaitMessageCallback := (rc.needWaitMessageCallback != nil && rc.needWaitMessageCallback(msgType, msgPack, c)) || (msgType == channeldpb.MessageType_AUTH && waitAuthSuccess)
						if needWaitMessageCallback {
							receiveCallback = func(client *client.ChanneldClient, channelId uint32, m client.Message) {
								atomic.AddInt32(&waitMessageCallback, -1)
							}
						}

						entry, ok := rc.beforeSendMessageMap[msgType]

						if !ok && entry == nil {
							log.Printf("client: %v packet: %v", c.Id, replayPacket.Packet.String())
							if needWaitMessageCallback {
								atomic.AddInt32(&waitMessageCallback, 1)
							}
							c.SendRaw(channelId, channeldpb.BroadcastType(msgPack.Broadcast), msgPack.MsgType, &msgPack.MsgBody, receiveCallback)
						} else {
							msg := proto.Clone(entry.msgTemp)
							err := proto.Unmarshal(msgPack.MsgBody, msg)
							if err != nil {
								log.Println(err)
								return
							}
							needToSend := entry.beforeSendHandler(msg, msgPack, c)
							if needToSend {
								log.Printf("client: %v packet: %v", c.Id, replayPacket.Packet.String())
								if needWaitMessageCallback {
									atomic.AddInt32(&waitMessageCallback, 1)
								}
								c.Send(channelId, channeldpb.BroadcastType(msgPack.Broadcast), msgPack.MsgType, msg, receiveCallback)
							} else {
								log.Printf("client: %v pass packet: %v", c.Id, replayPacket.Packet.String())
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
		// Run next mock client after connectionInterval
		time.Sleep(connInterval)
	}
	wg.Wait()
}
