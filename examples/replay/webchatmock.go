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

type ClientSubedChannel struct {
	channelId uint32
	m         sync.Mutex
}
type ClientSubedChannelMap map[*client.ChanneldClient]*ClientSubedChannel

var subToChannelMutexs = make([]sync.Mutex, 0)

var subedChannelMapMutex sync.RWMutex
var subedChannelMap = make(ClientSubedChannelMap)

func GetSubedChannel(c *client.ChanneldClient) (subedChannel *ClientSubedChannel, isExist bool) {
	subedChannelMapMutex.Lock()
	defer subedChannelMapMutex.Unlock()
	subedChannel, isExist = subedChannelMap[c]
	if !isExist {
		i := rand.Intn(channelNum)
		chId := channelPool[i]
		subedChannel = &ClientSubedChannel{
			channelId: chId,
		}
		subedChannelMap[c] = subedChannel
	}
	return subedChannel, isExist
}

func GetSubedChannelId(c *client.ChanneldClient) uint32 {
	subedChannelMapMutex.RLock()
	subedChannel, isExist := subedChannelMap[c]
	subedChannelMapMutex.RUnlock()
	if !isExist {
		return 0
	}
	subedChannel.m.Lock()
	defer subedChannel.m.Unlock()
	return subedChannel.channelId
}

func runChatMock() {
	rm, err := replay.CreateReplayMockByConfigFile("./web-chat/mock-config.json")
	if err != nil {
		log.Panicf("failed to create mock: %v\n", err)
		return
	}

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
					for i := channelNum; i >= 0; i-- {
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
		})

		c.Auth("test_lt", "test_pit")

		exited := false
		for !exited {
			c.Tick()
			time.Sleep(time.Millisecond * 10)
		}
	}()

	time.Sleep(time.Millisecond * 1000)

	rm.SetPreSendChannelIdHandler(
		func(channelId uint32, msgType channeldpb.MessageType, msgPack *channeldpb.MessagePack, c *client.ChanneldClient) (chId uint32, needToSend bool) {
			switch msgType {
			case channeldpb.MessageType_AUTH:
				return 0, true
			case channeldpb.MessageType_SUB_TO_CHANNEL:
				if subedChannel, isExist := GetSubedChannel(c); isExist {
					return 0, false
				} else {
					subedChannel.m.Lock()
					return subedChannel.channelId, true
				}
			default:
				return GetSubedChannelId(c), true
			}
		},
	)

	rm.SetPreSendMessageEntry(
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

	rm.AddMessageHandler(
		channeldpb.MessageType_SUB_TO_CHANNEL,
		func(c *client.ChanneldClient, channelId uint32, m proto.Message) {
			createChMsg, ok := m.(*channeldpb.SubscribedToChannelResultMessage)
			if !ok {
				return
			}
			if subedChannel, isExist := GetSubedChannel(c); isExist {
				subedChannel.m.Unlock()
			}
			log.Printf("client: %v sub to: %v", c.Id, createChMsg)
		},
	)

	rm.AddMessageHandler(
		channeldpb.MessageType_CREATE_CHANNEL,
		func(client *client.ChanneldClient, channelId uint32, m proto.Message) {
			createChMsg, ok := m.(*channeldpb.CreateChannelResultMessage)
			if !ok {
				return
			}
			log.Printf("created channeld: %v", createChMsg.ChannelId)
		},
	)
	rm.RunMock()

}
