package main

import (
	"fmt"
	"net/http"

	"github.com/metaworking/channeld/pkg/channeld"
	"github.com/metaworking/channeld/pkg/channeldpb"
	"github.com/metaworking/channeld/pkg/unreal"
	"github.com/metaworking/channeld/pkg/unrealpb"
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
	channeld.SetChannelDataExtension[unreal.RecoverableChannelDataExtension](channeldpb.ChannelType_GLOBAL)
	channeld.SetChannelDataExtension[unreal.RecoverableChannelDataExtension](channeldpb.ChannelType_SUBWORLD)
	channeld.InitChannels()

	channeld.InitSpatialController()

	unreal.InitMessageHandlers()
	channeld.RegisterChannelDataType(channeldpb.ChannelType_SPATIAL, &unrealpb.SpatialChannelData{})

	// Setup Prometheus
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":8080", nil)

	go channeld.StartListening(channeldpb.ConnectionType_SERVER, channeld.GlobalSettings.ServerNetwork, channeld.GlobalSettings.ServerAddress)

	if channeld.GlobalSettings.ClientNetworkWaitMasterServer {
		// After the Master server owned the GLOBAL channel, the client connection should be listened.
		<-channeld.Event_GlobalChannelPossessed.Wait()
	}
	channeld.StartListening(channeldpb.ConnectionType_CLIENT, channeld.GlobalSettings.ClientNetwork, channeld.GlobalSettings.ClientAddress)
}
