package main

import (
	"log"
	"math/rand"
	"sync"
	"time"

	"channeld.clewcat.com/channeld/pkg/channeld"
	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"channeld.clewcat.com/channeld/pkg/client"
	"channeld.clewcat.com/channeld/pkg/replay"
	"google.golang.org/protobuf/proto"
)

var channelNum int = 6
var channelPoolMutex sync.RWMutex
var channelPool = make([]uint32, 0, channelNum)

func GetRandChannelId() uint32 {
	i := rand.Intn(channelNum)
	channelPoolMutex.RLock()
	chId := channelPool[i]
	channelPoolMutex.RUnlock()
	return chId
}

func GetSubedChannelId(c *client.ChanneldClient) (channelId uint32, isExist bool) {
	if len(c.SubscribedChannels) > 0 {
		for chId := range c.SubscribedChannels {
			return chId, true
		}
	}
	return 0, false
}

func runChatMock() {
	rm, err := replay.CreateReplayMockByConfigFile("./web-chat/mock-config.json")
	if err != nil {
		log.Panicf("failed to create mock: %v\n", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(channelNum)
	go func() {
		c, err := client.NewClient(rm.ChanneldAddr)
		if err != nil {
			log.Println(err)
			return
		}
		defer c.Disconnect()

		go func() {
			for {
				if err := c.Receive(); err != nil {
					log.Println(err)
					break
				}
			}
		}()

		c.AddMessageHandler(uint32(channeldpb.MessageType_AUTH), func(_ *client.ChanneldClient, channelId uint32, m client.Message) {
			resultMsg := m.(*channeldpb.AuthResultMessage)
			if resultMsg.ConnId == c.Id {
				if resultMsg.Result == channeldpb.AuthResultMessage_SUCCESSFUL {
					for i := channelNum; i > 0; i-- {
						c.Send(uint32(channeld.GlobalChannelId), channeldpb.BroadcastType_NO_BROADCAST, uint32(channeldpb.MessageType_CREATE_CHANNEL), &channeldpb.CreateChannelMessage{
							ChannelType: channeldpb.ChannelType_SUBWORLD,
							Data:        nil,
						}, nil)
					}
				} else {
					log.Panicln("master server failed to auth")
				}
			}
		})

		c.AddMessageHandler(uint32(channeldpb.MessageType_CREATE_CHANNEL), func(_ *client.ChanneldClient, channelId uint32, m client.Message) {
			resultMsg := m.(*channeldpb.CreateChannelResultMessage)
			channelPoolMutex.Lock()
			channelPool = append(channelPool, resultMsg.ChannelId)
			channelPoolMutex.Unlock()
			wg.Done()
		})

		c.Auth("test_lt", "test_pit")

		exited := false
		for !exited {
			c.Tick()
			time.Sleep(time.Millisecond * 10)
		}
	}()

	wg.Wait()

	rm.SetBeforeSendChannelIdHandler(
		func(channelId uint32, msgType channeldpb.MessageType, msgPack *channeldpb.MessagePack, c *client.ChanneldClient) (chId uint32, needToSend bool) {
			switch msgType {
			case channeldpb.MessageType_AUTH:
				return 0, true
			case channeldpb.MessageType_SUB_TO_CHANNEL:
				if _, isExist := GetSubedChannelId(c); isExist {
					return 0, false
				} else {
					return GetRandChannelId(), true
				}
			default:
				if chId, isExist := GetSubedChannelId(c); isExist {
					return chId, false
				} else {
					return 0, false
				}
			}
		},
	)

	rm.SetBeforeSendMessageEntry(
		channeldpb.MessageType_SUB_TO_CHANNEL,
		&channeldpb.SubscribedToChannelMessage{},
		func(msg proto.Message, msgPack *channeldpb.MessagePack, c *client.ChanneldClient) (needToSend bool) {
			subMsg, ok := msg.(*channeldpb.SubscribedToChannelMessage)
			if !ok {
				return false
			}
			log.Printf("client: %v message: %v", c.Id, subMsg)
			subMsg.ConnId = c.Id
			return true
		},
	)

	rm.SetNeedWaitMessageCallback(func(msgType channeldpb.MessageType, msgPack *channeldpb.MessagePack, c *client.ChanneldClient) bool {
		if msgType == channeldpb.MessageType_SUB_TO_CHANNEL {
			return true
		}
		return false
	})

	rm.RunMock()

}
