package main

import (
	"log"
	"sync"
	"time"

	"channeld.clewcat.com/channeld/proto"
	protobuf "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// var ctxKeyTanksChannelData struct{}
// var ctxKeyClientNetId struct{}

const (
	ctxKeyTanksChannelData = 1
	ctxKeyClientNetId      = 2
)

func TanksInitFunc(client *Client, data *clientData) {
	data.ctx[ctxKeyTanksChannelData] = &proto.TankGameChannelData{}
	client.AddMessageHandler(uint32(proto.MessageType_AUTH), func(c *Client, channelId uint32, m Message) {
		resultMsg := m.(*proto.AuthResultMessage)
		if resultMsg.Result == proto.AuthResultMessage_SUCCESSFUL {
			// Re-sub to GLOBAL channel to update the sub options
			client.Send(0, proto.BroadcastType_NO_BROADCAST, uint32(proto.MessageType_SUB_TO_CHANNEL), &proto.SubscribedToChannelMessage{
				ConnId: resultMsg.ConnId,
				SubOptions: &proto.ChannelSubscriptionOptions{
					CanUpdateData:    true,
					FanOutIntervalMs: 10,
					DataFieldMasks:   []string{},
				},
			}, nil)
		}
	})
	client.AddMessageHandler(uint32(proto.MessageType_CHANNEL_DATA_UPDATE), wrapTanksChannelDataUpateHandle(data))
}

var TanksClientActions = []*clientAction{
	{
		name:        "updateTankTransform",
		probability: 1,
		minInterval: time.Millisecond * 200,
		perform: func(client *Client, data *clientData) bool {
			tanksChannelData, ok := data.ctx[ctxKeyTanksChannelData].(*proto.TankGameChannelData)
			if !ok {
				return false
			}

			netId, ok := data.ctx[ctxKeyClientNetId].(uint32)
			if !ok {
				log.Println("netId is not set in the ctx!")
				return true
			}

			transform, exists := tanksChannelData.TransformStates[netId]
			if !exists {
				return false
			}
			if transform.Removed {
				delete(tanksChannelData.TransformStates, netId)
				return true
			}

			transform.Rotation.Y += data.rnd.Float32() * 0.1
			/*
				if transform.Position == nil {
					log.Println("transform.position is not initialized yet!")
					return false
				}
				pos := transform.Position
				pos.Z += 0.1
				log.Printf("updating transform (netId=%d) to %s\n", netId, pos.String())
			*/

			any, err := anypb.New(&proto.TankGameChannelData{
				TransformStates: map[uint32]*proto.TransformState{
					netId: {
						//Position: transform.Position,
						Rotation: transform.Rotation,
					},
				},
			})
			if err != nil {
				log.Println(err)
				return false
			}
			client.Send(0, proto.BroadcastType_NO_BROADCAST, uint32(proto.MessageType_CHANNEL_DATA_UPDATE), &proto.ChannelDataUpdateMessage{
				Data: any,
			}, nil)

			return true
		},
	},
	{
		name:        "fire",
		probability: 0,
		minInterval: time.Millisecond * 200,
		perform: func(client *Client, data *clientData) bool {

			return true
		},
	},
	{
		name:        "ping",
		probability: 0,
		minInterval: time.Millisecond * 200,
		perform: func(client *Client, data *clientData) bool {

			return true
		},
	},
}

// netId -> connId
var tanksNetIdMapping sync.Map

func wrapTanksChannelDataUpateHandle(data *clientData) MessageHandlerFunc {
	return func(client *Client, channelId uint32, m Message) {
		tanksChannelData, ok := data.ctx[ctxKeyTanksChannelData].(*proto.TankGameChannelData)
		if !ok {
			log.Println("tanksChannelData is not initialized in the ctx!")
			return
		}
		updateMsg, _ := m.(*proto.ChannelDataUpdateMessage)
		var channelData proto.TankGameChannelData
		updateMsg.Data.UnmarshalTo(&channelData)
		protobuf.Merge(tanksChannelData, &channelData)

		if _, exists := data.ctx[ctxKeyClientNetId].(uint32); exists {
			//log.Printf("received transform (netId=%d): %s\n", netId, tanksChannelData.TransformStates[netId])
			return
		}

		// Randomly pick a tank that is not taken by other client (netId is not registered in tanksNetIdMapping)
		for netId := range tanksChannelData.TankStates {
			if _, ok := tanksNetIdMapping.Load(netId); !ok {
				tanksNetIdMapping.Store(netId, client.Id)
				data.ctx[ctxKeyClientNetId] = netId
				break
			}
		}
	}
}
