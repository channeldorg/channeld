package inspector

import (
	"github.com/channeldorg/channeld/pkg/channeld"
	"github.com/channeldorg/channeld/pkg/channeldpb"
	"github.com/channeldorg/channeld/pkg/common"
)

// InspectorConnection represents an Inspector WebSocket connection
type InspectorConnection struct {
	conn             *channeld.Connection
	subscribedToList bool // Whether subscribed to channel list updates
	fullSyncEnabled  bool // Whether full sync is enabled
	fullSyncInterval int  // Full sync interval in seconds
}

// GetInspectorConnection returns the InspectorConnection for a given connection, or nil if not an Inspector connection
func GetInspectorConnection(conn *channeld.Connection) *InspectorConnection {
	// TODO: Add connection type checking or metadata to identify Inspector connections
	// For now, we'll handle all connections that send Inspector messages
	return &InspectorConnection{
		conn:             conn,
		subscribedToList: false,
		fullSyncEnabled:  false,
		fullSyncInterval: 300, // Default 5 minutes
	}
}

// ChannelInfo represents channel information for Inspector
type ChannelInfo struct {
	ChannelID       common.ChannelId
	ChannelType     channeldpb.ChannelType
	OwnerConnID     channeld.ConnectionId
	SubscriberCount int
	Metadata        string
	CreatedTime     int64 // Unix timestamp in milliseconds
}
