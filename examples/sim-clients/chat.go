package main

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/metaworking/channeld/examples/chat-rooms/chatpb"
	"github.com/metaworking/channeld/pkg/channeld"
	"github.com/metaworking/channeld/pkg/channeldpb"
	"github.com/metaworking/channeld/pkg/client"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

var globalSendNum int32 = 0
var globalReceiveNum int32 = 0
var globalPacketReceiveNum int32 = 0

func OnChatFinished() {
	desired2ReceivedNum := float32(ClientNum) * float32(globalSendNum)
	log.Printf("loss ratio: %f percent", (desired2ReceivedNum-float32(globalReceiveNum))/desired2ReceivedNum*100)
	log.Printf("desired to received num: %d, global recevied num: %d, global send num: %d, client num: %d", int(desired2ReceivedNum), globalReceiveNum, globalSendNum, ClientNum)
	log.Printf("global packet received num: %d", globalPacketReceiveNum)
}

func ChatClientFinishedFunc(c *client.ChanneldClient, data *clientData) {
	log.Printf("client%d, received num: %d, send num: %d, received self msg num: %d", c.Id, data.ctx["receivedNum"], data.ctx["sendNum"], data.ctx["receivedSelfNum"])
}

func ChatInitFunc(c *client.ChanneldClient, data *clientData) {
	data.ctx["receivedNum"] = 0
	data.ctx["receivedSelfNum"] = 0

	c.AddMessageHandler(uint32(channeldpb.MessageType_CHANNEL_DATA_UPDATE), func(client *client.ChanneldClient, channelId uint32, m client.Message) {
		atomic.AddInt32(&globalPacketReceiveNum, 1)
		msg := m.(*channeldpb.ChannelDataUpdateMessage)
		chatData := &chatpb.ChatChannelData{}
		msg.Data.UnmarshalTo(chatData)
		if len(chatData.ChatMessages) > 0 {
			n := len(chatData.ChatMessages)
			if chatData.ChatMessages[0].Sender == "System" {
				n--
			}
			atomic.AddInt32(&globalReceiveNum, int32(n))
			data.ctx["receivedNum"] = data.ctx["receivedNum"].(int) + n
			sneder := fmt.Sprintf("Client%d", client.Id)
			receivedSelfMsgNum := 0
			for _, chatMsg := range chatData.ChatMessages {
				if chatMsg.Sender == sneder {
					receivedSelfMsgNum++
				}
			}
			if receivedSelfMsgNum > 0 {
				data.ctx["receivedSelfNum"] = data.ctx["receivedSelfNum"].(int) + receivedSelfMsgNum
			}
		}
	})

	time.Sleep(2 * time.Second)
}

var ChatClientActions = []*clientAction{
	{
		name:        "listChannel",
		probability: 0,                        //1,
		minInterval: time.Millisecond * 20000, //2000
		perform: func(c *client.ChanneldClient, data *clientData) bool {
			c.Send(0, channeldpb.BroadcastType_NO_BROADCAST, uint32(channeldpb.MessageType_LIST_CHANNEL), &channeldpb.ListChannelMessage{}, nil)
			return true
		},
	},
	{
		name:        "createChannel",
		probability: 0, //0.05,
		minInterval: time.Millisecond * 10000,
		perform: func(c *client.ChanneldClient, data *clientData) bool {
			if len(c.ListedChannels) >= MaxChannelNum {
				return false
			}

			c.Send(0, channeldpb.BroadcastType_NO_BROADCAST, uint32(channeldpb.MessageType_CREATE_CHANNEL), &channeldpb.CreateChannelMessage{
				ChannelType: channeldpb.ChannelType_SUBWORLD,
				Metadata:    fmt.Sprintf("Room%d", data.rnd.Uint32()),
				SubOptions: &channeldpb.ChannelSubscriptionOptions{
					DataAccess:       channeld.Pointer(channeldpb.ChannelDataAccess_WRITE_ACCESS),
					DataFieldMasks:   make([]string, 0),
					FanOutIntervalMs: proto.Uint32(100),
				},
			}, func(_ *client.ChanneldClient, channelId uint32, m client.Message) {
				//log.Printf("Client(%d) created channel %d, data clientId: %d", client.Id, channelId, data.clientId)
			})
			return true
		},
	},
	{
		name:        "removeChannel",
		probability: 0,
		minInterval: time.Millisecond * 12000,
		perform: func(client *client.ChanneldClient, data *clientData) bool {
			if len(client.CreatedChannels) == 0 {
				return false
			}
			channelToRemove := randUint32(client.CreatedChannels)
			client.Send(0,
				channeldpb.BroadcastType_NO_BROADCAST,
				uint32(channeldpb.MessageType_REMOVE_CHANNEL),
				&channeldpb.RemoveChannelMessage{
					ChannelId: channelToRemove,
				},
				nil,
			)
			//log.Printf("Client(%d) CREATE_CHANNEL", client.Id)
			return true
		},
	},
	{
		name:        "subToChannel",
		probability: 0, //0.1,
		minInterval: time.Millisecond * 3000,
		perform: func(client *client.ChanneldClient, data *clientData) bool {
			if list := client.ListedChannels; len(list) > 1 {
				copy := make(map[uint32]struct{})
				for chid := range list {
					copy[chid] = struct{}{}
				}
				for chid := range client.SubscribedChannels {
					delete(copy, chid)
				}
				if len(copy) == 0 {
					return false
				}
				channelIdToSub := randUint32(copy)
				client.Send(channelIdToSub, channeldpb.BroadcastType_NO_BROADCAST, uint32(channeldpb.MessageType_SUB_TO_CHANNEL), &channeldpb.SubscribedToChannelMessage{
					ConnId: client.Id,
					SubOptions: &channeldpb.ChannelSubscriptionOptions{
						DataAccess:       channeld.Pointer(channeldpb.ChannelDataAccess_WRITE_ACCESS),
						FanOutIntervalMs: proto.Uint32(200),
						DataFieldMasks:   []string{},
					},
				}, nil)
				//log.Printf("Client(%d) SUB_TO_CHANNEL: %d", client.Id, channelIdToSub)
				return true
			}
			return false
		},
	},
	{
		name:        "unsubToChannel",
		probability: 0, //0.1,
		minInterval: time.Millisecond * 3000,
		perform: func(client *client.ChanneldClient, data *clientData) bool {
			if len(client.SubscribedChannels) <= 1 {
				return false
			}
			channelIdToUnsub := data.activeChannelId
			// Only unsubscribe from an inactive channel
			for channelIdToUnsub == data.activeChannelId {
				channelIdToUnsub = randUint32(client.SubscribedChannels)
			}

			client.Send(channelIdToUnsub, channeldpb.BroadcastType_NO_BROADCAST, uint32(channeldpb.MessageType_UNSUB_FROM_CHANNEL), &channeldpb.UnsubscribedFromChannelMessage{
				ConnId: client.Id,
			}, nil)
			//log.Printf("Client(%d) UNSUB_FROM_CHANNEL: %d", client.Id, channelIdToUnsub)
			return true
		},
	},
	{
		name:        "sendChatMessage",
		probability: 1,
		minInterval: time.Millisecond * 1000,
		perform: func(client *client.ChanneldClient, data *clientData) bool {
			inum, exists := data.ctx["sendNum"]
			var num int = 0
			if exists {
				num = inum.(int)
			}
			num++
			data.ctx["sendNum"] = num
			atomic.AddInt32(&globalSendNum, 1)
			content := fmt.Sprintf("{\"clientSendNum\": %d, \"globalSendNum\": %d, \"globalReceiveNum\": %d}", num, globalSendNum, globalReceiveNum)
			// content := fmt.Sprintf("Client send message, sent num: %d, global send num: %d", num, globalSendNum)
			log.Println(content)

			dataUpdate, _ := anypb.New(&chatpb.ChatChannelData{
				ChatMessages: []*chatpb.ChatMessage{{
					Sender:   fmt.Sprintf("Client%d", client.Id),
					SendTime: time.Now().UnixMilli(),
					Content:  content,
				}},
			})

			client.Send(data.activeChannelId, channeldpb.BroadcastType_NO_BROADCAST, uint32(channeldpb.MessageType_CHANNEL_DATA_UPDATE),
				&channeldpb.ChannelDataUpdateMessage{
					Data: dataUpdate,
				}, nil)
			return true
		},
	},
}
