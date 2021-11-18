package main

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"channeld.clewcat.com/channeld/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type clientData struct {
	clientId          uint32
	rnd               *rand.Rand
	activeChannelId   uint32
	createdChannelIds map[uint32]struct{}
	listedChannels    map[uint32]struct{} //[]*proto.ListChannelResultMessage_ChannelInfo
}

type clientAction struct {
	name        string
	probability float32
	minInterval time.Duration
	perform     func(client *Client, data *clientData) bool
}

var clientActions = []*clientAction{
	{
		name:        "listChannel",
		probability: 1,
		minInterval: time.Millisecond * 10000, //2000
		perform: func(client *Client, data *clientData) bool {
			client.Send(0, proto.BroadcastType_NO, uint32(proto.MessageType_LIST_CHANNEL), &proto.ListChannelMessage{},
				func(c *Client, channelId uint32, m Message) {
					data.listedChannels = map[uint32]struct{}{}
					for _, info := range m.(*proto.ListChannelResultMessage).Channels {
						data.listedChannels[info.ChannelId] = struct{}{}
					}
				})
			return true
		},
	},
	{
		name:        "createChannel",
		probability: 0.3,
		minInterval: time.Millisecond * 10000,
		perform: func(client *Client, data *clientData) bool {
			client.Send(0, proto.BroadcastType_NO, uint32(proto.MessageType_CREATE_CHANNEL), &proto.CreateChannelMessage{
				ChannelType: proto.ChannelType_SUBWORLD,
				Metadata:    fmt.Sprintf("Room%d", data.rnd.Uint32()),
				SubOptions: &proto.ChannelSubscriptionOptions{
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
		probability: 0.5,
		minInterval: time.Millisecond * 12000,
		perform: func(client *Client, data *clientData) bool {
			if len(data.createdChannelIds) == 0 {
				return false
			}
			channelToRemove := randUint32(data.createdChannelIds)
			client.Send(0,
				proto.BroadcastType_NO,
				uint32(proto.MessageType_REMOVE_CHANNEL),
				&proto.RemoveChannelMessage{
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
		probability: 0.5,
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
				client.Send(channelIdToSub, proto.BroadcastType_NO, uint32(proto.MessageType_SUB_TO_CHANNEL), &proto.SubscribedToChannelMessage{
					ConnId: client.Id,
				}, nil)
				//log.Printf("Client(%d) SUB_TO_CHANNEL: %d", client.Id, channelIdToSub)
				return true
			}
			return false
		},
	},
	{
		name:        "unsubToChannel",
		probability: 0.5,
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

			client.Send(channelIdToUnsub, proto.BroadcastType_NO, uint32(proto.MessageType_UNSUB_TO_CHANNEL), &proto.UnsubscribedToChannelMessage{
				ConnId: client.Id,
			}, nil)
			//log.Printf("Client(%d) UNSUB_TO_CHANNEL: %d", client.Id, channelIdToUnsub)
			return true
		},
	},
	{
		name:        "sendChatMessage",
		probability: 1,
		minInterval: time.Millisecond * 1000,
		perform: func(client *Client, data *clientData) bool {
			if data.activeChannelId == 0 {
				return false
			}

			dataUpdate, _ := anypb.New(&proto.ChatChannelData{
				ChatMessages: []*proto.ChatMessage{{
					Sender:   fmt.Sprintf("Client%d", client.Id),
					SendTime: time.Now().Unix(),
					Content:  "How are you?",
				}},
			})
			client.Send(data.activeChannelId, proto.BroadcastType_NO, uint32(proto.MessageType_CHANNEL_DATA_UPDATE),
				&proto.ChannelDataUpdateMessage{
					Data: dataUpdate,
				}, nil)
			return true
		},
	},
}

func removeChannelId(client *Client, data *clientData, channelId uint32) {
	if data.activeChannelId == channelId {
		if len(client.subscribedChannels) > 0 {
			data.activeChannelId = randUint32(client.subscribedChannels)
		} else {
			data.activeChannelId = 0
		}
	}

	for id := range data.createdChannelIds {
		if id == channelId {
			delete(data.createdChannelIds, id)
			break
		}
	}

	for id := range data.listedChannels {
		if id == channelId {
			delete(data.listedChannels, id)
			break
		}
	}
}

func runClient() {
	defer wg.Done()
	c, err := NewClient("ws://localhost:12108")
	if err != nil {
		log.Println(err)
		return
	} else {
		log.Println("Connected from " + c.conn.LocalAddr().String())
	}
	defer c.Disconnect()

	go func() {
		for {
			c.Receive()
		}
	}()

	r := c.Auth("test", "test")
	if (<-r).Result != proto.AuthResultMessage_SUCCESSFUL {
		return
	}

	time.Sleep(100 * time.Millisecond)

	data := &clientData{
		clientId:          c.Id,
		rnd:               rand.New(rand.NewSource(time.Now().Unix())),
		activeChannelId:   0,
		createdChannelIds: make(map[uint32]struct{}),
	}

	c.AddMessageHandler(uint32(proto.MessageType_SUB_TO_CHANNEL), func(client *Client, channelId uint32, m Message) {
		data.activeChannelId = channelId
	})
	c.AddMessageHandler(uint32(proto.MessageType_UNSUB_TO_CHANNEL), func(client *Client, channelId uint32, m Message) {
		msg := m.(*proto.UnsubscribedToChannelMessage)
		if msg.ConnId != client.Id {
			return
		}
		removeChannelId(client, data, channelId)
	})
	c.AddMessageHandler(uint32(proto.MessageType_REMOVE_CHANNEL), func(client *Client, channelId uint32, m Message) {
		msg := m.(*proto.RemoveChannelMessage)
		removeChannelId(client, data, msg.ChannelId)
	})

	actionInstances := map[*clientAction]*struct{ time.Time }{}
	for t := time.Now(); time.Since(t) < 60*time.Second; {
		tickStartTime := time.Now()

		var action *clientAction
		actions := make([]*clientAction, 0)
		var probSum float32 = 0
		for _, a := range clientActions {
			i, exists := actionInstances[a]
			if !exists {
				i = &struct{ time.Time }{time.Now()}
				actionInstances[a] = i
				actions = append(actions, a)
				probSum += a.probability
			} else {
				if time.Since(i.Time) >= a.minInterval {
					actions = append(actions, a)
					probSum += a.probability
				}
			}
		}
		probabilities := make([]float32, len(actions))
		prob := data.rnd.Float32()
		for i, a := range actions {
			probabilities[i] = a.probability / probSum
			prob -= probabilities[i]
			if prob <= 0 {
				action = a
				break
			}
		}

		if action != nil {
			if action.perform(c, data) {
				actionInstances[action].Time = time.Now()
			}
		}

		// Max tick interval: 200ms
		time.Sleep(50*time.Millisecond - time.Since(tickStartTime))
	}

	c.Disconnect()
}

func randUint32(m map[uint32]struct{}) uint32 {
	// The order of iterating a map is random:
	// https://medium.com/i0exception/map-iteration-in-go-275abb76f721
	for k := range m {
		return k
	}
	return 0
}

var wg = sync.WaitGroup{}

func main() {

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go runClient()
		time.Sleep(200 * time.Millisecond)
	}
	//time.Sleep(60 * time.Second)
	wg.Wait()
}
