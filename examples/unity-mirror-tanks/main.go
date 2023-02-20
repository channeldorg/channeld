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
	channeld.InitLogs()
	channeld.InitMetrics()
	channeld.InitConnections(channeld.GlobalSettings.ServerFSM, channeld.GlobalSettings.ClientFSM)
	channeld.InitChannels()

	channeld.RegisterChannelDataType(channeldpb.ChannelType_SUBWORLD, &tankspb.TankGameChannelData{})

	channeld.InitSpatialController()

	// Setup Prometheus
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":8080", nil)

	go channeld.StartListening(channeldpb.ConnectionType_SERVER, channeld.GlobalSettings.ServerNetwork, channeld.GlobalSettings.ServerAddress)
	// FIXME: After all the server connections are established, the client connection should be listened.*/
	channeld.StartListening(channeldpb.ConnectionType_CLIENT, channeld.GlobalSettings.ClientNetwork, channeld.GlobalSettings.ClientAddress)

}
