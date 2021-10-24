package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"channeld.clewcat.com/channeld/pkg/channeld"
	"channeld.clewcat.com/channeld/proto"
)

var webAddr = flag.String("web", "localhost:8080", "http service address")
var wsAddr = flag.String("ws", "localhost:12108", "websocket service address")

func handleChanneldProto(w http.ResponseWriter, r *http.Request) {
	bytes, err := os.ReadFile("../../proto/channeld.proto")
	if err != nil {
		log.Panic(err)
	}
	w.Write(bytes)
}

func handleChatProto(w http.ResponseWriter, r *http.Request) {
	bytes, err := os.ReadFile("../../proto/example_chat_rooms.proto")
	if err != nil {
		log.Panic(err)
	}
	w.Write(bytes)
}

func handleMain(w http.ResponseWriter, r *http.Request) {
	homeTemplate, err := template.ParseFiles("./main.html")
	if err != nil {
		log.Panic(err)
	}

	homeTemplate.Execute(w, *wsAddr)
}

func main() {

	http.HandleFunc("/", handleMain)
	http.HandleFunc("/proto", handleChanneldProto)
	http.HandleFunc("/proto/chat", handleChatProto)

	channeld.InitConnections(1024, "../../config/server_authoratative_fsm.json", "../../config/client_authoratative_fsm.json")
	channeld.InitChannels()
	channeld.GetChannel(channeld.GlobalChannelId).InitData(
		&proto.ChatChannelData{ChatMessages: []*proto.ChatMessage{ //make([]*proto.ChatMessage, 0)},
			{Sender: "System", SendTime: time.Now().Unix(), Content: "Welcome!"},
		}},
		&channeld.DataMergeOptions{ListSizeLimit: 100},
	)
	//channeld.SetWebSocketTrustedOrigins(["localhost"])
	go channeld.StartListening(channeld.CLIENT, "ws", *wsAddr)

	log.Fatal(http.ListenAndServe(*webAddr, nil))
}
