package inspector

import (
	"errors"

	"github.com/channeldorg/channeld/pkg/channeld"
	"github.com/channeldorg/channeld/pkg/common"
)

// GetAllChannels returns a snapshot of all channels
// This function is thread-safe as it creates a snapshot
func GetAllChannels() []*ChannelInfo {
	var channels []*ChannelInfo

	// Get snapshot from channeld
	channelSnapshots := channeld.GetAllChannelsSnapshot()

	for _, ch := range channelSnapshots {
		info := getChannelInfo(ch)
		if info != nil {
			channels = append(channels, info)
		}
	}

	return channels
}

// GetChannelInfo returns channel information for a specific channel
func GetChannelInfo(channelID common.ChannelId) *ChannelInfo {
	ch := channeld.GetChannel(channelID)
	if ch == nil {
		return nil
	}
	return getChannelInfo(ch)
}

// getChannelInfo extracts channel information from a Channel object
// Must be called in a thread-safe context
func getChannelInfo(ch *channeld.Channel) *ChannelInfo {
	if ch == nil {
		return nil
	}

	// Get owner connection ID
	owner := ch.GetOwner()
	var ownerConnID channeld.ConnectionId = 0
	if owner != nil {
		ownerConnID = owner.Id()
	}

	// Get subscriber count (thread-safe)
	allConns := ch.GetAllConnections()
	subscriberCount := len(allConns)

	return &ChannelInfo{
		ChannelID:       ch.Id(),
		ChannelType:     ch.Type(),
		OwnerConnID:     ownerConnID,
		SubscriberCount: subscriberCount,
		Metadata:        ch.Metadata(),
		CreatedTime:     ch.StartTime().UnixMilli(),
	}
}

// GetChannelDataJSON is moved to handlers.go to avoid import cycle

// UpdateChannelDataByPath updates a channel data field by JSON path
// Must be called in the channel's goroutine for thread safety
// UpdateChannelDataByPath is implemented in handlers.go

// Helper functions to access channeld internals (if needed)
// These will need to be added to channeld package or use existing public APIs

var (
	ErrChannelNotFound  = errors.New("channel not found")
	ErrChannelHasNoData = errors.New("channel has no data")
)
