package main

import (
	"fmt"
	"net/http"

	"channeld.clewcat.com/channeld/pkg/channeld"
	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {

	/*
		getopt.Aliases(
			"sn", "serverNetwork",
			"sa", "serverAddress",
			"sfsm", "serverConnFSM",

			"cn", "clientNetwork",
			"ca", "clientAddress",
			"cfsm", "clientConnFSM",

			// "cs", "connSize",
		)

		sn := flag.String("sn", "tcp", "the network type for the server connections")
		sa := flag.String("sa", ":11288", "the network address for the server connections")
		sfsm := flag.String("sfsm", "../config/server_authoratative_fsm.json", "the path to the server FSM config")
		cn := flag.String("cn", "tcp", "the network type for the client connections")
		ca := flag.String("ca", ":12108", "the network address for the client connections")
		cfsm := flag.String("cfsm", "../config/client_non_authoratative_fsm.json", "the path to the client FSM config")
		ct := flag.Uint("ct", 0, "The compression type, 0 = No, 1 = Snappy")

		//getopt.Parse()
		flag.Parse()
	*/

	if err := channeld.GlobalSettings.ParseFlag(); err != nil {
		fmt.Printf("error parsing CLI flag: %v\n", err)
	}
	channeld.StartProfiling()
	channeld.InitLogs()
	channeld.InitMetrics()
	channeld.InitConnections(channeld.GlobalSettings.ServerFSM, channeld.GlobalSettings.ClientFSM)
	channeld.InitChannels()

	// Setup Prometheus
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":8080", nil)

	go channeld.StartListening(channeldpb.ConnectionType_SERVER, channeld.GlobalSettings.ServerNetwork, channeld.GlobalSettings.ServerAddress)
	// FIXME: After all the server connections are established, the client connection should be listened.*/
	channeld.StartListening(channeldpb.ConnectionType_CLIENT, channeld.GlobalSettings.ClientNetwork, channeld.GlobalSettings.ClientAddress)

}
