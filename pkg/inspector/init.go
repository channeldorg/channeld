package inspector

import (
	"github.com/channeldorg/channeld/pkg/channeld"
	"github.com/channeldorg/channeld/pkg/channeldpb"
)

// RegisterInspectorHandlers registers Inspector message handlers
// This should be called after protobuf code generation
func RegisterInspectorHandlers() {
	// Register request handlers
	channeld.RegisterMessageHandler(
		uint32(channeldpb.MessageType_DEBUG_GET_CHANNELS),
		&channeldpb.DebugGetChannelsRequest{},
		HandleDebugGetChannels,
	)

	channeld.RegisterMessageHandler(
		uint32(channeldpb.MessageType_DEBUG_GET_CHANNEL),
		&channeldpb.DebugGetChannelRequest{},
		HandleDebugGetChannel,
	)

	channeld.RegisterMessageHandler(
		uint32(channeldpb.MessageType_DEBUG_GET_CHANNEL_DATA),
		&channeldpb.DebugGetChannelDataRequest{},
		HandleDebugGetChannelData,
	)

	channeld.RegisterMessageHandler(
		uint32(channeldpb.MessageType_DEBUG_UPDATE_CHANNEL_DATA),
		&channeldpb.DebugUpdateChannelDataRequest{},
		HandleDebugUpdateChannelData,
	)

	channeld.RegisterMessageHandler(
		uint32(channeldpb.MessageType_DEBUG_SUBSCRIBE_CHANNEL_LIST),
		&channeldpb.DebugSubscribeChannelListRequest{},
		HandleDebugSubscribeChannelList,
	)

	channeld.RegisterMessageHandler(
		uint32(channeldpb.MessageType_DEBUG_UNSUBSCRIBE_CHANNEL_LIST),
		&channeldpb.DebugUnsubscribeChannelListRequest{},
		HandleDebugUnsubscribeChannelList,
	)

	// Initialize event listeners
	InitInspector()
}
