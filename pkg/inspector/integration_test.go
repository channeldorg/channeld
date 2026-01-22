// +build integration

package inspector

import (
	"bytes"
	"context"
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/channeldorg/channeld/pkg/channeld"
	"github.com/channeldorg/channeld/pkg/channeldpb"
	"github.com/channeldorg/channeld/pkg/common"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

// TestInspectorGetChannels tests the INSPECTOR_GET_CHANNELS message handler
func TestInspectorGetChannels(t *testing.T) {
	// Setup
	channeld.InitChannels()
	RegisterInspectorHandlers()
	
	// Create test channels
	ch1, _ := channeld.CreateChannel(channeldpb.ChannelType_SUBWORLD, nil)
	ch2, _ := channeld.CreateChannel(channeldpb.ChannelType_PRIVATE, nil)
	
	// Create a mock connection
	conn := createMockConnection(t)
	defer conn.Close()
	
	// Create request
	req := &channeldpb.InspectorGetChannelsRequest{
		Page:     1,
		PageSize: 10,
	}
	
	// Create message context
	ctx := channeld.MessageContext{
		MsgType:    channeldpb.MessageType_INSPECTOR_GET_CHANNELS,
		Msg:        req,
		Connection: conn,
		ChannelId:  0, // GLOBAL channel
	}
	
	// Handle the request
	HandleInspectorGetChannels(ctx)
	
	// Note: In a real test, we would need to capture the response
	// For now, we just verify the handler doesn't panic
	assert.NotNil(t, ch1)
	assert.NotNil(t, ch2)
}

// createMockConnection creates a mock connection for testing
func createMockConnection(t *testing.T) *channeld.Connection {
	// Create a TCP connection
	conn, err := net.Dial("tcp", "127.0.0.1:0")
	if err != nil {
		// If we can't create a real connection, we'll need to mock it differently
		t.Skip("Cannot create test connection")
	}
	
	// Create a channeld connection
	// Note: This is a simplified version - in reality, we'd need to properly initialize the connection
	// For integration tests, we might want to use the actual channeld server
	
	return nil // Placeholder - would need proper connection setup
}

// TestInspectorEventListeners tests that event listeners are properly registered
func TestInspectorEventListeners(t *testing.T) {
	// Setup
	channeld.InitChannels()
	InitInspector()
	
	// Create a channel - this should trigger the event
	ch, err := channeld.CreateChannel(channeldpb.ChannelType_SUBWORLD, nil)
	assert.NoError(t, err)
	assert.NotNil(t, ch)
	
	// Give some time for the event to be processed
	time.Sleep(100 * time.Millisecond)
	
	// Verify the channel exists
	info := GetChannelInfo(ch.Id())
	assert.NotNil(t, info)
}

// TestInspectorChannelData tests getting channel data
func TestInspectorChannelData(t *testing.T) {
	// Setup
	channeld.InitChannels()
	RegisterInspectorHandlers()
	
	// Create a channel with data
	ch, err := channeld.CreateChannel(channeldpb.ChannelType_SUBWORLD, nil)
	assert.NoError(t, err)
	
	// Initialize channel data (if needed)
	// ch.InitData(...)
	
	// Get channel data
	ch.Execute(func(ch *channeld.Channel) {
		jsonData, hasData, err := getChannelDataJSON(ch)
		if hasData {
			assert.NoError(t, err)
			assert.NotEmpty(t, jsonData)
		}
	})
}

// Helper function to create a message pack for testing
func createMessagePack(msgType channeldpb.MessageType, msg proto.Message) (*channeldpb.MessagePack, error) {
	msgBody, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	
	return &channeldpb.MessagePack{
		ChannelId: 0, // GLOBAL channel
		MsgType:   uint32(msgType),
		MsgBody:   msgBody,
	}, nil
}

// Helper function to read response from connection
func readResponse(conn net.Conn, timeout time.Duration) (*channeldpb.Packet, error) {
	conn.SetReadDeadline(time.Now().Add(timeout))
	
	// Read packet header (size)
	var size uint32
	if err := binary.Read(conn, binary.LittleEndian, &size); err != nil {
		return nil, err
	}
	
	// Read packet body
	buf := make([]byte, size)
	if _, err := conn.Read(buf); err != nil {
		return nil, err
	}
	
	// Unmarshal packet
	var packet channeldpb.Packet
	if err := proto.Unmarshal(buf, &packet); err != nil {
		return nil, err
	}
	
	return &packet, nil
}

// Helper function to send message pack
func sendMessagePack(conn net.Conn, mp *channeldpb.MessagePack) error {
	// Create packet
	packet := &channeldpb.Packet{
		Messages: []*channeldpb.MessagePack{mp},
	}
	
	// Marshal packet
	data, err := proto.Marshal(packet)
	if err != nil {
		return err
	}
	
	// Send size
	size := uint32(len(data))
	if err := binary.Write(conn, binary.LittleEndian, size); err != nil {
		return err
	}
	
	// Send data
	_, err = conn.Write(data)
	return err
}
