//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"time"

	"github.com/channeldorg/channeld/pkg/channeld"
	"github.com/channeldorg/channeld/pkg/channeldpb"
	"github.com/channeldorg/channeld/pkg/inspector"
)

// This is a simple test program to verify Inspector backend functionality
// Run with: go run test_main.go
func main() {
	fmt.Println("=== Inspector Backend Test ===")

	// Initialize channeld (minimal setup for testing)
	channeld.InitLogs()

	// Try to initialize connections with default configs
	// If config files don't exist, we'll skip connection initialization
	serverFSM := "../../config/server_authoratative_fsm.json"
	clientFSM := "../../config/client_non_authoratative_fsm.json"

	fmt.Println("Initializing connections...")
	err := func() error {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Warning: Connection initialization failed: %v\n", r)
				fmt.Println("Continuing without connection initialization...")
			}
		}()
		channeld.InitConnections(serverFSM, clientFSM)
		return nil
	}()
	if err != nil {
		fmt.Printf("Note: %v\n", err)
	}

	fmt.Println("Initializing channels...")
	channeld.InitChannels()

	// Initialize Inspector
	fmt.Println("Registering Inspector handlers...")
	inspector.RegisterInspectorHandlers()

	// Test 1: Create test channels
	fmt.Println("\n1. Creating test channels...")
	ch1, err1 := channeld.CreateChannel(channeldpb.ChannelType_SUBWORLD, nil)
	if err1 != nil {
		fmt.Printf("   Error creating channel: %v\n", err1)
	} else {
		fmt.Printf("   Created SUBWORLD channel: %d\n", ch1.Id())
	}

	ch2, err2 := channeld.CreateChannel(channeldpb.ChannelType_PRIVATE, nil)
	if err2 != nil {
		fmt.Printf("   Error creating channel: %v\n", err2)
	} else {
		fmt.Printf("   Created PRIVATE channel: %d\n", ch2.Id())
	}

	// Test 2: Get all channels
	fmt.Println("\n2. Testing GetAllChannels()...")
	channels := inspector.GetAllChannels()
	fmt.Printf("   Found %d channels:\n", len(channels))
	for i, ch := range channels {
		if i >= 5 {
			fmt.Printf("   ... and %d more\n", len(channels)-5)
			break
		}
		fmt.Printf("   - Channel %d: Type=%s, Owner=%d, Subscribers=%d, Metadata=%s\n",
			ch.ChannelID, ch.ChannelType.String(), ch.OwnerConnID, ch.SubscriberCount, ch.Metadata)
	}

	// Test 3: Get specific channel info
	if len(channels) > 0 {
		fmt.Println("\n3. Testing GetChannelInfo()...")
		info := inspector.GetChannelInfo(channels[0].ChannelID)
		if info != nil {
			fmt.Printf("   Channel %d info:\n", info.ChannelID)
			fmt.Printf("   - Type: %s\n", info.ChannelType.String())
			fmt.Printf("   - Owner: %d\n", info.OwnerConnID)
			fmt.Printf("   - Subscribers: %d\n", info.SubscriberCount)
			fmt.Printf("   - Metadata: %s\n", info.Metadata)
			fmt.Printf("   - Created: %s\n", time.UnixMilli(info.CreatedTime).Format(time.RFC3339))
		}
	}

	// Test 4: Test event listeners
	fmt.Println("\n4. Testing event listeners...")
	fmt.Println("   Creating a new channel to trigger event...")
	ch3, err3 := channeld.CreateChannel(channeldpb.ChannelType_SUBWORLD, nil)
	if err3 != nil {
		fmt.Printf("   Error: %v\n", err3)
	} else {
		fmt.Printf("   Created channel %d (event should have been triggered)\n", ch3.Id())
		time.Sleep(100 * time.Millisecond) // Give time for event processing
	}

	// Test 5: Verify Inspector handlers are registered
	fmt.Println("\n5. Verifying Inspector handlers...")
	fmt.Println("   Inspector handlers should be registered in MessageMap")
	fmt.Println("   (This would be verified by actually sending messages)")

	fmt.Println("\n=== Test Complete ===")
	fmt.Println("\nTo test with actual connections:")
	fmt.Println("1. Start the server: go run cmd/main.go")
	fmt.Println("2. Connect a WebSocket client to the server")
	fmt.Println("3. Send Inspector messages (INSPECTOR_GET_CHANNELS, etc.)")
}
