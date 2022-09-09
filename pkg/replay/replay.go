package replay

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"channeld.clewcat.com/channeld/pkg/client"
	"channeld.clewcat.com/channeld/pkg/replaypb"
	"google.golang.org/protobuf/proto"
)

type JSON_MockClientSetting struct {
	Concurrent               int
	CprFilePath              string
	RunningTime              string
	MaxTickInterval          string
	ActionIntervalMultiplier float64
	WaitAuthSuccess          bool
	AuthOnlyOnce             bool
}

type JSON_ReplayMockSetting struct {
	ChanneldAddr   string
	ClientSettings []JSON_MockClientSetting
}

type MockClientSetting struct {
	Concurrent               int
	CprFilePath              string
	Session                  *replaypb.ReplaySession
	RunningTime              time.Duration
	MaxTickInterval          time.Duration
	ActionIntervalMultiplier float64
	WaitAuthSuccess          bool
	AuthOnlyOnce             bool
}

type PreSendChannelIdHandlerFunc func(channelId uint32, msgType channeldpb.MessageType, msgPack *channeldpb.MessagePack, c *client.ChanneldClient) (chId uint32, needToSend bool)

type PreSendMessageHandlerFunc func(msg proto.Message, msgPack *channeldpb.MessagePack, c *client.ChanneldClient) (needToSend bool)
type preSendMessageMapEntry struct {
	msgTemp        proto.Message
	preSendHandler PreSendMessageHandlerFunc
}

type MessageHandlerFunc func(c *client.ChanneldClient, channelId uint32, m proto.Message)
type messageMapEntry struct {
	msg      proto.Message
	handlers []MessageHandlerFunc
}

type ReplayMock struct {
	ChanneldAddr            string
	ClientSettings          []MockClientSetting
	preSendChannelIdHandler PreSendChannelIdHandlerFunc
	preSendMessageMap       map[channeldpb.MessageType]*preSendMessageMapEntry
	messageMap              map[channeldpb.MessageType]*messageMapEntry
}

func CreateReplayMockBySetting(settingPath string) (*ReplayMock, error) {
	rm := &ReplayMock{}
	err := rm.LoadSetting(settingPath)
	if err != nil {
		return nil, err
	}
	rm.preSendMessageMap = make(map[channeldpb.MessageType]*preSendMessageMapEntry)
	rm.messageMap = make(map[channeldpb.MessageType]*messageMapEntry)
	return rm, nil
}

func (rm *ReplayMock) LoadSetting(path string) error {
	var jsonSetting JSON_ReplayMockSetting
	config, err := ioutil.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(config, &jsonSetting); err != nil {
			return fmt.Errorf("failed to unmarshall channel settings: %v", err)
		}
	} else {
		return fmt.Errorf("failed to read channel settings: %v", err)
	}

	for _, cs := range jsonSetting.ClientSettings {
		if t, err := time.ParseDuration(cs.RunningTime); err != nil {
			return fmt.Errorf("failed to parse running time to time.Duration: %v", err)
		} else if s, err := ReadReplaySession(cs.CprFilePath); err != nil {
			return fmt.Errorf("failed to read channeld packet recording file: %v", err)
		} else {

			mti := time.Millisecond * 100
			if cs.MaxTickInterval != "" {
				if t, err := time.ParseDuration(cs.MaxTickInterval); err != nil {
					return fmt.Errorf("failed to parse max tick interval to time.Duration: %v", err)
				} else {
					mti = t
				}
			}
			aim := cs.ActionIntervalMultiplier
			if aim <= 0 {
				aim = 1
			}
			rm.ClientSettings = append(rm.ClientSettings, MockClientSetting{
				Concurrent:               cs.Concurrent,
				CprFilePath:              cs.CprFilePath,
				Session:                  s,
				RunningTime:              t,
				MaxTickInterval:          mti,
				ActionIntervalMultiplier: aim,
				WaitAuthSuccess:          cs.WaitAuthSuccess,
				AuthOnlyOnce:             cs.AuthOnlyOnce,
			})
		}
	}

	rm.ChanneldAddr = jsonSetting.ChanneldAddr

	return nil
}

func (rm *ReplayMock) SetPreSendChannelIdHandler(handler PreSendChannelIdHandlerFunc) {
	rm.preSendChannelIdHandler = handler
}

func (rm *ReplayMock) SetPreSendMessageEntry(msgType channeldpb.MessageType, msgTemp proto.Message, handler PreSendMessageHandlerFunc) {
	rm.preSendMessageMap[msgType] = &preSendMessageMapEntry{
		msgTemp:        msgTemp,
		preSendHandler: handler,
	}
}

func (rm *ReplayMock) AddMessageHandler(msgType channeldpb.MessageType, handlers ...MessageHandlerFunc) {
	entry := rm.messageMap[msgType]
	if entry != nil {
		entry.handlers = append(entry.handlers, handlers...)
	} else {
		rm.messageMap[msgType] = &messageMapEntry{
			handlers: handlers,
		}
	}
}

func (rm *ReplayMock) SetMessageEntry(msgType uint32, msgTemp proto.Message, handlers ...MessageHandlerFunc) {
	rm.messageMap[channeldpb.MessageType(msgType)] = &messageMapEntry{
		msg:      msgTemp,
		handlers: handlers,
	}
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

func (rm *ReplayMock) RunMock() {
	var wg sync.WaitGroup
	channeldAddr := rm.ChanneldAddr
	for _, clientSetting := range rm.ClientSettings {
		wg.Add(1)
		rs := clientSetting.Session
		mti := clientSetting.MaxTickInterval
		aim := clientSetting.ActionIntervalMultiplier
		was := clientSetting.WaitAuthSuccess
		aoo := clientSetting.AuthOnlyOnce
		stopFlag := make(chan struct{})
		for ci := 0; ci < clientSetting.Concurrent; ci++ {
			go func() {
				c, err := client.NewClient(channeldAddr)
				if err != nil {
					log.Println(err)
					return
				}
				defer c.Disconnect()

				for msgType, entry := range rm.messageMap {
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

				go func() {
					for {
						tickStartTime := time.Now()
						c.Tick()
						timer := time.NewTimer(mti - time.Since(tickStartTime))
						select {
						case <-stopFlag:
							return
						case <-timer.C:
						}
					}
				}()

				rm.ReplaySession(c, rs, aim, was, aoo, stopFlag)
			}()
		}
		t := clientSetting.RunningTime
		go func() {
			time.Sleep(t)
			close(stopFlag)
			wg.Done()
		}()
	}

	wg.Wait()
}

func (rm *ReplayMock) ReplaySession(c *client.ChanneldClient, rs *replaypb.ReplaySession, actionIntervalMultiplier float64, waitAuthSuccess bool, authOnlyOnce bool, stopFlag chan struct{}) error {
	if !c.IsConnected() {
		return errors.New("client not connected")
	}
	hasAuth := make(chan struct{})
	hasAuthClosed := false
	var hasAuthClosedLock sync.Mutex
	firstAuth := true
	if waitAuthSuccess {
		c.AddMessageHandler(
			uint32(channeldpb.MessageType_AUTH),
			func(client *client.ChanneldClient, channelId uint32, m client.Message) {
				resultMsg := m.(*channeldpb.AuthResultMessage)
				if resultMsg.ConnId == c.Id {
					if resultMsg.Result == channeldpb.AuthResultMessage_SUCCESSFUL {
						hasAuthClosedLock.Lock()
						if !hasAuthClosed {
							hasAuthClosed = true
							close(hasAuth)
						}
						hasAuthClosedLock.Unlock()
					} else {
						log.Panicln("mock client failed to auth")
					}
				}
			},
		)
	} else {
		close(hasAuth)
	}
	var timer *time.Timer
	for {
		for _, packet := range rs.Packets {
			startTime := time.Now()
			for _, msgPack := range packet.Packet.Messages {
				msgType := channeldpb.MessageType(msgPack.MsgType)

				if msgType == channeldpb.MessageType_AUTH {
					if firstAuth {
						firstAuth = false
					} else if authOnlyOnce {
						continue
					}
				} else {
					if waitAuthSuccess {
						<-hasAuth
					}
				}

				channelId := msgPack.ChannelId
				if rm.preSendChannelIdHandler != nil {
					newChId, needToSend := rm.preSendChannelIdHandler(channelId, msgType, msgPack, c)
					if !needToSend {
						continue
					}
					channelId = newChId
				}

				entry, ok := rm.preSendMessageMap[msgType]
				if !ok && entry == nil {
					c.SendRaw(channelId, channeldpb.BroadcastType(msgPack.Broadcast), msgPack.MsgType, &msgPack.MsgBody, nil)
				} else {
					msg := proto.Clone(entry.msgTemp)
					err := proto.Unmarshal(msgPack.MsgBody, msg)
					if err != nil {
						return err
					}
					needToSend := entry.preSendHandler(msg, msgPack, c)
					if needToSend {
						c.Send(channelId, channeldpb.BroadcastType(msgPack.Broadcast), msgPack.MsgType, msg, nil)
					}
				}
			}
			log.Printf("v: %v", packet.Packet.String())
			timer = time.NewTimer(time.Duration(actionIntervalMultiplier * float64(time.Duration(packet.OffsetTime)-time.Since(startTime))))
			select {
			case <-stopFlag:
				return nil
			case <-timer.C:
			}
		}
	}
}
