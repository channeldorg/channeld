package channeld

import (
	"log"

	"clewcat.com/channeld/proto"
	protobuf "google.golang.org/protobuf/proto"
)

type Message = protobuf.Message //protoreflect.ProtoMessage
// The parameters of the handler function: 1. the weak-typed Message object popped from the message queue; 2. the connection that received the message; 3. the channel that the message is specified to handle.
type MessageHandlerFunc func(Message, *Connection, *Channel)
type MessageMapEntry struct {
	msg     Message
	handler MessageHandlerFunc
}

var MessageMap = map[proto.MessageType]*MessageMapEntry{
	proto.MessageType_AUTH:                {&proto.AuthMessage{}, handleAuth},
	proto.MessageType_CREATE_CHANNEL:      {&proto.CreateChannelMessage{}, handleCreateChannel},
	proto.MessageType_REMOVE_CHANNEL:      {&proto.RemoveChannelMessage{}, handleRemoveChannel},
	proto.MessageType_SUB_TO_CHANNEL:      {&proto.SubscribedToChannelsMessage{}, handleSubToChannels},
	proto.MessageType_UNSUB_TO_CHANNEL:    {&proto.UnsubscribedToChannelsMessage{}, handleUnsubToChannels},
	proto.MessageType_CHANNEL_DATA_GLOBAL: {&proto.GlobalChannelDataMessage{}, handleChannelDataUpdate},
}

func handleAuth(m Message, c *Connection, ch *Channel) {

}

func handleCreateChannel(m Message, c *Connection, ch *Channel) {
	// Only the WORLD channel can handle channel creation/deletion
	if ch != globalChannel {
		log.Panicln("Illegal attemp to create channel outside the WOLRD channel, connection: ", c)
	}

	msg, ok := m.(*proto.CreateChannelMessage)
	if !ok {
		log.Panicln("Message is not a CreateChannelMessage, will not be handled.")
	}

	var newChannel *Channel
	// World channel is initially created by the system.
	if msg.ChannelType == proto.ChannelType_GLOBAL {
		newChannel = globalChannel
		if globalChannel.ownerConnection == nil {
			globalChannel.ownerConnection = c
		} else {
			log.Panicln("Illegal attempt to create the WOLRD channel, connection: ", c)
		}
	} else {
		newChannel = CreateChannel(msg.ChannelType, c)
	}
	// Subscribe to channel after creation
	c.SubscribeToChannel(newChannel, ChannelSubscriptionOptions{})
	// Also send the Sub message to the creator (no need to broadcast as there's only 1 subscriptor)
	c.sendSubscribed(newChannel.id)
}

func handleRemoveChannel(m Message, c *Connection, ch *Channel) {
	_, ok := m.(*proto.RemoveChannelMessage)
	if !ok {
		log.Panicln("Message is not a RemoveChannelMessage, will not be handled.")
	}

	// Only the owner can remove the channel
	if ch.ownerConnection != c {
		log.Panicf("%s tried to remove %s but it's not the owner.", c, ch)
	}

	for connId := range ch.subscribedConnections {
		GetConnection(connId).sendUnsubscribed(ch.id)
	}
	RemoveChannel(ch)
}

func handleSubToChannels(m Message, c *Connection, ch *Channel) {
	msg, ok := m.(*proto.SubscribedToChannelsMessage)
	if !ok {
		log.Panicln("Message is not a SubscribedToChannelsMessage, will not be handled.")
	}

	// The connection that subscribes. Could be different to c which sends the message.
	connToSub := GetConnection(ConnectionId(msg.ConnId))
	connChannelIds := make(map[*Connection][]ChannelId)
	for id := range msg.ChannelIds {
		ch := GetChannel(ChannelId(id))
		if ch == nil {
			log.Printf("Failed to subscribe to channel %d as it doesn't exist\n", id)
			continue
		}
		err := connToSub.SubscribeToChannel(ch, ChannelSubscriptionOptions{})
		if err != nil {
			log.Printf("Failed to subscribe to channel %d, err: %s\n", id, err)
			continue
		}

		// Optimize to send all channelIds to each connection once
		if ch.ownerConnection != nil {
			channelIds := connChannelIds[ch.ownerConnection]
			if channelIds == nil {
				channelIds = make([]ChannelId, 1)
			}
			channelIds = append(channelIds, ch.id)
			connChannelIds[ch.ownerConnection] = channelIds
		}
	}

	for conn, channelIds := range connChannelIds {
		conn.sendConnSubscribed(ConnectionId(msg.ConnId), channelIds...)
	}
}

func handleUnsubToChannels(m Message, c *Connection, ch *Channel) {
	msg, ok := m.(*proto.UnsubscribedToChannelsMessage)
	if !ok {
		log.Panicln("Message is not a UnsubscribedToChannelsMessage, will not be handled.")
	}

	connToUnsub := GetConnection(ConnectionId(msg.ConnId))
	connChannelIds := make(map[*Connection][]ChannelId)
	for id := range msg.ChannelIds {
		ch := GetChannel(ChannelId(id))
		if ch == nil {
			log.Printf("Failed to unsubscribe to channel %d as it doesn't exist\n", id)
			continue
		}

		err := connToUnsub.UnsubscribeToChannel(ch)
		if err != nil {
			log.Printf("Failed to unsubscribe to channel %d, err: %s\n", id, err)
			continue
		}

		// Optimize to send all channelIds to each connection once
		if ch.ownerConnection != nil {
			channelIds := connChannelIds[ch.ownerConnection]
			if channelIds == nil {
				channelIds = make([]ChannelId, 1)
			}
			channelIds = append(channelIds, ch.id)
			connChannelIds[ch.ownerConnection] = channelIds
		}

		for conn, channelIds := range connChannelIds {
			conn.sendConnUnsubscribed(ConnectionId(msg.ConnId), channelIds...)
		}
	}
}

func handleChannelDataUpdate(m Message, c *Connection, ch *Channel) {
	// Only channel owner or writable subsciptors can update the data
	if ch.ownerConnection != c {
		cs := ch.subscribedConnections[c.id]
		if cs == nil || !cs.options.CanUpdateData {
			log.Panicf("%s tries to update %s but has no access.\n", c, ch)
		}
	}

	ch.Data().Update(m, ch.subscribedConnections[c.id])
}
