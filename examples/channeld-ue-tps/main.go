package main

import (
	"fmt"
	"net/http"

	"channeld.clewcat.com/channeld/examples/channeld-ue-tps/tpspb"
	"channeld.clewcat.com/channeld/pkg/channeld"
	"channeld.clewcat.com/channeld/pkg/channeldpb"
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

	channeld.RegisterChannelDataType(channeldpb.ChannelType_GLOBAL, &tpspb.GlobalChannelData{})
	channeld.RegisterChannelDataType(channeldpb.ChannelType_SUBWORLD, &tpspb.TestRepChannelData{})
	channeld.RegisterChannelDataType(channeldpb.ChannelType_SPATIAL, &tpspb.TestRepChannelData{})

	channeld.InitSpatialController(&channeld.StaticGrid2DSpatialController{
		// WorldOffsetX: -40,
		// WorldOffsetZ: -40,
		// GridWidth:    8,
		// GridHeight:   8,
		// GridCols:     10,
		// GridRows:     10,

		WorldOffsetX: -1000,
		WorldOffsetZ: -1000,
		GridWidth:    1000,
		GridHeight:   1000,
		GridCols:     2,
		GridRows:     2,
		ServerCols:   1,
		ServerRows:   2,
		// GridWidth:                10,
		// GridHeight:               10,
		// GridCols:                 1,
		// GridRows:                 1,
		// ServerCols:               1,
		// ServerRows:               1,
		ServerInterestBorderSize: 0})

	channeld.RegisterMessageHandler(uint32(unrealpb.MessageType_SPAWN), &channeldpb.ServerForwardMessage{}, tpspb.HandleUnrealSpawnObject)
	channeld.RegisterMessageHandler(uint32(unrealpb.MessageType_HANDOVER_CONTEXT), &unrealpb.GetHandoverContextResultMessage{}, tpspb.HandleHandoverContextResult)

	// Setup Prometheus
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":8080", nil)

	go channeld.StartListening(channeldpb.ConnectionType_SERVER, channeld.GlobalSettings.ServerNetwork, channeld.GlobalSettings.ServerAddress)
	// FIXME: After all the server connections are established, the client connection should be listened.*/
	channeld.StartListening(channeldpb.ConnectionType_CLIENT, channeld.GlobalSettings.ClientNetwork, channeld.GlobalSettings.ClientAddress)

}
