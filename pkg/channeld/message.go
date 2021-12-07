package channeld

import (
	"strings"

	"channeld.clewcat.com/channeld/proto"
	"go.uber.org/zap"
	protobuf "google.golang.org/protobuf/proto"
)

type Message = protobuf.Message //protoreflect.ProtoMessage
type MessageContext struct {
	MsgType    proto.MessageType
	Msg        Message     // The weak-typed Message object popped from the message queue
	Connection *Connection // The connection that received the message
	Channel    *Channel    // The channel that handling the message
	Broadcast  proto.BroadcastType
	StubId     uint32
	ChannelId  uint32 // The original channelId in the Packet, could be different from Channel.id.
}
type MessageHandlerFunc func(ctx MessageContext)
type messageMapEntry struct {
	msg     Message
	handler MessageHandlerFunc
}

var MessageMap = map[proto.MessageType]*messageMapEntry{
	proto.MessageType_AUTH:                {&proto.AuthMessage{}, handleAuth},
	proto.MessageType_CREATE_CHANNEL:      {&proto.CreateChannelMessage{}, handleCreateChannel},
	proto.MessageType_REMOVE_CHANNEL:      {&proto.RemoveChannelMessage{}, handleRemoveChannel},
	proto.MessageType_LIST_CHANNEL:        {&proto.ListChannelMessage{}, handleListChannel},
	proto.MessageType_SUB_TO_CHANNEL:      {&proto.SubscribedToChannelMessage{}, handleSubToChannel},
	proto.MessageType_UNSUB_TO_CHANNEL:    {&proto.UnsubscribedToChannelMessage{}, handleUnsubToChannel},
	proto.MessageType_CHANNEL_DATA_UPDATE: {&proto.ChannelDataUpdateMessage{}, handleChannelDataUpdate},
}

func RegisterMessageHandler(msgType uint32, msg Message, handler MessageHandlerFunc) {
	MessageMap[proto.MessageType(msgType)] = &messageMapEntry{msg, handler}
}

// Forward or broadcast user-space messages
func handleUserSpaceMessage(ctx MessageContext) {
	if ctx.Connection.connectionType == CLIENT {
		if ctx.Channel.ownerConnection != nil {
			ctx.Channel.ownerConnection.Send(ctx)
		} else if ctx.Broadcast != proto.BroadcastType_NO {
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
				zap.Uint32("connId", uint32(ctx.Connection.id)),
			)
		}
	} else {
		switch ctx.Broadcast {
		case proto.BroadcastType_NO:
			if ctx.Channel.ownerConnection != nil {
				ctx.Channel.ownerConnection.Send(ctx)
			} else {
				ctx.Connection.Logger().Error("cannot forward the message as the channel has no owner",
					zap.Uint32("msgType", uint32(ctx.MsgType)),
					zap.String("channelType", ctx.Channel.channelType.String()),
					zap.Uint32("channelId", uint32(ctx.Channel.id)),
				)
			}

		case proto.BroadcastType_ALL, proto.BroadcastType_ALL_BUT_SENDER:
			ctx.Channel.Broadcast(ctx)

		case proto.BroadcastType_SINGLE_CONNECTION:
			conn := GetConnection(ConnectionId(ctx.ChannelId))
			if conn != nil {
				conn.Send(ctx)
			} else {
				ctx.Connection.Logger().Error("cannot forward the message as the target connection does not exist",
					zap.Uint32("msgType", uint32(ctx.MsgType)),
					zap.Uint32("targetConnId", ctx.ChannelId),
				)
			}
		}
	}
}

func handleAuth(ctx MessageContext) {
	if ctx.Channel != globalChannel {
		ctx.Connection.Logger().Error("illegal attemp to authenticate outside the GLOBAL channel")
		return
	}
	_, ok := ctx.Msg.(*proto.AuthMessage)
	if !ok {
		ctx.Connection.Logger().Error("mssage is not a AuthMessage, will not be handled.")
		return
	}
	//log.Printf("Auth PIT: %s, LT: %s\n", msg.PlayerIdentifierToken, msg.LoginToken)

	// TODO: Authentication

	ctx.Connection.fsm.MoveToNextState()

	ctx.Msg = &proto.AuthResultMessage{
		Result: proto.AuthResultMessage_SUCCESSFUL,
		ConnId: uint32(ctx.Connection.id),
	}
	ctx.Connection.Send(ctx)
}

func handleCreateChannel(ctx MessageContext) {
	// Only the GLOBAL channel can handle channel creation/deletion/listing
	if ctx.Channel != globalChannel {
		ctx.Connection.Logger().Error("illegal attemp to create channel outside the GLOBAL channel")
		return
	}

	msg, ok := ctx.Msg.(*proto.CreateChannelMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a CreateChannelMessage, will not be handled.")
		return
	}

	var newChannel *Channel
	var err error
	if msg.ChannelType == proto.ChannelType_UNKNOWN {
		ctx.Connection.Logger().Error("illegal attemp to create the UNKNOWN channel")
		return
	} else if msg.ChannelType == proto.ChannelType_GLOBAL {
		// Global channel is initially created by the system. Creating the channel will attempt to own it.
		newChannel = globalChannel
		if globalChannel.ownerConnection == nil {
			globalChannel.ownerConnection = ctx.Connection
		} else {
			ctx.Connection.Logger().Error("illegal attemp to create the GLOBAL channel")
			return
		}
	} else {
		newChannel, err = CreateChannel(msg.ChannelType, ctx.Connection)
		if err != nil {
			ctx.Connection.Logger().Error("failed to create channel",
				zap.Uint32("channelType", uint32(msg.ChannelType)),
				zap.Error(err),
			)
			return
		}
	}
	newChannel.Logger().Info("created channel with owner", zap.Uint32("ownerConnId", uint32(newChannel.ownerConnection.id)))

	newChannel.metadata = msg.Metadata
	if msg.Data != nil {
		dataMsg, err := msg.Data.UnmarshalNew()
		if err != nil {
			newChannel.Logger().Error("failed to unmarshal data message for the new channel", zap.Error(err))
			return
		} else {
			newChannel.InitData(dataMsg, nil)
		}
	}

	// Subscribe to channel after creation
	ctx.Connection.SubscribeToChannel(newChannel, msg.SubOptions)
	// Also send the Sub message to the creator (no need to broadcast as there's only 1 subscriber)
	ctx.Connection.sendSubscribed(ctx, newChannel, ctx.StubId)
}

func handleRemoveChannel(ctx MessageContext) {
	if ctx.Channel != globalChannel {
		ctx.Connection.Logger().Error("illegal attemp to remove channel outside the GLOBAL channel")
		return
	}

	msg, ok := ctx.Msg.(*proto.RemoveChannelMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a RemoveChannelMessage, will not be handled.")
		return
	}

	channelToRemove := GetChannel(ChannelId(msg.ChannelId))
	if channelToRemove == nil {
		ctx.Connection.Logger().Error("invalid channelId for removing", zap.Uint32("channelId", msg.ChannelId))
		return
	}
	// Only the owner can remove the channel
	if channelToRemove.ownerConnection != ctx.Connection {
		ownerConnId := uint32(0)
		if channelToRemove.ownerConnection != nil {
			ownerConnId = uint32(channelToRemove.ownerConnection.id)
		}
		ctx.Connection.Logger().Error("illegal attemp to remove channel as the connection is not the channel owner",
			zap.String("channelType", channelToRemove.channelType.String()),
			zap.Uint32("channelId", uint32(channelToRemove.id)),
			zap.Uint32("ownerConnId", ownerConnId),
		)
		return
	}

	for connId := range channelToRemove.subscribedConnections {
		sc := GetConnection(connId)
		if sc != nil {
			//sc.sendUnsubscribed(ctx, channelToRemove, 0)
			respond := ctx
			respond.StubId = 0
			sc.Send(respond)
		}
	}
	RemoveChannel(channelToRemove)

	ctx.Connection.Logger().Info("removed channel",
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

	msg, ok := ctx.Msg.(*proto.ListChannelMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a ListChannelMessage, will not be handled.")
		return
	}

	result := make([]*proto.ListChannelResultMessage_ChannelInfo, 0)
	allChannels.Range(func(k interface{}, v interface{}) bool {
		channel := v.(*Channel)
		if msg.TypeFilter != proto.ChannelType_UNKNOWN && msg.TypeFilter != channel.channelType {
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
			result = append(result, &proto.ListChannelResultMessage_ChannelInfo{
				ChannelId:   uint32(channel.id),
				ChannelType: channel.channelType,
				Metadata:    channel.metadata,
			})
		}
		return true
	})

	ctx.Msg = &proto.ListChannelResultMessage{
		Channels: result,
	}
	ctx.Connection.Send(ctx)
}

func handleSubToChannel(ctx MessageContext) {
	msg, ok := ctx.Msg.(*proto.SubscribedToChannelMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a SubscribedToChannelMessage, will not be handled.")
		return
	}

	// The connection that subscribes. Could be different to the connection that sends the message.
	connToSub := GetConnection(ConnectionId(msg.ConnId))
	if connToSub == nil {
		ctx.Connection.Logger().Error("invalid ConnectionId for sub", zap.Uint32("connId", msg.ConnId))
		return
	}

	if connToSub.id != ctx.Connection.id && ctx.Connection != ctx.Channel.ownerConnection {
		ctx.Connection.Logger().Error("illegal attemp to sub another connection as the sender is not the channel owener",
			zap.Uint32("subConnId", msg.ConnId),
			zap.String("channelType", ctx.Channel.channelType.String()),
			zap.Uint32("channelId", uint32(ctx.Channel.id)),
		)
		return
	}

	err := connToSub.SubscribeToChannel(ctx.Channel, msg.SubOptions)
	if err != nil {
		ctx.Connection.Logger().Error("failed to sub to channel",
			zap.String("channelType", ctx.Channel.channelType.String()),
			zap.Uint32("channelId", uint32(ctx.Channel.id)),
			zap.Error(err),
		)
		return
	}

	connToSub.sendSubscribed(ctx, ctx.Channel, ctx.StubId)
	if ctx.Channel.ownerConnection != nil {
		ctx.Channel.ownerConnection.sendSubscribed(ctx, ctx.Channel, 0)
	}
}

func handleUnsubToChannel(ctx MessageContext) {
	msg, ok := ctx.Msg.(*proto.UnsubscribedToChannelMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a UnsubscribedToChannelMessage, will not be handled.")
		return
	}

	// The connection that unsubscribes. Could be different to c which sends the message.
	connToUnsub := GetConnection(ConnectionId(msg.ConnId))
	if connToUnsub == nil {
		ctx.Connection.Logger().Error("invalid ConnectionId for unsub", zap.Uint32("connId", msg.ConnId))
		return
	}
	err := connToUnsub.UnsubscribeToChannel(ctx.Channel)
	if err != nil {
		ctx.Connection.Logger().Error("failed to unsub to channel",
			zap.String("channelType", ctx.Channel.channelType.String()),
			zap.Uint32("channelId", uint32(ctx.Channel.id)),
			zap.Error(err),
		)
		return
	}

	connToUnsub.sendUnsubscribed(ctx, ctx.Channel, ctx.StubId)
	if ctx.Channel.ownerConnection != nil {
		if ctx.Channel.ownerConnection == connToUnsub {
			// Reset the owner if it unsubscribed
			ctx.Channel.ownerConnection = nil
		} else {
			ctx.Channel.ownerConnection.sendUnsubscribed(ctx, ctx.Channel, 0)
		}
	}
}

func handleChannelDataUpdate(ctx MessageContext) {
	// Only channel owner or writable subsciptors can update the data
	if ctx.Channel.ownerConnection != ctx.Connection {
		cs := ctx.Channel.subscribedConnections[ctx.Connection.id]
		if cs == nil || !cs.options.CanUpdateData {
			ctx.Connection.Logger().Error("attempt to update channel data but has no access",
				zap.String("channelType", ctx.Channel.channelType.String()),
				zap.Uint32("channelId", uint32(ctx.Channel.id)),
			)
			return
		}
	}

	msg, ok := ctx.Msg.(*proto.ChannelDataUpdateMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not a ChannelDataUpdateMessage, will not be handled.")
		return
	}
	updateMsg, err := msg.Data.UnmarshalNew()
	if err != nil {
		ctx.Connection.Logger().Error("failed to unmarshal channel update data", zap.Error(err))
		return
	}

	if ctx.Channel.Data() == nil {
		ctx.Channel.InitData(updateMsg, nil)
		ctx.Channel.Logger().Info("initialized channel data from update msg", zap.Any("msg", updateMsg))
	} else {
		ctx.Channel.Data().OnUpdate(updateMsg, ctx.Channel.GetTime())
	}
}
