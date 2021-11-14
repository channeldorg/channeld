package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"channeld.clewcat.com/channeld/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type clientData struct {
	activeChannelId uint32
	channelList     []*proto.ListChannelResultMessage_ChannelInfo
}

var clientDataMap = make(map[*Client]*clientData)

type clientAction struct {
	probability float32
	minInterval time.Duration
	perform     func(client *Client) bool
}

var clientActions = []*clientAction{
	{
		probability: 1,
		minInterval: time.Millisecond * 3000,
		perform: func(client *Client) bool {
			client.Send(0, proto.BroadcastType_NO, uint32(proto.MessageType_LIST_CHANNEL), &proto.ListChannelMessage{},
				func(c *Client, channelId uint32, m Message) {
					clientDataMap[c].channelList = m.(*proto.ListChannelResultMessage).Channels
				})
			return true
		},
	},
	{
		probability: 0.5,
		minInterval: time.Millisecond * 10000,
		perform: func(client *Client) bool {
			client.Send(0, proto.BroadcastType_NO, uint32(proto.MessageType_CREATE_CHANNEL), &proto.CreateChannelMessage{
				ChannelType: proto.ChannelType_SUBWORLD,
				Metadata:    fmt.Sprintf("Room%d", rand.Uint32()),
				SubOptions: &proto.ChannelSubscriptionOptions{
					CanUpdateData:    true,
					DataFieldMasks:   make([]string, 0),
					FanOutIntervalMs: 100,
				},
			}, nil)
			//log.Printf("Client(%d) CREATE_CHANNEL", client.Id)
			return true
		},
	},
	{
		probability: 0.5,
		minInterval: time.Millisecond * 3000,
		perform: func(client *Client) bool {
			if list := clientDataMap[client].channelList; len(list) > 1 {
				cm := make(map[uint32]uint32)
				for _, ci := range list {
					cm[ci.ChannelId] = ci.ChannelId
				}
				for _, chid := range client.subscribedChannels {
					delete(cm, chid)
				}
				var channelIdToSub uint32
				// The order of iterating a map is random:
				// https://medium.com/i0exception/map-iteration-in-go-275abb76f721
				for k, _ := range cm {
					channelIdToSub = k
					break
				}
				client.Send(channelIdToSub, proto.BroadcastType_NO, uint32(proto.MessageType_SUB_TO_CHANNEL), &proto.SubscribedToChannelMessage{
					ConnId: client.Id,
				}, func(client *Client, channelId uint32, m Message) {
					clientDataMap[client].activeChannelId = channelId
				})
				//log.Printf("Client(%d) SUB_TO_CHANNEL: %d", client.Id, channelIdToSub)
				return true
			}
			return false
		},
	},
	{
		probability: 0.5,
		minInterval: time.Millisecond * 3000,
		perform: func(client *Client) bool {
			if len(client.subscribedChannels) <= 1 {
				return false
			}
			data := clientDataMap[client]
			channelIdToUnsub := data.activeChannelId
			// Only unsubscribe from an inactive channel
			for channelIdToUnsub == data.activeChannelId {
				channelIdToUnsub = client.subscribedChannels[rand.Intn(len(client.subscribedChannels))]
			}
			client.Send(channelIdToUnsub, proto.BroadcastType_NO, uint32(proto.MessageType_UNSUB_TO_CHANNEL), &proto.UnsubscribedToChannelMessage{
				ConnId: client.Id,
			}, nil)
			//log.Printf("Client(%d) UNSUB_TO_CHANNEL: %d", client.Id, channelIdToUnsub)
			return true
		},
	},
	{
		probability: 1,
		minInterval: time.Millisecond * 1000,
		perform: func(client *Client) bool {
			dataUpdate, _ := anypb.New(&proto.ChatChannelData{
				ChatMessages: []*proto.ChatMessage{{
					Sender:   fmt.Sprintf("Client%d", client.Id),
					SendTime: time.Now().Unix(),
					Content:  "How are you?",
				}},
			})
			client.Send(clientDataMap[client].activeChannelId, proto.BroadcastType_NO, uint32(proto.MessageType_CHANNEL_DATA_UPDATE),
				&proto.ChannelDataUpdateMessage{
					Data: dataUpdate,
				}, nil)
			return true
		},
	},
}

func runClient() {
	c, err := NewClient("ws://localhost:12108")
	if err != nil {
		log.Panic(err)
	} else {
		log.Println("Connected from " + c.conn.LocalAddr().String())
	}
	defer c.Disconnect()

	go func() {
		for {
			c.Receive()
		}
	}()

	result := c.Auth("test", "test")
	if (<-result).Result != proto.AuthResultMessage_SUCCESSFUL {
		return
	}

	clientDataMap[c] = &clientData{}

	actionInstances := map[*clientAction]*struct{ time.Time }{}
	for {
		if c.Id == 0 {
			time.Sleep(100 * time.Millisecond)
			continue
		}

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
		rnd := rand.Float32()
		for i, a := range actions {
			probabilities[i] = a.probability / probSum
			rnd -= probabilities[i]
			if rnd <= 0 {
				action = a
			}
		}

		if action != nil {
			if action.perform(c) {
				actionInstances[action].Time = time.Now()
			}
		}

		// Max tick interval: 200ms
		time.Sleep(200*time.Millisecond - time.Since(tickStartTime))
	}
	/*
		ticker := time.NewTimer(1000 * time.Millisecond)
		done := make(chan struct{})
		interrupt := make(chan os.Signal, 1)

		for {
			select {
			case <-done:
				return
			case <-interrupt:
				log.Println("interrupt")

				// Cleanly close the connection by sending a close message and then
				// waiting (with timeout) for the server to close the connection.
				err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				if err != nil {
					log.Println("write close:", err)
					return
				}
				select {
				case <-done:
				case <-time.After(time.Second):
				}
				return
			}
		}
	*/

	//wg.Done()
}

//var wg = sync.WaitGroup{}

func main() {

	for i := 0; i < 100; i++ {
		//wg.Add(1)
		go runClient()
		time.Sleep(200 * time.Millisecond)
	}
	time.Sleep(60 * time.Second)
	//wg.Wait()
}
