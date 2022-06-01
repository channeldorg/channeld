package main

import (
	"fmt"
	"net/http"

	"channeld.clewcat.com/channeld/examples/unity-mirror-tanks/tankspb"
	"channeld.clewcat.com/channeld/pkg/channeld"
	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	if err := channeld.GlobalSettings.ParseFlag(); err != nil {
		fmt.Printf("error parsing CLI flag: %v\n", err)
	}
	channeld.StartProfiling()
	channeld.InitLogsAndMetrics()
	channeld.InitConnections(channeld.GlobalSettings.ServerFSM, channeld.GlobalSettings.ClientFSM)
	channeld.InitChannels()

	channeld.RegisterChannelDataType(channeldpb.ChannelType_SUBWORLD, &tankspb.TankGameChannelData{})

	channeld.InitSpatialController(&channeld.StaticGrid2DSpatialController{
		// WorldOffsetX: -40,
		// WorldOffsetZ: -40,
		// GridWidth:    8,
		// GridHeight:   8,
		// GridCols:     10,
		// GridRows:     10,
		WorldOffsetX:             -5,
		WorldOffsetZ:             -5,
		GridWidth:                5,
		GridHeight:               5,
		GridCols:                 2,
		GridRows:                 2,
		ServerCols:               1,
		ServerRows:               2,
		ServerInterestBorderSize: 0,
	})

	// Setup Prometheus
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":8080", nil)

	go channeld.StartListening(channeldpb.ConnectionType_SERVER, channeld.GlobalSettings.ServerNetwork, channeld.GlobalSettings.ServerAddress)
	// FIXME: After all the server connections are established, the client connection should be listened.*/
	channeld.StartListening(channeldpb.ConnectionType_CLIENT, channeld.GlobalSettings.ClientNetwork, channeld.GlobalSettings.ClientAddress)

}
