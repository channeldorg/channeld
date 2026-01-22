package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/channeldorg/channeld/pkg/channeld"
	"github.com/channeldorg/channeld/pkg/channeldpb"
	"github.com/channeldorg/channeld/pkg/inspector"
	"google.golang.org/protobuf/proto"
)

func main() {
	// Initialize channeld
	if err := channeld.GlobalSettings.ParseFlag(); err != nil {
		log.Printf("error parsing CLI flag: %v\n", err)
	}
	channeld.StartProfiling()
	channeld.InitLogs()
	channeld.InitMetrics()
	channeld.InitConnections(channeld.GlobalSettings.ServerFSM, channeld.GlobalSettings.ClientFSM)
	channeld.InitChannels()
	
	// Initialize Inspector
	inspector.RegisterInspectorHandlers()
	
	// Create some test channels
	fmt.Println("Creating test channels...")
	ch1, _ := channeld.CreateChannel(channeldpb.ChannelType_SUBWORLD, nil)
	ch2, _ := channeld.CreateChannel(channeldpb.ChannelType_PRIVATE, nil)
	fmt.Printf("Created channels: %d, %d\n", ch1.Id(), ch2.Id())
	
	// Start server
	serverAddr := ":12108"
	if channeld.GlobalSettings.ClientAddress != "" {
		serverAddr = channeld.GlobalSettings.ClientAddress
	}
	
	fmt.Printf("Starting Inspector test server on %s\n", serverAddr)
	fmt.Println("You can connect using a WebSocket client or the test client")
	fmt.Println("Press Enter to start interactive test...")
	
	// Wait for user input
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')
	
	// Run interactive test
	runInteractiveTest()
}

func runInteractiveTest() {
	fmt.Println("\n=== Inspector Backend Test ===")
	
	// Test 1: Get all channels
	fmt.Println("\n1. Testing GetAllChannels()...")
	channels := inspector.GetAllChannels()
	fmt.Printf("   Found %d channels:\n", len(channels))
	for _, ch := range channels {
		fmt.Printf("   - Channel %d: Type=%s, Owner=%d, Subscribers=%d\n",
			ch.ChannelID, ch.ChannelType.String(), ch.OwnerConnID, ch.SubscriberCount)
	}
	
	// Test 2: Get specific channel info
	if len(channels) > 0 {
		fmt.Println("\n2. Testing GetChannelInfo()...")
		info := inspector.GetChannelInfo(channels[0].ChannelID)
		if info != nil {
			fmt.Printf("   Channel %d info retrieved successfully\n", info.ChannelID)
		}
	}
	
	// Test 3: Test channel data
	fmt.Println("\n3. Testing channel data access...")
	allChannels := channeld.GetAllChannelsSnapshot()
	if len(allChannels) > 0 {
		ch := allChannels[0]
		ch.Execute(func(ch *channeld.Channel) {
			jsonData, hasData, err := getChannelDataJSON(ch)
			if hasData {
				if err != nil {
					fmt.Printf("   Error getting channel data: %v\n", err)
				} else {
					fmt.Printf("   Channel %d has data (length: %d bytes)\n", ch.Id(), len(jsonData))
				}
			} else {
				fmt.Printf("   Channel %d has no data\n", ch.Id())
			}
		})
	}
	
	fmt.Println("\n=== Test Complete ===")
	fmt.Println("Server is still running. You can test with WebSocket clients.")
}

// Helper function to get channel data JSON (copied from handlers.go)
func getChannelDataJSON(ch *channeld.Channel) (string, bool, error) {
	if ch == nil {
		return "", false, nil
	}
	
	data := ch.Data()
	if data == nil {
		return "", false, nil
	}
	
	msg := ch.GetDataMessage()
	if msg == nil {
		return "", false, nil
	}
	
	// Convert protobuf message to JSON
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		return "", true, err
	}
	
	return string(jsonBytes), true, nil
}

// Helper function to send a message and receive response
func sendInspectorMessage(conn net.Conn, msgType channeldpb.MessageType, msg proto.Message) (*channeldpb.Packet, error) {
	// Create message pack
	msgBody, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	
	mp := &channeldpb.MessagePack{
		ChannelId: 0, // GLOBAL channel
		MsgType:   uint32(msgType),
		MsgBody:   msgBody,
	}
	
	// Create packet
	packet := &channeldpb.Packet{
		Messages: []*channeldpb.MessagePack{mp},
	}
	
	// Marshal and send
	data, err := proto.Marshal(packet)
	if err != nil {
		return nil, err
	}
	
	// Send size
	size := uint32(len(data))
	if err := binary.Write(conn, binary.LittleEndian, size); err != nil {
		return nil, err
	}
	
	// Send data
	if _, err := conn.Write(data); err != nil {
		return nil, err
	}
	
	// Read response
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	
	// Read size
	var respSize uint32
	if err := binary.Read(conn, binary.LittleEndian, &respSize); err != nil {
		return nil, err
	}
	
	// Read data
	respData := make([]byte, respSize)
	if _, err := conn.Read(respData); err != nil {
		return nil, err
	}
	
	// Unmarshal response
	var respPacket channeldpb.Packet
	if err := proto.Unmarshal(respData, &respPacket); err != nil {
		return nil, err
	}
	
	return &respPacket, nil
}
