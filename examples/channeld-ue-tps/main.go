package main

import (
	"fmt"
	"net/http"

	"channeld.clewcat.com/channeld/examples/channeld-ue-tps/tpspb"
	"channeld.clewcat.com/channeld/pkg/channeld"
	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"channeld.clewcat.com/channeld/pkg/common"
	"channeld.clewcat.com/channeld/pkg/unrealpb"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	if err := channeld.GlobalSettings.ParseFlag(); err != nil {
		fmt.Printf("error parsing CLI flag: %v\n", err)
	}
	channeld.StartProfiling()
	channeld.InitLogs()
	channeld.InitMetrics()
	channeld.InitConnections(channeld.GlobalSettings.ServerFSM, channeld.GlobalSettings.ClientFSM)
	channeld.InitChannels()

	channeld.RegisterChannelDataType(channeldpb.ChannelType_GLOBAL, &tpspb.TestRepChannelData{})
	channeld.RegisterChannelDataType(channeldpb.ChannelType_SUBWORLD, &tpspb.TestRepChannelData{})
	channeld.RegisterChannelDataType(channeldpb.ChannelType_SPATIAL, &tpspb.TestRepChannelData{})

	channeld.InitSpatialController()

	channeld.RegisterMessageHandler(uint32(unrealpb.MessageType_SPAWN), &channeldpb.ServerForwardMessage{}, tpspb.HandleUnrealSpawnObject)
	channeld.RegisterMessageHandler(uint32(unrealpb.MessageType_HANDOVER_CONTEXT), &unrealpb.GetHandoverContextResultMessage{}, tpspb.HandleHandoverContextResult)
	channeld.RegisterMessageHandler(uint32(unrealpb.MessageType_GET_UNREAL_OBJECT_REF), &unrealpb.GetUnrealObjectRefMessage{}, tpspb.HandleGetUnrealObjectRef)

	channeld.Event_GlobalChannelUnpossessed.Listen(func(struct{}) {
		// Global server exits. Clear up all the cache.
		tpspb.AllSpawnedObj = make(map[uint32]*unrealpb.UnrealObjectRef)
		tpspb.HandoverDataProviders = make(map[uint64]chan common.Message)
	})

	// Setup Prometheus
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":8080", nil)

	go channeld.StartListening(channeldpb.ConnectionType_SERVER, channeld.GlobalSettings.ServerNetwork, channeld.GlobalSettings.ServerAddress)

	// After the Master server owned the GLOBAL channel, the client connection should be listened.*/
	<-channeld.Event_GlobalChannelPossessed.Wait()
	channeld.StartListening(channeldpb.ConnectionType_CLIENT, channeld.GlobalSettings.ClientNetwork, channeld.GlobalSettings.ClientAddress)
}
