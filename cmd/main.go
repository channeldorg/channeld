package main

import (
	"flag"

	"channeld.clewcat.com/channeld/pkg/channeld"
	"rsc.io/getopt"
)

func main() {

	getopt.Aliases(
		"sn", "serverNetwork",
		"sa", "serverAddress",
		"sfsm", "serverConnFSM",

		"cn", "clientNetwork",
		"ca", "clientAddress",
		"cfsm", "clientConnFSM",

		"cs", "connSize",
	)

	sn := flag.String("sn", "tcp", "the network type for the server connections")
	sa := flag.String("sa", ":11288", "the network address for the server connections")
	sfsm := flag.String("sfsm", "../config/server_authoratative_fsm.json", "the path to the server FSM config")
	cn := flag.String("cn", "tcp", "the network type for the client connections")
	ca := flag.String("ca", ":12108", "the network address for the client connections")
	cfsm := flag.String("cfsm", "../config/client_non_authoratative_fsm.json", "the path to the client FSM config")
	cs := flag.Int("cs", 1024, "the connection map buffer size")

	getopt.Parse()
	flag.Parse()

	channeld.InitConnections(*cs, *sfsm, *cfsm)
	channeld.InitChannels()

	go channeld.StartListening(channeld.SERVER, *sn, *sa)
	///* After all the server connections established, the client connection will be listened.*/
	go channeld.StartListening(channeld.CLIENT, *cn, *ca)

}
