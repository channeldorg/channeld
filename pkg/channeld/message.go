package channeld

import (
	"log"
	"strings"

	"channeld.clewcat.com/channeld/proto"
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

func handleUserSpaceMessage(ctx MessageContext) {
	// Forward or broadcast user-space messages
	if ctx.Connection.connectionType == CLIENT {
		if ctx.Channel.ownerConnection != nil {
			ctx.Channel.ownerConnection.Send(ctx)
		} else if ctx.Broadcast != proto.BroadcastType_NO {
			if ctx.Channel.enableClientBroadcast {
				ctx.Channel.Broadcast(ctx)
			} else {
				log.Panicf("%s attempted to broadcast message(%d) while %s's client broadcasting is disabled.\n", ctx.Connection, ctx.MsgType, ctx.Channel)
			}
		} else {
			log.Printf("%s sent a user-space message(%d) but %s has no owner to forward the message.\n", ctx.Connection, ctx.MsgType, ctx.Channel)
		}
	} else {
		ctx.Channel.Broadcast(ctx)
	}
}

func handleAuth(ctx MessageContext) {
	if ctx.Channel != globalChannel {
		log.Panicln("Illegal attemp to authenticate outside the GLOBAL channel: ", ctx.Connection)
	}
	_, ok := ctx.Msg.(*proto.AuthMessage)
	if !ok {
		log.Panicln("Message is not a AuthMessage, will not be handled.")
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
		log.Panicln("Illegal attemp to create channel outside the GLOBAL channel, connection: ", ctx.Connection)
	}

	msg, ok := ctx.Msg.(*proto.CreateChannelMessage)
	if !ok {
		log.Panicln("Message is not a CreateChannelMessage, will not be handled.")
	}

	var newChannel *Channel
	if msg.ChannelType == proto.ChannelType_UNKNOWN {
		log.Panicln("Illegal attempt to create the UNKNOWN channel, connection: ", ctx.Connection)
	} else if msg.ChannelType == proto.ChannelType_GLOBAL {
		// Global channel is initially created by the system. Creating the channel will attempt to own it.
		newChannel = globalChannel
		if globalChannel.ownerConnection == nil {
			globalChannel.ownerConnection = ctx.Connection
		} else {
			log.Panicln("Illegal attempt to create the GLOBAL channel, connection: ", ctx.Connection)
		}
	} else {
		newChannel = CreateChannel(msg.ChannelType, ctx.Connection)
	}

	newChannel.metadata = msg.Metadata
	if msg.Data != nil {
		dataMsg, err := msg.Data.UnmarshalNew()
		if err != nil {
			log.Printf("Failed to unmarshal data message when creating %s, error: %s\n", newChannel, err)
		} else {
			newChannel.InitData(dataMsg, nil)
		}
	}

	// Subscribe to channel after creation
	ctx.Connection.SubscribeToChannel(newChannel, msg.SubOptions)
	// Also send the Sub message to the creator (no need to broadcast as there's only 1 subscriptor)
	ctx.Connection.sendSubscribed(ctx, newChannel)
}

func handleRemoveChannel(ctx MessageContext) {
	if ctx.Channel != globalChannel {
		log.Panicln("Illegal attemp to remove channel outside the GLOBAL channel, connection: ", ctx.Connection)
	}

	_, ok := ctx.Msg.(*proto.RemoveChannelMessage)
	if !ok {
		log.Panicln("Message is not a RemoveChannelMessage, will not be handled.")
	}

	// Only the owner can remove the channel
	if ctx.Channel.ownerConnection != ctx.Connection {
		log.Panicf("%s tried to remove %s but it's not the owner.", ctx.Connection, ctx.Channel)
	}

	for connId := range ctx.Channel.subscribedConnections {
		sc := GetConnection(connId)
		sc.sendUnsubscribed(ctx, ctx.Channel)
		//sc.Flush()
	}
	RemoveChannel(ctx.Channel)
}

func handleListChannel(ctx MessageContext) {
	if ctx.Channel != globalChannel {
		log.Panicln("Illegal attemp to list channel outside the GLOBAL channel, connection: ", ctx.Connection)
	}

	msg, ok := ctx.Msg.(*proto.ListChannelMessage)
	if !ok {
		log.Panicln("Message is not a ListChannelMessage, will not be handled.")
	}

	result := make([]*proto.ListChannelResultMessage_ChannelInfo, 0)
	for _, channel := range allChannels {
		if msg.TypeFilter != proto.ChannelType_UNKNOWN && msg.TypeFilter != channel.channelType {
			continue
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
	}

	ctx.Msg = &proto.ListChannelResultMessage{
		Channels: result,
	}
	ctx.Connection.Send(ctx)
}

func handleSubToChannel(ctx MessageContext) {
	msg, ok := ctx.Msg.(*proto.SubscribedToChannelMessage)
	if !ok {
		log.Panicln("Message is not a SubscribedToChannelMessage, will not be handled.")
	}

	// The connection that subscribes. Could be different to the connection that sends the message.
	connToSub := GetConnection(ConnectionId(msg.ConnId))
	if connToSub == nil {
		log.Panicln("Invalid ConnectionId:", msg.ConnId)
	}

	if connToSub.id != ctx.Connection.id && ctx.Connection != ctx.Channel.ownerConnection {
		log.Panicf("%s is not the channel owner but tried to subscribe %s to %s\n", ctx.Connection, connToSub, ctx.Channel)
	}

	err := connToSub.SubscribeToChannel(ctx.Channel, msg.SubOptions)
	if err != nil {
		log.Panicf("%s failed to subscribe to %s, error: %s\n", connToSub, ctx.Channel, err)
	}

	connToSub.sendSubscribed(ctx, ctx.Channel)
	if ctx.Channel.ownerConnection != nil {
		ctx.Channel.ownerConnection.sendSubscribed(ctx, ctx.Channel)
	}
}

func handleUnsubToChannel(ctx MessageContext) {
	msg, ok := ctx.Msg.(*proto.UnsubscribedToChannelMessage)
	if !ok {
		log.Panicln("Message is not a UnsubscribedToChannelMessage, will not be handled.")
	}

	// The connection that unsubscribes. Could be different to c which sends the message.
	connToUnsub := GetConnection(ConnectionId(msg.ConnId))
	if connToUnsub == nil {
		log.Panicln("Invalid ConnectionId:", msg.ConnId)
	}
	err := connToUnsub.UnsubscribeToChannel(ctx.Channel)
	if err != nil {
		log.Panicf("%s failed to unsubscribe to %s, error: %s\n", connToUnsub, ctx.Channel, err)
	}

	connToUnsub.sendUnsubscribed(ctx, ctx.Channel)
	if ctx.Channel.ownerConnection != nil {
		if ctx.Channel.ownerConnection == connToUnsub {
			// Reset the owner if it unsubscribed
			ctx.Channel.ownerConnection = nil
		} else {
			ctx.Channel.ownerConnection.sendUnsubscribed(ctx, ctx.Channel)
		}
	}
}

func handleChannelDataUpdate(ctx MessageContext) {
	// Only channel owner or writable subsciptors can update the data
	if ctx.Channel.ownerConnection != ctx.Connection {
		cs := ctx.Channel.subscribedConnections[ctx.Connection.id]
		if cs == nil || !cs.options.CanUpdateData {
			log.Panicf("%s tries to update %s but has no access.\n", ctx.Connection, ctx.Channel)
		}
	}

	msg, ok := ctx.Msg.(*proto.ChannelDataUpdateMessage)
	if !ok {
		log.Panicln("Message is not a ChannelDataUpdateMessage, will not be handled.")
	}
	updateMsg, err := msg.Data.UnmarshalNew()
	if err != nil {
		log.Panicln(err)
	}

	if ctx.Channel.Data() == nil {
		ctx.Channel.InitData(updateMsg, nil)
		log.Printf("%s initialized data from update msg: %s\n", ctx.Channel, updateMsg)
	} else {
		ctx.Channel.Data().OnUpdate(updateMsg, ctx.Channel.GetTime())
	}
}
