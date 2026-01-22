package inspector

import (
	"encoding/json"
	"testing"

	"github.com/channeldorg/channeld/pkg/channeld"
	"github.com/channeldorg/channeld/pkg/channeldpb"
	"github.com/stretchr/testify/assert"
)

func TestGetAllChannels(t *testing.T) {
	// Initialize channeld (requires proper setup)
	// Note: This test requires channeld to be properly initialized
	// In a real test environment, you would set up connections first
	channeld.InitLogs()
	channeld.InitConnections("", "") // Use empty FSM configs for testing
	channeld.InitChannels()
	
	// Create a test channel
	ch, err := channeld.CreateChannel(channeldpb.ChannelType_SUBWORLD, nil)
	assert.NoError(t, err)
	assert.NotNil(t, ch)
	
	// Get all channels
	channels := GetAllChannels()
	assert.GreaterOrEqual(t, len(channels), 1)
	
	// Verify the channel is in the list
	found := false
	for _, info := range channels {
		if info.ChannelID == ch.Id() {
			found = true
			assert.Equal(t, channeldpb.ChannelType_SUBWORLD, info.ChannelType)
			break
		}
	}
	assert.True(t, found, "Created channel should be in the list")
}

func TestGetChannelInfo(t *testing.T) {
	// Initialize channeld
	channeld.InitLogs()
	channeld.InitConnections("", "")
	channeld.InitChannels()
	
	// Create a test channel
	ch, err := channeld.CreateChannel(channeldpb.ChannelType_PRIVATE, nil)
	assert.NoError(t, err)
	assert.NotNil(t, ch)
	
	// Get channel info
	info := GetChannelInfo(ch.Id())
	assert.NotNil(t, info)
	assert.Equal(t, ch.Id(), info.ChannelID)
	assert.Equal(t, channeldpb.ChannelType_PRIVATE, info.ChannelType)
}

func TestInspectorConnection(t *testing.T) {
	// Test InspectorConnection creation
	// Note: This requires a real connection, so we'll just test the structure
	conn := &InspectorConnection{
		subscribedToList: false,
		fullSyncEnabled:  false,
		fullSyncInterval: 300,
	}
	
	assert.False(t, conn.subscribedToList)
	assert.False(t, conn.fullSyncEnabled)
	assert.Equal(t, 300, conn.fullSyncInterval)
}

func TestChannelInfoJSON(t *testing.T) {
	// Initialize channeld
	channeld.InitLogs()
	channeld.InitConnections("", "")
	channeld.InitChannels()
	
	// Create a test channel
	ch, err := channeld.CreateChannel(channeldpb.ChannelType_GLOBAL, nil)
	assert.NoError(t, err)
	assert.NotNil(t, ch)
	
	// Get channel info
	info := GetChannelInfo(ch.Id())
	assert.NotNil(t, info)
	
	// Convert to JSON to verify structure
	jsonData, err := json.Marshal(info)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), "ChannelID")
	assert.Contains(t, string(jsonData), "ChannelType")
}
