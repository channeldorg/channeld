package main

import (
	"fmt"
	"math/rand"
	"time"

	"google.golang.org/protobuf/types/known/anypb"
)

var ChatClientActions = []*clientAction{
	{
		name:        "listChannel",
		probability: 1,
		minInterval: time.Millisecond * 20000, //2000
		perform: func(client *Client, data *clientData) bool {
			client.Send(0, channeldpb.BroadcastType_NO_BROADCAST, uint32(channeldpb.MessageType_LIST_CHANNEL), &channeldpb.ListChannelMessage{},
				func(c *Client, channelId uint32, m Message) {
					data.listedChannels = map[uint32]struct{}{}
					for _, info := range m.(*channeldpb.ListChannelResultMessage).Channels {
						data.listedChannels[info.ChannelId] = struct{}{}
					}
				})
			return true
		},
	},
	{
		name:        "createChannel",
		probability: 0.05,
		minInterval: time.Millisecond * 10000,
		perform: func(client *Client, data *clientData) bool {
			if len(data.listedChannels) >= MaxChannelNum {
				return false
			}

			client.Send(0, channeldpb.BroadcastType_NO_BROADCAST, uint32(channeldpb.MessageType_CREATE_CHANNEL), &channeldpb.CreateChannelMessage{
				ChannelType: channeldpb.ChannelType_SUBWORLD,
				Metadata:    fmt.Sprintf("Room%d", data.rnd.Uint32()),
				SubOptions: &channeldpb.ChannelSubscriptionOptions{
					CanUpdateData:    true,
					DataFieldMasks:   make([]string, 0),
					FanOutIntervalMs: 100,
				},
			}, func(c *Client, channelId uint32, m Message) {
				data.createdChannelIds[channelId] = struct{}{}
				//log.Printf("Client(%d) created channel %d, data clientId: %d", client.Id, channelId, data.clientId)
			})
			return true
		},
	},
	{
		name:        "removeChannel",
		probability: 0,
		minInterval: time.Millisecond * 12000,
		perform: func(client *Client, data *clientData) bool {
			if len(data.createdChannelIds) == 0 {
				return false
			}
			channelToRemove := randUint32(data.createdChannelIds)
			client.Send(0,
				channeldpb.BroadcastType_NO_BROADCAST,
				uint32(channeldpb.MessageType_REMOVE_CHANNEL),
				&channeldpb.RemoveChannelMessage{
					ChannelId: channelToRemove,
				},
				nil,
			)
			// Remove the channel id from the list
			delete(data.createdChannelIds, channelToRemove)
			//log.Printf("Client(%d) CREATE_CHANNEL", client.Id)
			return true
		},
	},
	{
		name:        "subToChannel",
		probability: 0.1,
		minInterval: time.Millisecond * 3000,
		perform: func(client *Client, data *clientData) bool {
			if list := data.listedChannels; len(list) > 1 {
				copy := make(map[uint32]struct{})
				for chid := range list {
					copy[chid] = struct{}{}
				}
				for chid := range client.subscribedChannels {
					delete(copy, chid)
				}
				if len(copy) == 0 {
					return false
				}
				channelIdToSub := randUint32(copy)
				client.Send(channelIdToSub, channeldpb.BroadcastType_NO_BROADCAST, uint32(channeldpb.MessageType_SUB_TO_CHANNEL), &channeldpb.SubscribedToChannelMessage{
					ConnId: client.Id,
					SubOptions: &channeldpb.ChannelSubscriptionOptions{
						CanUpdateData:    true,
						FanOutIntervalMs: 200,
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
		probability: 0.1,
		minInterval: time.Millisecond * 3000,
		perform: func(client *Client, data *clientData) bool {
			if len(client.subscribedChannels) <= 1 {
				return false
			}
			channelIdToUnsub := data.activeChannelId
			// Only unsubscribe from an inactive channel
			for channelIdToUnsub == data.activeChannelId {
				channelIdToUnsub = randUint32(client.subscribedChannels)
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
		perform: func(client *Client, data *clientData) bool {
			dataUpdate, _ := anypb.New(&channeldpb.ChatChannelData{
				ChatMessages: []*channeldpb.ChatMessage{{
					Sender:   fmt.Sprintf("Client%d", client.Id),
					SendTime: time.Now().Unix(),
					Content:  fmt.Sprintf("How are you, User%d?", rand.Intn(ClientNum)),
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
