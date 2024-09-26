package channeld

import (
	"strings"

	"github.com/metaworking/channeld/pkg/channeldpb"
	"github.com/metaworking/channeld/pkg/common"
	"go.uber.org/zap"
)

// The context of a message for both sending and receiving
type MessageContext struct {
	MsgType channeldpb.MessageType
	// The weak-typed Message object popped from the message queue
	Msg       common.Message
	Broadcast uint32 //channeldpb.BroadcastType
	StubId    uint32
	// The original channelId in the Packet, could be different from Channel.id.
	// Used for both send and receive.
	ChannelId uint32

	// The connection that received the message. Required for BroadcastType_ALL_BUT_SENDER but not for sending.
	Connection ConnectionInChannel
	// The channel that handling the message. Not required for sending or broadcasting.
	Channel *Channel
	// Internally used for receiving
	arrivalTime ChannelTime
}

func (ctx *MessageContext) HasConnection() bool {
	conn, ok := ctx.Connection.(*Connection)
	return ok && conn != nil && !conn.IsClosing()
}

type MessageHandlerFunc func(ctx MessageContext)
type messageMapEntry struct {
	msg     common.Message
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
	channeldpb.MessageType_UPDATE_SPATIAL_INTEREST:   {&channeldpb.UpdateSpatialInterestMessage{}, handleUpdateSpatialInterest},
	channeldpb.MessageType_CREATE_ENTITY_CHANNEL:     {&channeldpb.CreateEntityChannelMessage{}, handleCreateEntityChannel},
	channeldpb.MessageType_ENTITY_GROUP_ADD:          {&channeldpb.AddEntityGroupMessage{}, handleAddEntityGroup},
	channeldpb.MessageType_ENTITY_GROUP_REMOVE:       {&channeldpb.RemoveEntityGroupMessage{}, handleRemoveEntityGroup},
}

func RegisterMessageHandler(msgType uint32, msg common.Message, handler MessageHandlerFunc) {
	MessageMap[channeldpb.MessageType(msgType)] = &messageMapEntry{msg, handler}
}

func handleClientToServerUserMessage(ctx MessageContext) {
	msg, ok := ctx.Msg.(*channeldpb.ServerForwardMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a ServerForwardMessage, will not be handled.")
		return
	}

	var channelOwnerConnId uint32 = 0
	if ownerConn := ctx.Channel.GetOwner(); ownerConn != nil && !ownerConn.IsClosing() {
		if ownerConn.ShouldRecover() {
			ctx.Connection.Logger().Verbose("dropp the client message as the channel owner connection is in recovery",
				zap.Uint32("msgType", uint32(ctx.MsgType)),
				zap.String("channelType", ctx.Channel.channelType.String()),
				zap.Uint32("channelId", uint32(ctx.Channel.id)),
			)
			return
		}
		ownerConn.Send(ctx)
		channelOwnerConnId = uint32(ownerConn.Id())
	} else if ctx.Broadcast > 0 {
		if ctx.Channel.enableClientBroadcast {
			ctx.Channel.Broadcast(ctx)
		} else {
			ctx.Connection.Logger().Error("illegal attempt to broadcast message as the channel's client broadcasting is disabled",
				zap.Uint32("msgType", uint32(ctx.MsgType)),
				zap.String("channelType", ctx.Channel.channelType.String()),
				zap.Uint32("channelId", uint32(ctx.Channel.id)),
			)
			return
		}
	} else {
		if len(ctx.Channel.recoverableSubs) == 0 {
			ctx.Channel.Logger().Warn("channel has no owner to forward the user-space messaged",
				zap.Uint32("msgType", uint32(ctx.MsgType)),
				zap.Uint32("connId", uint32(ctx.Connection.Id())),
			)
		}
		return
	}

	if len(msg.Payload) < 128 {
		ctx.Connection.Logger().Verbose("forward user-space message from client to server",
			zap.Uint32("msgType", uint32(ctx.MsgType)),
			zap.Uint32("clientConnId", msg.ClientConnId),
			zap.Uint32("channelId", uint32(ctx.Channel.id)),
			zap.Uint32("channelOwner", channelOwnerConnId),
			zap.Uint32("broadcastType", ctx.Broadcast),
			zap.Int("payloadSize", len(msg.Payload)),
		)
	} else {
		ctx.Connection.Logger().Debug("forward user-space message from client to server",
			zap.Uint32("msgType", uint32(ctx.MsgType)),
			zap.Uint32("clientConnId", msg.ClientConnId),
			zap.Uint32("channelId", uint32(ctx.Channel.id)),
			zap.Uint32("channelOwner", channelOwnerConnId),
			zap.Uint32("broadcastType", ctx.Broadcast),
			zap.Int("payloadSize", len(msg.Payload)),
		)
	}
}

func HandleServerToClientUserMessage(ctx MessageContext) {
	msg, ok := ctx.Msg.(*channeldpb.ServerForwardMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a ServerForwardMessage, will not be handled.")
		return
	}

	if len(msg.Payload) < 128 {
		ctx.Connection.Logger().Verbose("forward user-space message from server to client/server",
			zap.Uint32("msgType", uint32(ctx.MsgType)),
			zap.Uint32("clientConnId", msg.ClientConnId),
			zap.Uint32("channelId", uint32(ctx.Channel.id)),
			zap.Uint32("broadcastType", ctx.Broadcast),
			zap.Int("payloadSize", len(msg.Payload)),
		)
	} else {
		ctx.Connection.Logger().Debug("forward user-space message from server to client/server",
			zap.Uint32("msgType", uint32(ctx.MsgType)),
			zap.Uint32("clientConnId", msg.ClientConnId),
			zap.Uint32("channelId", uint32(ctx.Channel.id)),
			zap.Uint32("broadcastType", ctx.Broadcast),
			zap.Int("payloadSize", len(msg.Payload)),
		)

	}

	switch channeldpb.BroadcastType(ctx.Broadcast) {
	case channeldpb.BroadcastType_NO_BROADCAST:
		if !ctx.Channel.SendToOwner(ctx) {
			ctx.Connection.Logger().Error("cannot forward the message as the channel has no owner",
				zap.Uint32("msgType", uint32(ctx.MsgType)),
				zap.String("channelType", ctx.Channel.channelType.String()),
				zap.Uint32("channelId", uint32(ctx.Channel.id)),
			)
		}

		/*
			case channeldpb.BroadcastType_ALL, channeldpb.BroadcastType_ALL_BUT_SENDER, channeldpb.BroadcastType_ALL_BUT_OWNER,
				channeldpb.BroadcastType_ALL_BUT_CLIENT, channeldpb.BroadcastType_ALL_BUT_SERVER:
				ctx.Channel.Broadcast(ctx)
		*/
	case channeldpb.BroadcastType_SINGLE_CONNECTION:
		var conn *Connection = nil
		if msg.ClientConnId == 0 {
			// server to server
			conn = ctx.Channel.GetOwner().(*Connection)
		} else {
			// server to client
			conn = GetConnection(ConnectionId(msg.ClientConnId))
		}

		if conn != nil && !conn.IsClosing() {
			conn.Send(ctx)
		} else {
			ctx.Connection.Logger().Info("drop the forward message as the target connection does not exist",
				zap.Uint32("msgType", uint32(ctx.MsgType)),
				zap.Uint32("targetConnId", msg.ClientConnId),
			)
		}

	default:
		if ctx.Broadcast >= uint32(channeldpb.BroadcastType_ALL) && ctx.Broadcast < uint32(channeldpb.BroadcastType_ADJACENT_CHANNELS) {
			ctx.Channel.Broadcast(ctx)
		} else if channeldpb.BroadcastType_ADJACENT_CHANNELS.Check(ctx.Broadcast) {
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
				// Ignore the clients?
				if channeldpb.BroadcastType_ALL_BUT_CLIENT.Check(ctx.Broadcast) && conn.GetConnectionType() == channeldpb.ConnectionType_CLIENT {
					continue
				}
				// Ignore the servers?
				if channeldpb.BroadcastType_ALL_BUT_SERVER.Check(ctx.Broadcast) && conn.GetConnectionType() == channeldpb.ConnectionType_SERVER {
					continue
				}
				// Ignore the client specified in the ServerForwardMessage
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
		ctx.Connection.Close(nil)
		return
	}
	msg, ok := ctx.Msg.(*channeldpb.AuthMessage)
	if !ok {
		ctx.Connection.Logger().Error("mssage is not an AuthMessage, will not be handled.")
		ctx.Connection.Close(nil)
		return
	}
	//log.Printf("Auth PIT: %s, LT: %s\n", msg.PlayerIdentifierToken, msg.LoginToken)

	_, banned := pitBlacklist[msg.PlayerIdentifierToken]
	if banned {
		securityLogger.Info("refused authentication of banned PIT", zap.String("pit", msg.PlayerIdentifierToken))
		ctx.Connection.Close(nil)
		return
	}

	if authProvider == nil && !GlobalSettings.Development {
		rootLogger.Panic("no auth provider")
		return
	}

	authResult := channeldpb.AuthResultMessage_SUCCESSFUL
	if ctx.Connection.GetConnectionType() == channeldpb.ConnectionType_SERVER && GlobalSettings.ServerBypassAuth {
		onAuthComplete(ctx, authResult, msg.PlayerIdentifierToken)
	} else if authProvider != nil {
		go func() {
			authResult, err := authProvider.DoAuth(ctx.Connection.Id(), msg.PlayerIdentifierToken, msg.LoginToken)
			if err != nil {
				ctx.Connection.Logger().Error("failed to do auth", zap.Error(err))
				ctx.Connection.Close(nil)
			} else {
				onAuthComplete(ctx, authResult, msg.PlayerIdentifierToken)
			}
		}()
	} else {
		onAuthComplete(ctx, authResult, msg.PlayerIdentifierToken)
	}
}

func onAuthComplete(ctx MessageContext, authResult channeldpb.AuthResultMessage_AuthResult, pit string) {
	if ctx.Connection.IsClosing() {
		return
	}

	if authResult == channeldpb.AuthResultMessage_SUCCESSFUL {
		ctx.Connection.OnAuthenticated(pit)
	}

	ctx.Msg = &channeldpb.AuthResultMessage{
		Result:          authResult,
		ConnId:          uint32(ctx.Connection.Id()),
		CompressionType: GlobalSettings.CompressionType,
		ShouldRecover:   ctx.Connection.ShouldRecover(),
	}
	ctx.Connection.Send(ctx)

	// Also send the respond to The GLOBAL channel owner (to handle the client's subscription if it doesn't have the authority to).
	if globalChannel.HasOwner() {
		ctx.StubId = 0
		globalChannel.SendToOwner(ctx)
	}

	Event_AuthComplete.Broadcast(AuthEventData{
		AuthResult:            authResult,
		Connection:            ctx.Connection,
		PlayerIdentifierToken: pit,
	})
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
			globalChannel.SetOwner(ctx.Connection)
			Event_GlobalChannelPossessed.Broadcast(globalChannel)
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
		newChannel.Logger().Info("created channel with owner", zap.Uint32("ownerConnId", uint32(newChannel.GetOwner().Id())))
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
	if globalChannel.GetOwner() != ctx.Connection && globalChannel.HasOwner() {
		ctx.StubId = 0
		globalChannel.SendToOwner(ctx)
	}

	// Subscribe to channel after creation
	cs, _ := ctx.Connection.SubscribeToChannel(newChannel, msg.SubOptions)
	if cs != nil {
		ctx.Connection.sendSubscribed(ctx, newChannel, ctx.Connection, 0, &cs.options)
	}
}

func handleRemoveChannel(ctx MessageContext) {

	msg, ok := ctx.Msg.(*channeldpb.RemoveChannelMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a RemoveChannelMessage, will not be handled.")
		return
	}

	channelToRemove := GetChannel(common.ChannelId(msg.ChannelId))
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
			ownerConnId = uint32(channelToRemove.GetOwner().Id())
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

	var logger *Logger
	if ctx.HasConnection() {
		logger = ctx.Connection.Logger()
	} else {
		logger = RootLogger()
	}
	logger.Info("removed channel",
		zap.String("channelType", channelToRemove.channelType.String()),
		zap.Uint32("channelId", uint32(channelToRemove.id)),
		zap.Int("subs", len(channelToRemove.subscribedConnections)),
	)
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
	allChannels.Range(func(_ common.ChannelId, channel *Channel) bool {
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

	var connToSub *Connection
	if ctx.Connection.GetConnectionType() == channeldpb.ConnectionType_CLIENT {
		connToSub = ctx.Connection.(*Connection)
	} else {
		// Only the server can specify a ConnId.
		connToSub = GetConnection(ConnectionId(msg.ConnId))
	}

	if connToSub == nil {
		ctx.Connection.Logger().Error("invalid ConnectionId for sub", zap.Uint32("connIdInMsg", msg.ConnId))
		return
	}

	hasAccess, err := ctx.Channel.CheckACL(ctx.Connection, ChannelAccessType_Sub)
	if connToSub.Id() != ctx.Connection.Id() && !hasAccess {
		ctx.Connection.Logger().Warn("connection doesn't have access to sub connection to this channel",
			zap.Uint32("subConnId", msg.ConnId),
			zap.String("channelType", ctx.Channel.channelType.String()),
			zap.Uint32("channelId", uint32(ctx.Channel.id)),
			zap.Error(err),
		)
		return
	}

	/*
		cs, exists := ctx.Channel.subscribedConnections[connToSub]
		if exists {
			ctx.Connection.Logger().Debug("already subscribed to channel, the subscription options will be merged",
				zap.String("channelType", ctx.Channel.channelType.String()),
				zap.Uint32("channelId", uint32(ctx.Channel.id)),
			)
			if msg.SubOptions != nil {
				proto.Merge(&cs.options, msg.SubOptions)
			}
			connToSub.sendSubscribed(ctx, ctx.Channel, connToSub, 0, &cs.options)
			// Do not send the SubscribedToChannelResultMessage to the sender or channel owner if already subed.
			return
		}
	*/

	cs, shouldSend := connToSub.SubscribeToChannel(ctx.Channel, msg.SubOptions)
	if !shouldSend {
		return
	}

	// Always notify the sender - may need to update the sub options.
	ctx.Connection.sendSubscribed(ctx, ctx.Channel, connToSub, ctx.StubId, &cs.options)

	// Notify the subscribed if it's not the sender.
	if connToSub != ctx.Connection {
		connToSub.sendSubscribed(ctx, ctx.Channel, connToSub, 0, &cs.options)
	}

	// Notify the channel owner if not already subed and it's not the sender.
	if ownerConn := ctx.Channel.GetOwner(); ownerConn != nil && ownerConn != ctx.Connection && !ownerConn.IsClosing() {
		ownerConn.sendSubscribed(ctx, ctx.Channel, connToSub, 0, &cs.options)
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
		ctx.Connection.Logger().Warn("failed to unsub from channel",
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
	if ownerConn := ctx.Channel.GetOwner(); ownerConn != nil && !ownerConn.IsClosing() {
		if ownerConn != ctx.Connection && ownerConn != connToUnsub {
			ownerConn.sendUnsubscribed(ctx, ctx.Channel, connToUnsub, 0)
		} else if ownerConn == connToUnsub {
			// Reset the owner if it unsubscribed itself
			ctx.Channel.SetOwner(nil)
		}
	}
}

func handleChannelDataUpdate(ctx MessageContext) {
	// Only channel owner or writable subsciptors can update the data
	if ownerConn := ctx.Channel.GetOwner(); ownerConn != ctx.Connection {
		cs := ctx.Channel.subscribedConnections[ctx.Connection]
		if cs == nil || *cs.options.DataAccess != channeldpb.ChannelDataAccess_WRITE_ACCESS {
			if ctx.Connection.GetConnectionType() == channeldpb.ConnectionType_SERVER && ownerConn != nil && !ownerConn.IsClosing() {
				// Quick fix: set the sender to the channel owner if it's a server connection
				ctx.Connection = ownerConn
			} else {
				ctx.Connection.Logger().Warn("attempt to update channel data but has no access",
					zap.String("channelType", ctx.Channel.channelType.String()),
					zap.Uint32("channelId", uint32(ctx.Channel.id)),
				)
				return
			}
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
	ctx.Channel.Data().OnUpdate(updateMsg, ctx.arrivalTime, ctx.Connection.Id(), ctx.Channel.spatialNotifier)
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
	connToDisconnect.Close(nil)
}
