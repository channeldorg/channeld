package main

import (
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"channeld.clewcat.com/channeld/proto"
)

var ServerAddr string = "ws://localhost:12108" //"49.234.9.192:12108" //"ws://49.234.9.192:12108"

const (
	ClientNum                int           = 500
	MaxChannelNum            int           = 0
	RunDuration              time.Duration = 120 * time.Second
	ConnectInterval          time.Duration = 100 * time.Millisecond
	MaxTickInterval          time.Duration = 100 * time.Millisecond
	ActionIntervalMultiplier float64       = 0.2
)

type clientData struct {
	clientId          uint32
	rnd               *rand.Rand
	activeChannelId   uint32
	createdChannelIds map[uint32]struct{}
	listedChannels    map[uint32]struct{}
	ctx               map[interface{}]interface{}
}

type clientAction struct {
	name        string
	probability float32
	minInterval time.Duration
	perform     func(client *Client, data *clientData) bool
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

func runClient(clientActions []*clientAction, initFunc func(client *Client, data *clientData)) {
	defer wg.Done()
	c, err := NewClient(ServerAddr)
	if err != nil {
		log.Println(err)
		return
	} else {
		log.Println("Connected from " + c.conn.LocalAddr().String())
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
		clientId:          c.Id,
		rnd:               rand.New(rand.NewSource(time.Now().Unix())),
		activeChannelId:   0,
		createdChannelIds: make(map[uint32]struct{}),
		ctx:               make(map[interface{}]interface{}),
	}

	c.AddMessageHandler(uint32(proto.MessageType_SUB_TO_CHANNEL), func(client *Client, channelId uint32, m Message) {
		data.activeChannelId = channelId
	})
	c.AddMessageHandler(uint32(proto.MessageType_UNSUB_FROM_CHANNEL), func(client *Client, channelId uint32, m Message) {
		msg := m.(*proto.UnsubscribedFromChannelResultMessage)
		if msg.ConnId != client.Id {
			return
		}
		removeChannelId(client, data, channelId)
	})
	c.AddMessageHandler(uint32(proto.MessageType_REMOVE_CHANNEL), func(client *Client, channelId uint32, m Message) {
		msg := m.(*proto.RemoveChannelMessage)
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
