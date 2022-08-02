package main

import (
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"channeld.clewcat.com/channeld/pkg/client"
)

var ServerAddr string = "ws://localhost:12108" //"49.234.9.192:12108" //"ws://49.234.9.192:12108"

const (
	ClientNum                int           = 500
	MaxChannelNum            int           = 100
	RunDuration              time.Duration = 120 * time.Second
	ConnectInterval          time.Duration = 100 * time.Millisecond
	MaxTickInterval          time.Duration = 100 * time.Millisecond
	ActionIntervalMultiplier float64       = 0.2
	SubToGlobalAfterAuth     bool          = true
)

type clientData struct {
	clientId        uint32
	rnd             *rand.Rand
	activeChannelId uint32
	ctx             map[interface{}]interface{}
}

type clientAction struct {
	name        string
	probability float32
	minInterval time.Duration
	perform     func(client *client.ChanneldClient, data *clientData) bool
}

func removeChannelId(client *client.ChanneldClient, data *clientData, channelId uint32) {
	if data.activeChannelId == channelId {
		if len(client.SubscribedChannels) > 0 {
			data.activeChannelId = randUint32(client.SubscribedChannels)
		} else {
			data.activeChannelId = 0
		}
	}
}

func runClient(clientActions []*clientAction, initFunc func(client *client.ChanneldClient, data *clientData)) {
	defer wg.Done()
	c, err := client.NewClient(ServerAddr)
	if err != nil {
		log.Println(err)
		return
	} else {
		log.Println("Connected from " + c.Conn.LocalAddr().String())
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

	c.Auth("test", "test")

	time.Sleep(100 * time.Millisecond)

	data := &clientData{
		clientId:        c.Id,
		rnd:             rand.New(rand.NewSource(time.Now().Unix())),
		activeChannelId: 0,
		ctx:             make(map[interface{}]interface{}),
	}

	if SubToGlobalAfterAuth {
		c.AddMessageHandler(uint32(channeldpb.MessageType_AUTH), func(c *client.ChanneldClient, channelId uint32, m client.Message) {
			resultMsg := m.(*channeldpb.AuthResultMessage)
			if resultMsg.Result == channeldpb.AuthResultMessage_SUCCESSFUL {
				c.Send(0, channeldpb.BroadcastType_NO_BROADCAST, uint32(channeldpb.MessageType_SUB_TO_CHANNEL), &channeldpb.SubscribedToChannelMessage{
					ConnId: resultMsg.ConnId,
					SubOptions: &channeldpb.ChannelSubscriptionOptions{
						DataAccess:       channeldpb.ChannelDataAccess_WRITE_ACCESS,
						FanOutIntervalMs: 10,
						DataFieldMasks:   []string{},
					},
				}, nil)
			}
		})
	}

	c.AddMessageHandler(uint32(channeldpb.MessageType_SUB_TO_CHANNEL), func(client *client.ChanneldClient, channelId uint32, m client.Message) {
		data.activeChannelId = channelId
	})
	c.AddMessageHandler(uint32(channeldpb.MessageType_UNSUB_FROM_CHANNEL), func(client *client.ChanneldClient, channelId uint32, m client.Message) {
		msg := m.(*channeldpb.UnsubscribedFromChannelResultMessage)
		if msg.ConnId != client.Id {
			return
		}
		removeChannelId(client, data, channelId)
	})
	c.AddMessageHandler(uint32(channeldpb.MessageType_REMOVE_CHANNEL), func(client *client.ChanneldClient, channelId uint32, m client.Message) {
		msg := m.(*channeldpb.RemoveChannelMessage)
		removeChannelId(client, data, msg.ChannelId)
	})

	if initFunc != nil {
		initFunc(c, data)
	}

	actionInstances := map[*clientAction]*struct{ time.Time }{}
	for t := time.Now(); time.Since(t) < RunDuration && c.IsConnected(); {
		tickStartTime := time.Now()

		c.Tick()

		// Authenticated
		if c.Id > 0 {
			var actionToPerform *clientAction
			actions := make([]*clientAction, 0)
			var probSum float32 = 0
			for _, action := range clientActions {
				instance, exists := actionInstances[action]
				if !exists {
					instance = &struct{ time.Time }{time.Now()}
					actionInstances[action] = instance
					actions = append(actions, action)
					probSum += action.probability
				} else {
					if time.Since(instance.Time) >= time.Duration(float64(action.minInterval)*ActionIntervalMultiplier) {
						actions = append(actions, action)
						probSum += action.probability
					}
				}
			}
			probabilities := make([]float32, len(actions))
			prob := data.rnd.Float32()
			for i, a := range actions {
				probabilities[i] = a.probability / probSum
				prob -= probabilities[i]
				if prob <= 0 {
					actionToPerform = a
					break
				}
			}

			if actionToPerform != nil {
				if actionToPerform.perform(c, data) {
					actionInstances[actionToPerform].Time = time.Now()
				}
			}
		}

		time.Sleep(MaxTickInterval - time.Since(tickStartTime))
	}

	if !c.IsConnected() {
		log.Printf("client %d is disconnected by the server.\n", c.Id)
	} else {
		c.Disconnect()
	}
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
	if len(os.Args) > 1 {
		ServerAddr = os.Args[1]
	}
	for i := 0; i < ClientNum; i++ {
		wg.Add(1)
		//go runClient(TanksClientActions, TanksInitFunc)
		go runClient(ChatClientActions, nil)
		time.Sleep(ConnectInterval)
	}

	wg.Wait()
}
