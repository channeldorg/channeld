package channeld

import (
	"strings"

	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"channeld.clewcat.com/channeld/pkg/common"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type Message = proto.Message //protoreflect.ProtoMessage

// The context of a message for both sending and receiving
type MessageContext struct {
	MsgType channeldpb.MessageType
	// The weak-typed Message object popped from the message queue
	Msg       Message
	Broadcast uint32 //channeldpb.BroadcastType
	StubId    uint32
	// The original channelId in the Packet, could be different from Channel.id.
	ChannelId uint32

	// The connection that received the message. Required for BroadcastType_ALL_BUT_SENDER but not for sending.
	Connection ConnectionInChannel
	// The channel that handling the message. Not required for sending or broadcasting.
	Channel *Channel
}

func (ctx *MessageContext) HasConnection() bool {
	conn, ok := ctx.Connection.(*Connection)
	return ok && conn != nil && !conn.IsClosing()
}

type MessageHandlerFunc func(ctx MessageContext)
type messageMapEntry struct {
	msg     Message
	handler MessageHandlerFunc
}

var MessageMap = map[channeldpb.MessageType]*messageMapEntry{
	channeldpb.MessageType_AUTH:                {&channeldpb.AuthMessage{}, handleAuth},
	channeldpb.MessageType_CREATE_CHANNEL:      {&channeldpb.CreateChannelMessage{}, handleCreateChannel},
	channeldpb.MessageType_REMOVE_CHANNEL:      {&channeldpb.RemoveChannelMessage{}, handleRemoveChannel},
	channeldpb.MessageType_LIST_CHANNEL:        {&channeldpb.ListChannelMessage{}, handleListChannel},
	channeldpb.MessageType_SUB_TO_CHANNEL:      {&channeldpb.SubscribedToChannelMessage{}, handleSubToChannel},
	channeldpb.MessageType_UNSUB_FROM_CHANNEL:  {&channeldpb.UnsubscribedFromChannelMessage{}, handleUnsubFromChannel},
	channeldpb.MessageType_CHANNEL_DATA_UPDATE: {&channeldpb.ChannelDataUpdateMessage{}, handleChannelDataUpdate},
	channeldpb.MessageType_DISCONNECT:          {&channeldpb.DisconnectMessage{}, handleDisconnect},
	// CREATE_CHANNEL and CREATE_SPATIAL_CHANNEL shared the same message structure and handler
	channeldpb.MessageType_CREATE_SPATIAL_CHANNEL:    {&channeldpb.CreateChannelMessage{}, handleCreateChannel},
	channeldpb.MessageType_QUERY_SPATIAL_CHANNEL:     {&channeldpb.QuerySpatialChannelMessage{}, handleQuerySpatialChannel},
	channeldpb.MessageType_DEBUG_GET_SPATIAL_REGIONS: {&channeldpb.DebugGetSpatialRegionsMessage{}, handleGetSpatialRegionsMessage},
}

func RegisterMessageHandler(msgType uint32, msg Message, handler MessageHandlerFunc) {
	MessageMap[channeldpb.MessageType(msgType)] = &messageMapEntry{msg, handler}
}

func handleClientToServerUserMessage(ctx MessageContext) {
	msg, ok := ctx.Msg.(*channeldpb.ServerForwardMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a ServerForwardMessage, will not be handled.")
		return
	}
	ctx.Connection.Logger().Debug("forward user-space message from client to server",
		zap.Uint32("msgType", uint32(ctx.MsgType)),
		zap.Uint32("clientConnId", msg.ClientConnId),
		zap.Uint32("channelId", uint32(ctx.Channel.id)),
		zap.Uint32("broadcastType", ctx.Broadcast),
		zap.Int("payloadSize", len(msg.Payload)),
	)

	if ctx.Channel.HasOwner() {
		ctx.Channel.ownerConnection.Send(ctx)
	} else if ctx.Broadcast > 0 {
		if ctx.Channel.enableClientBroadcast {
			ctx.Channel.Broadcast(ctx)
		} else {
			ctx.Connection.Logger().Error("illegal attempt to broadcast message as the channel's client broadcasting is disabled",
				zap.Uint32("msgType", uint32(ctx.MsgType)),
				zap.String("channelType", ctx.Channel.channelType.String()),
				zap.Uint32("channelId", uint32(ctx.Channel.id)),
			)
		}
	} else {
		ctx.Channel.Logger().Error("channel has no owner to forward the user-space messaged",
			zap.Uint32("msgType", uint32(ctx.MsgType)),
			zap.Uint32("connId", uint32(ctx.Connection.Id())),
		)
	}
}

func handleServerToClientUserMessage(ctx MessageContext) {
	msg, ok := ctx.Msg.(*channeldpb.ServerForwardMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a ServerForwardMessage, will not be handled.")
		return
	}

	ctx.Connection.Logger().Debug("forward user-space message from server to client",
		zap.Uint32("msgType", uint32(ctx.MsgType)),
		zap.Uint32("clientConnId", msg.ClientConnId),
		zap.Uint32("channelId", uint32(ctx.Channel.id)),
		zap.Uint32("broadcastType", ctx.Broadcast),
		zap.Int("payloadSize", len(msg.Payload)),
	)

	switch channeldpb.BroadcastType(ctx.Broadcast) {
	case channeldpb.BroadcastType_NO_BROADCAST:
		if ctx.Channel.HasOwner() {
			ctx.Channel.ownerConnection.Send(ctx)
		} else {
			ctx.Connection.Logger().Error("cannot forward the message as the channel has no owner",
				zap.Uint32("msgType", uint32(ctx.MsgType)),
				zap.String("channelType", ctx.Channel.channelType.String()),
				zap.Uint32("channelId", uint32(ctx.Channel.id)),
			)
		}

	case channeldpb.BroadcastType_ALL, channeldpb.BroadcastType_ALL_BUT_SENDER, channeldpb.BroadcastType_ALL_BUT_OWNER:
		ctx.Channel.Broadcast(ctx)

	case channeldpb.BroadcastType_SINGLE_CONNECTION:
		clientConn := GetConnection(ConnectionId(msg.ClientConnId))
		if clientConn != nil {
			clientConn.Send(ctx)
		} else {
			ctx.Connection.Logger().Warn("cannot forward the message as the target connection does not exist",
				zap.Uint32("msgType", uint32(ctx.MsgType)),
				zap.Uint32("targetConnId", msg.ClientConnId),
			)
		}

	default:
		if channeldpb.BroadcastType_ADJACENT_CHANNELS.Check(ctx.Broadcast) {
			if ctx.Channel.channelType != channeldpb.ChannelType_SPATIAL {
				ctx.Connection.Logger().Warn("BroadcastType_ADJACENT_CHANNELS only works for Spatial channel")
				return
			}
			if spatialController == nil {
				ctx.Connection.Logger().Error("spatial controller doesn't exist")
				return
			}
			channelIds, err := spatialController.GetAdjacentChannels(ctx.Channel.id)
			if err != nil {
				ctx.Connection.Logger().Error("failed to retrieve spatial regions", zap.Error(err))
				return
			}
			// Add the connections in the owner(center) channel?
			if !channeldpb.BroadcastType_ALL_BUT_OWNER.Check(ctx.Broadcast) {
				channelIds = append(channelIds, ctx.Channel.id)
			}

			// Merge all connection in the adjacent channels to one map, to avoid duplicate send.
			adjacentConns := make(map[ConnectionInChannel]struct{})
			for _, id := range channelIds {
				channel := GetChannel(id)
				if channel == nil {
					ctx.Connection.Logger().Error("invalid channel id for broadcast", zap.Uint32("channelId", uint32(id)))
					continue
				}
				conns := channel.GetAllConnections()
				for conn := range conns {
					adjacentConns[conn] = struct{}{}
				}
			}
			for conn := range adjacentConns {
				// Ignore the sender?
				if channeldpb.BroadcastType_ALL_BUT_SENDER.Check(ctx.Broadcast) && conn == ctx.Connection {
					continue
				}
				// Ignore the client
				if conn.Id() == ConnectionId(msg.ClientConnId) {
					continue
				}
				conn.Send(ctx)
			}
		}
	}
}

func handleAuth(ctx MessageContext) {
	if ctx.Channel != globalChannel {
		ctx.Connection.Logger().Error("illegal attemp to authenticate outside the GLOBAL channel")
		ctx.Connection.Close()
		return
	}
	msg, ok := ctx.Msg.(*channeldpb.AuthMessage)
	if !ok {
		ctx.Connection.Logger().Error("mssage is not an AuthMessage, will not be handled.")
		ctx.Connection.Close()
		return
	}
	//log.Printf("Auth PIT: %s, LT: %s\n", msg.PlayerIdentifierToken, msg.LoginToken)

	if authProvider == nil && !GlobalSettings.Development {
		rootLogger.Panic("no auth provider")
		return
	}

	authResult := channeldpb.AuthResultMessage_SUCCESSFUL
	if ctx.Connection.GetConnectionType() == channeldpb.ConnectionType_SERVER && GlobalSettings.ServerBypassAuth {
		onAuthComplete(ctx, authResult)
	} else if authProvider != nil {
		go func() {
			authResult, err := authProvider.DoAuth(ctx.Connection.Id(), msg.PlayerIdentifierToken, msg.LoginToken)
			if err != nil {
				ctx.Connection.Logger().Error("failed to do auth", zap.Error(err))
				ctx.Connection.Close()
			} else {
				onAuthComplete(ctx, authResult)
			}
		}()
	} else {
		onAuthComplete(ctx, authResult)
	}
}

func onAuthComplete(ctx MessageContext, authResult channeldpb.AuthResultMessage_AuthResult) {
	if authResult == channeldpb.AuthResultMessage_SUCCESSFUL {
		ctx.Connection.OnAuthenticated()
	}

	ctx.Msg = &channeldpb.AuthResultMessage{
		Result:          authResult,
		ConnId:          uint32(ctx.Connection.Id()),
		CompressionType: GlobalSettings.CompressionType,
	}
	ctx.Connection.Send(ctx)

	// Also send the respond to The GLOBAL channel owner (to handle the client's subscription if it doesn't have the authority to).
	if globalChannel.HasOwner() {
		ctx.StubId = 0
		globalChannel.ownerConnection.Send(ctx)
	}
}

func handleCreateChannel(ctx MessageContext) {
	// Only the GLOBAL channel can handle channel creation/deletion/listing
	if ctx.Channel != globalChannel {
		ctx.Connection.Logger().Error("illegal attemp to create channel outside the GLOBAL channel")
		return
	}

	msg, ok := ctx.Msg.(*channeldpb.CreateChannelMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a CreateChannelMessage, will not be handled.")
		return
	}

	var newChannel *Channel
	var err error
	if msg.ChannelType == channeldpb.ChannelType_UNKNOWN {
		ctx.Connection.Logger().Error("illegal attemp to create the UNKNOWN channel")
		return
	} else if msg.ChannelType == channeldpb.ChannelType_GLOBAL {
		// Global channel is initially created by the system. Creating the channel will attempt to own it.
		newChannel = globalChannel
		if !globalChannel.HasOwner() {
			globalChannel.ownerConnection = ctx.Connection
			ctx.Connection.Logger().Info("owned the GLOBAL channel")
		} else {
			ctx.Connection.Logger().Error("illegal attemp to create the GLOBAL channel")
			return
		}
	} else if msg.ChannelType == channeldpb.ChannelType_SPATIAL {
		handleCreateSpatialChannel(ctx, msg)
		return
	} else {
		newChannel, err = CreateChannel(msg.ChannelType, ctx.Connection)
		if err != nil {
			ctx.Connection.Logger().Error("failed to create channel",
				zap.Uint32("channelType", uint32(msg.ChannelType)),
				zap.Error(err),
			)
			return
		}
		newChannel.Logger().Info("created channel with owner", zap.Uint32("ownerConnId", uint32(newChannel.ownerConnection.Id())))
	}

	newChannel.metadata = msg.Metadata
	if msg.Data != nil {
		dataMsg, err := msg.Data.UnmarshalNew()
		if err != nil {
			newChannel.Logger().Error("failed to unmarshal data message for the new channel", zap.Error(err))
			return
		} else {
			newChannel.InitData(dataMsg, msg.MergeOptions)
		}
	} else {
		// Channel data should always be initialized
		newChannel.InitData(nil, msg.MergeOptions)
	}

	ctx.Msg = &channeldpb.CreateChannelResultMessage{
		ChannelType: newChannel.channelType,
		Metadata:    newChannel.metadata,
		OwnerConnId: uint32(ctx.Connection.Id()),
		ChannelId:   uint32(newChannel.id),
	}
	ctx.Connection.Send(ctx)
	// Also send the response to the GLOBAL channel owner.
	if globalChannel.ownerConnection != ctx.Connection && globalChannel.HasOwner() {
		ctx.StubId = 0
		globalChannel.ownerConnection.Send(ctx)
	}

	// Subscribe to channel after creation
	cs := ctx.Connection.SubscribeToChannel(newChannel, msg.SubOptions)
	if cs != nil {
		ctx.Connection.sendSubscribed(ctx, newChannel, ctx.Connection, 0, &cs.options)
	}
}

func handleCreateSpatialChannel(ctx MessageContext, msg *channeldpb.CreateChannelMessage) {
	if ctx.Connection.GetConnectionType() != channeldpb.ConnectionType_SERVER {
		ctx.Connection.Logger().Error("illegal attemp to create Spatial channel from client connection")
		return
	}

	if spatialController == nil {
		ctx.Connection.Logger().Error("illegal attemp to create Spatial channel as there's no controller")
		return
	}

	channels, err := spatialController.CreateChannels(ctx)
	if err != nil {
		ctx.Connection.Logger().Error("failed to create Spatial channel", zap.Error(err))
		return
	}

	resultMsg := &channeldpb.CreateSpatialChannelsResultMessage{
		SpatialChannelId: make([]uint32, len(channels)),
		Metadata:         msg.Metadata,
		OwnerConnId:      uint32(ctx.Connection.Id()),
	}

	for i := range channels {
		resultMsg.SpatialChannelId[i] = uint32(channels[i].id)
	}

	ctx.MsgType = channeldpb.MessageType_CREATE_SPATIAL_CHANNEL
	ctx.Msg = resultMsg
	ctx.Connection.Send(ctx)
	// Also send the response to the GLOBAL channel owner.
	if globalChannel.ownerConnection != ctx.Connection && globalChannel.HasOwner() {
		ctx.StubId = 0
		globalChannel.ownerConnection.Send(ctx)
	}

	for _, newChannel := range channels {
		// Subscribe to channel after creation
		cs := ctx.Connection.SubscribeToChannel(newChannel, msg.SubOptions)
		if cs != nil {
			ctx.Connection.sendSubscribed(ctx, newChannel, ctx.Connection, 0, &cs.options)
		}
	}

	ctx.Connection.Logger().Info("created spatial channel(s)", zap.Uint32s("channelIds", resultMsg.SpatialChannelId))

	// Send the regions info upon the spatial channels creation
	regions, err := spatialController.GetRegions()
	if err != nil {
		ctx.Connection.Logger().Error("failed to send Spatial regions info upon the spatial channels creation",
			zap.Uint32s("channelIds", resultMsg.SpatialChannelId))
		return
	}
	ctx.MsgType = channeldpb.MessageType_SPATIAL_REGIONS_UPDATE
	ctx.Msg = &channeldpb.SpatialRegionsUpdateMessage{
		Regions: regions,
	}
	ctx.Connection.Send(ctx)
}

func handleRemoveChannel(ctx MessageContext) {

	msg, ok := ctx.Msg.(*channeldpb.RemoveChannelMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a RemoveChannelMessage, will not be handled.")
		return
	}

	channelToRemove := GetChannel(ChannelId(msg.ChannelId))
	if channelToRemove == nil {
		ctx.Connection.Logger().Error("invalid channelId for removing", zap.Uint32("channelId", msg.ChannelId))
		return
	}

	// Check ACL from settings
	// If ctx.Connection == nil, the removal is triggered internally (e.g. ChannelSettings.RemoveChannelAfterOwnerRemoved)
	hasAccess, err := channelToRemove.CheckACL(ctx.Connection, ChannelAccessType_Remove)
	if ctx.HasConnection() && !hasAccess {
		ownerConnId := uint32(0)
		if channelToRemove.HasOwner() {
			ownerConnId = uint32(channelToRemove.ownerConnection.Id())
		}
		ctx.Connection.Logger().Error("connection doesn't have access to remove channel",
			zap.String("channelType", channelToRemove.channelType.String()),
			zap.Uint32("channelId", uint32(channelToRemove.id)),
			zap.Uint32("ownerConnId", ownerConnId),
			zap.Error(err))
		return
	}

	for sc := range channelToRemove.subscribedConnections {
		if sc != nil {
			//sc.sendUnsubscribed(ctx, channelToRemove, 0)
			response := ctx
			response.StubId = 0
			sc.Send(response)
		}
	}
	RemoveChannel(channelToRemove)

	if ctx.HasConnection() {
		ctx.Connection.Logger().Info("removed channel",
			zap.String("channelType", channelToRemove.channelType.String()),
			zap.Uint32("channelId", uint32(channelToRemove.id)),
			zap.Int("subs", len(channelToRemove.subscribedConnections)),
		)
	}
}

func handleListChannel(ctx MessageContext) {
	if ctx.Channel != globalChannel {
		ctx.Connection.Logger().Error("illegal attemp to list channel outside the GLOBAL channel")
		return
	}

	msg, ok := ctx.Msg.(*channeldpb.ListChannelMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a ListChannelMessage, will not be handled.")
		return
	}

	result := make([]*channeldpb.ListChannelResultMessage_ChannelInfo, 0)
	allChannels.Range(func(_, v interface{}) bool {
		channel := v.(*Channel)
		if msg.TypeFilter != channeldpb.ChannelType_UNKNOWN && msg.TypeFilter != channel.channelType {
			return true
		}
		matched := len(msg.MetadataFilters) == 0
		for _, keyword := range msg.MetadataFilters {
			if strings.Contains(channel.metadata, keyword) {
				matched = true
				break
			}
		}
		if matched {
			result = append(result, &channeldpb.ListChannelResultMessage_ChannelInfo{
				ChannelId:   uint32(channel.id),
				ChannelType: channel.channelType,
				Metadata:    channel.metadata,
			})
		}
		return true
	})

	ctx.Msg = &channeldpb.ListChannelResultMessage{
		Channels: result,
	}
	ctx.Connection.Send(ctx)
}

func handleSubToChannel(ctx MessageContext) {
	msg, ok := ctx.Msg.(*channeldpb.SubscribedToChannelMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a SubscribedToChannelMessage, will not be handled.")
		return
	}

	// The connection that subscribes. Could be different to the connection that sends the message.
	connToSub := GetConnection(ConnectionId(msg.ConnId))
	if connToSub == nil {
		ctx.Connection.Logger().Error("invalid ConnectionId for sub", zap.Uint32("connIdInMsg", msg.ConnId))
		return
	}

	hasAccess, err := ctx.Channel.CheckACL(ctx.Connection, ChannelAccessType_Sub)
	if connToSub.id != ctx.Connection.Id() && !hasAccess {
		ctx.Connection.Logger().Error("connection doesn't have access to sub connection to this channel",
			zap.Uint32("subConnId", msg.ConnId),
			zap.String("channelType", ctx.Channel.channelType.String()),
			zap.Uint32("channelId", uint32(ctx.Channel.id)),
			zap.Error(err),
		)
		return
	}

	cs, exists := ctx.Channel.subscribedConnections[connToSub]
	if exists {
		ctx.Connection.Logger().Info("already subscribed to channel, the subscription options will be merged",
			zap.String("channelType", ctx.Channel.channelType.String()),
			zap.Uint32("channelId", uint32(ctx.Channel.id)),
		)
		if msg.SubOptions != nil {
			proto.Merge(&cs.options, msg.SubOptions)
		}
		//// Do not send the SubscribedToChannelResultMessage if already subed.
		connToSub.sendSubscribed(ctx, ctx.Channel, connToSub, 0, &cs.options)
		return
	}

	cs = connToSub.SubscribeToChannel(ctx.Channel, msg.SubOptions)
	if cs == nil {
		return
	}
	// Notify the sender.
	ctx.Connection.sendSubscribed(ctx, ctx.Channel, connToSub, ctx.StubId, &cs.options)

	// Notify the subscribed (if not the sender).
	if connToSub != ctx.Connection {
		connToSub.sendSubscribed(ctx, ctx.Channel, connToSub, 0, &cs.options)
	}
	// Notify the channel owner.
	if ctx.Channel.HasOwner() && ctx.Channel.ownerConnection != ctx.Connection {
		ctx.Channel.ownerConnection.sendSubscribed(ctx, ctx.Channel, connToSub, 0, &cs.options)
	}
}

func handleUnsubFromChannel(ctx MessageContext) {
	msg, ok := ctx.Msg.(*channeldpb.UnsubscribedFromChannelMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a UnsubscribedFromChannelMessage, will not be handled.")
		return
	}

	// The connection that unsubscribes. Could be different to the connection that sends the message.
	connToUnsub := GetConnection(ConnectionId(msg.ConnId))
	if connToUnsub == nil {
		ctx.Connection.Logger().Error("invalid ConnectionId for unsub", zap.Uint32("connId", msg.ConnId))
		return
	}

	hasAccess, accessErr := ctx.Channel.CheckACL(ctx.Connection, ChannelAccessType_Unsub)
	if connToUnsub.id != ctx.Connection.Id() && !hasAccess {
		ctx.Connection.Logger().Error("connection dosen't have access to unsub connection from this channel",
			zap.Uint32("unsubConnId", msg.ConnId),
			zap.String("channelType", ctx.Channel.channelType.String()),
			zap.Uint32("channelId", uint32(ctx.Channel.id)),
			zap.Error(accessErr),
		)
		return
	}

	_, err := connToUnsub.UnsubscribeFromChannel(ctx.Channel)
	if err != nil {
		ctx.Connection.Logger().Error("failed to unsub from channel",
			zap.String("channelType", ctx.Channel.channelType.String()),
			zap.Uint32("channelId", uint32(ctx.Channel.id)),
			zap.Error(err),
		)
		return
	}

	// Notify the sender.
	ctx.Connection.sendUnsubscribed(ctx, ctx.Channel, connToUnsub, ctx.StubId)

	// Notify the unsubscribed.
	if connToUnsub != ctx.Connection {
		connToUnsub.sendUnsubscribed(ctx, ctx.Channel, connToUnsub, 0)
	}
	// Notify the channel owner.
	if ctx.Channel.HasOwner() {
		if ctx.Channel.ownerConnection != ctx.Connection {
			ctx.Channel.ownerConnection.sendUnsubscribed(ctx, ctx.Channel, connToUnsub, 0)
		} else {
			// Reset the owner if it unsubscribed
			ctx.Channel.ownerConnection = nil
		}
	}
}

func handleChannelDataUpdate(ctx MessageContext) {
	// Only channel owner or writable subsciptors can update the data
	if ctx.Channel.ownerConnection != ctx.Connection {
		cs := ctx.Channel.subscribedConnections[ctx.Connection]
		if cs == nil || cs.options.DataAccess != channeldpb.ChannelDataAccess_WRITE_ACCESS {
			ctx.Connection.Logger().Warn("attempt to update channel data but has no access",
				zap.String("channelType", ctx.Channel.channelType.String()),
				zap.Uint32("channelId", uint32(ctx.Channel.id)),
			)
			return
		}
	}

	if ctx.Channel.Data() == nil {
		ctx.Channel.Logger().Info("channel data is not initialized - should send CreateChannelMessage before ChannelDataUpdateMessage",
			zap.Uint32("connId", uint32(ctx.Connection.Id())))
		return
	}

	msg, ok := ctx.Msg.(*channeldpb.ChannelDataUpdateMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a ChannelDataUpdateMessage, will not be handled.")
		return
	}
	updateMsg, err := msg.Data.UnmarshalNew()
	if err != nil {
		ctx.Connection.Logger().Error("failed to unmarshal channel update data", zap.Error(err),
			zap.String("channelType", ctx.Channel.channelType.String()),
			zap.String("typeUrl", msg.Data.TypeUrl))
		return
	}

	if ctx.Channel.spatialNotifier != nil {
		if ctx.Connection.GetConnectionType() == channeldpb.ConnectionType_CLIENT {
			ctx.Channel.SetDataUpdateConnId(ctx.Connection.Id())
		} else {
			ctx.Channel.SetDataUpdateConnId(ConnectionId(msg.ContextConnId))
		}
	}
	ctx.Channel.Data().OnUpdate(updateMsg, ctx.Channel.GetTime(), ctx.Channel.spatialNotifier)
}

func handleDisconnect(ctx MessageContext) {
	if ctx.Channel != globalChannel {
		ctx.Connection.Logger().Error("illegal attemp to disconnect another connection outside the GLOBAL channel")
		return
	}

	msg, ok := ctx.Msg.(*channeldpb.DisconnectMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a DisconnectMessage, will not be handled.")
		return
	}

	connToDisconnect := GetConnection(ConnectionId(msg.ConnId))
	if connToDisconnect == nil {
		ctx.Connection.Logger().Warn("could not find the connection to disconnect",
			zap.Uint32("targetConnId", msg.ConnId),
		)
		return
	}

	if err := connToDisconnect.Disconnect(); err != nil {
		ctx.Connection.Logger().Warn("failed to disconnect a connection",
			zap.Uint32("targetConnId", msg.ConnId),
			zap.String("targetConnType", connToDisconnect.connectionType.String()),
		)
	} else {
		ctx.Connection.Logger().Info("successfully disconnected a connection",
			zap.Uint32("targetConnId", msg.ConnId),
			zap.String("targetConnType", connToDisconnect.connectionType.String()),
		)
	}
	connToDisconnect.Close()
}

func handleQuerySpatialChannel(ctx MessageContext) {
	if ctx.Channel != globalChannel {
		ctx.Connection.Logger().Error("illegal attemp to query spatial channel outside the GLOBAL channel")
		return
	}

	msg, ok := ctx.Msg.(*channeldpb.QuerySpatialChannelMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a QuerySpatialChannelMessage, will not be handled.")
		return
	}

	if spatialController == nil {
		ctx.Connection.Logger().Error("cannot query spatial channel as the spatial controller does not exist")
		return
	}

	channelIds := make([]uint32, len(msg.SpatialInfo))
	for i, info := range msg.SpatialInfo {
		channelId, err := spatialController.GetChannelId(common.SpatialInfo{
			X: info.X,
			Y: info.Y,
			Z: info.Z,
		})
		if err != nil {
			ctx.Connection.Logger().Warn("failed to GetChannelId", zap.Error(err),
				zap.Float64("x", info.X), zap.Float64("y", info.Y), zap.Float64("z", info.Z))
		}
		channelIds[i] = uint32(channelId)
	}

	ctx.Msg = &channeldpb.QuerySpatialChannelResultMessage{
		ChannelId: channelIds,
	}
	ctx.Connection.Send(ctx)
}
