package main

import (
	"fmt"
	"log"

	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"channeld.clewcat.com/channeld/pkg/client"
	"channeld.clewcat.com/channeld/pkg/replay"
	"google.golang.org/protobuf/proto"
)

type SubedChannelMap map[uint32]bool
type ClientSubedChannelMap map[*client.ChanneldClient]SubedChannelMap

func SubToChannel(cm ClientSubedChannelMap, c *client.ChanneldClient, chId uint32) {
	subedChannelMap, ok := cm[c]
	if !ok {
		subedChannelMap = make(SubedChannelMap)
		cm[c] = subedChannelMap
	}
	subedChannelMap[chId] = true
}

func IsSubed(cm ClientSubedChannelMap, c *client.ChanneldClient, chId uint32) bool {
	subedChannelMap, ok := cm[c]
	if !ok {
		return false
	}
	subed, ok := subedChannelMap[chId]
	if ok {
		return subed
	}
	return false
}

func main() {

	rm, err := replay.CreateReplayMockBySetting("./test.json")
	if err != nil {
		fmt.Printf("create mock error: %v\n", err)
		return
	}
	subedChannel := make(ClientSubedChannelMap)

	rm.SetPreSendChannelIdHandler(
		func(channelId uint32, msgType channeldpb.MessageType, msgPack *channeldpb.MessagePack, c *client.ChanneldClient) (chId uint32, needToSend bool) {
			switch msgType {
			case channeldpb.MessageType_AUTH:
				return 0, true
			case channeldpb.MessageType_SUB_TO_CHANNEL:
				if IsSubed(subedChannel, c, channelId) {
					return channelId, false
				}
			}
			return 0, true
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
			log.Printf("client: %v conn id: %v", c.Id, subMsg)
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
			log.Printf("client: %v sub to: %v", c.Id, createChMsg)
			SubToChannel(subedChannel, c, channelId)
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
