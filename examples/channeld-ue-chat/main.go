package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"channeld.clewcat.com/channeld/examples/channeld-ue-chat/chatpb"
	"channeld.clewcat.com/channeld/pkg/channeld"
	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"channeld.clewcat.com/channeld/pkg/client"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/protobuf/types/known/anypb"
)

func runMasterServer() {
	c, err := client.NewClient("127.0.0.1:11288")
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
				globalInitData := &anypb.Any{}
				globalInitData.MarshalFrom(&chatpb.ChatChannelData{})
				c.Send(uint32(channeld.GlobalChannelId), channeldpb.BroadcastType_NO_BROADCAST, uint32(channeldpb.MessageType_CREATE_CHANNEL), &channeldpb.CreateChannelMessage{
					ChannelType: channeldpb.ChannelType_GLOBAL,
					Data:        globalInitData,
				}, nil)
			} else {
				log.Panicln("master server failed to auth")
			}
		} else {
			// Handle auth result of other connections
			log.Printf("master server received auth result of conn %d: %s\n", resultMsg.ConnId, channeldpb.AuthResultMessage_AuthResult_name[int32(resultMsg.Result)])
			if resultMsg.Result == channeldpb.AuthResultMessage_SUCCESSFUL {
				c.Send(uint32(channeld.GlobalChannelId), channeldpb.BroadcastType_NO_BROADCAST, uint32(channeldpb.MessageType_SUB_TO_CHANNEL), &channeldpb.SubscribedToChannelMessage{
					ConnId: resultMsg.ConnId,
				}, nil)
			}
		}
	})

	c.Auth("test_lt", "test_pit")

	exited := false
	for !exited {
		c.Tick()
		time.Sleep(time.Millisecond * 100)
	}
}

func main() {
	if err := channeld.GlobalSettings.ParseFlag(); err != nil {
		fmt.Printf("error parsing CLI flag: %v\n", err)
	}
	channeld.StartProfiling()
	channeld.InitLogs()
	channeld.InitMetrics()
	channeld.InitConnections("../../config/server_authoratative_fsm.json", "../../config/client_authoratative_fsm.json")
	channeld.InitChannels()
	channeld.GetChannel(channeld.GlobalChannelId).InitData(
		&chatpb.ChatChannelData{ChatMessages: []*chatpb.ChatMessage{
			{Sender: "System", SendTime: time.Now().Unix(), Content: "Welcome!"},
		}},
		&channeldpb.ChannelDataMergeOptions{
			ListSizeLimit: 10,
			TruncateTop:   true,
		},
	)

	channeld.RegisterChannelDataType(channeldpb.ChannelType_SUBWORLD, &chatpb.ChatChannelData{})

	SetupAuth()

	// Setup Prometheus
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":8080", nil)

	go runMasterServer()

	go channeld.StartListening(channeldpb.ConnectionType_SERVER, channeld.GlobalSettings.ServerNetwork, channeld.GlobalSettings.ServerAddress)
	// FIXME: After all the server connections are established, the client connection should be listened.*/
	channeld.StartListening(channeldpb.ConnectionType_CLIENT, channeld.GlobalSettings.ClientNetwork, channeld.GlobalSettings.ClientAddress)

}
