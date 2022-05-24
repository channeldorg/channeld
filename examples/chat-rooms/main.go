package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"channeld.clewcat.com/channeld/pkg/channeld"
	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"channeld.clewcat.com/examples/chat-rooms/chatpb"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// The addresses should only include the port for running in the docker containers.
var webAddr = flag.String("web", ":8080", "http service address")
var wsAddr = flag.String("ws", ":12108", "websocket service address")

func handleChanneldProto(w http.ResponseWriter, r *http.Request) {
	bytes, err := os.ReadFile("../../pkg/channeldpb/channeld.proto")
	if err != nil {
		log.Panic(err)
	}
	w.Write(bytes)
}

func handleChatProto(w http.ResponseWriter, r *http.Request) {
	bytes, err := os.ReadFile("./chatpb/chat.proto")
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

	localAddr := r.Host[:strings.Index(r.Host, ":")]
	//log.Println(localAddr)
	templateData.ServerAddress = localAddr + *wsAddr
	homeTemplate.Execute(w, templateData)
}

type TemplateData struct {
	ServerAddress   string
	CompressionType uint
}

var templateData TemplateData

func main() {
	if err := channeld.GlobalSettings.ParseFlag(); err != nil {
		fmt.Printf("error parsing CLI flag: %v\n", err)
	}
	channeld.StartProfiling()
	templateData.CompressionType = uint(channeld.GlobalSettings.CompressionType)

	http.HandleFunc("/", handleMain)
	http.HandleFunc("/proto", handleChanneldProto)
	http.HandleFunc("/proto/chat", handleChatProto)
	http.Handle("/metrics", promhttp.Handler())

	channeld.InitLogsAndMetrics()
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
	//channeld.SetWebSocketTrustedOrigins(["localhost"])
	go channeld.StartListening(channeldpb.ConnectionType_CLIENT, "ws", *wsAddr)

	log.Fatal(http.ListenAndServe(*webAddr, nil))

}
