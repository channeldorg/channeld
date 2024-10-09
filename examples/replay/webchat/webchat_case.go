package webchat

import (
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/channeldorg/channeld/pkg/channeld"
	"github.com/channeldorg/channeld/pkg/channeldpb"
	"github.com/channeldorg/channeld/pkg/client"
	"github.com/channeldorg/channeld/pkg/replay"
	"google.golang.org/protobuf/proto"
)

var channelNum int = 6
var channelPoolMutex sync.RWMutex
var channelPool = make([]uint32, 0, channelNum)

func getRandChannelId() uint32 {
	i := rand.Intn(channelNum)
	channelPoolMutex.RLock()
	chId := channelPool[i]
	channelPoolMutex.RUnlock()
	return chId
}

func getSubedChannelId(c *client.ChanneldClient) (channelId uint32, isExist bool) {
	if len(c.SubscribedChannels) > 0 {
		for chId := range c.SubscribedChannels {
			return chId, true
		}
	}
	return 0, false
}

func RunChatMock() {
	rc, err := replay.CreateReplayClientByConfigFile("./webchat/case-config.json")
	if err != nil {
		log.Panicf("failed to create replay client: %v\n", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(channelNum)
	go func() {
		c, err := client.NewClient(rc.CaseConfig.ChanneldAddr)
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

	rc.SetAlterChannelIdBeforeSendHandler(
		func(channelId uint32, msgType channeldpb.MessageType, msgPack *channeldpb.MessagePack, c *client.ChanneldClient) (chId uint32, needToSend bool) {
			switch msgType {
			case channeldpb.MessageType_AUTH:
				return 0, true
			case channeldpb.MessageType_SUB_TO_CHANNEL:
				if _, isExist := getSubedChannelId(c); isExist {
					return 0, false
				} else {
					return getRandChannelId(), true
				}
			default:
				if chId, isExist := getSubedChannelId(c); isExist {
					return chId, true
				} else {
					return 0, false
				}
			}
		},
	)

	rc.SetBeforeSendMessageEntry(
		channeldpb.MessageType_SUB_TO_CHANNEL,
		&channeldpb.SubscribedToChannelMessage{},
		func(msg proto.Message, msgPack *channeldpb.MessagePack, c *client.ChanneldClient) (needToSend bool) {
			subMsg, ok := msg.(*channeldpb.SubscribedToChannelMessage)
			if !ok {
				return false
			}
			subMsg.ConnId = c.Id
			return true
		},
	)

	rc.SetNeedWaitMessageCallback(func(msgType channeldpb.MessageType, msgPack *channeldpb.MessagePack, c *client.ChanneldClient) bool {
		if msgType == channeldpb.MessageType_SUB_TO_CHANNEL {
			return true
		}
		return false
	})

	rc.Run()

}
