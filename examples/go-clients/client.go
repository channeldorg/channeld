package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"channeld.clewcat.com/channeld/proto"
	"github.com/gorilla/websocket"
	protobuf "google.golang.org/protobuf/proto"
)

type Message = protobuf.Message
type MessageHandlerFunc func(client *Client, channelId uint32, m Message)
type messageMapEntry struct {
	msg      Message
	handlers []MessageHandlerFunc
}

type Client struct {
	Id                 uint32
	subscribedChannels []uint32
	conn               net.Conn
	messageMap         map[uint32]*messageMapEntry
	stubCallbacks      map[uint32]MessageHandlerFunc
}

func NewClient(addr string) (*Client, error) {
	var conn net.Conn
	if strings.HasPrefix(addr, "ws") {
		c, _, err := websocket.DefaultDialer.Dial(addr, nil)
		if err != nil {
			return nil, err
		}
		conn = &wsConn{conn: c}
	} else {
		var err error
		conn, err = net.Dial("tcp", addr)
		if err != nil {
			return nil, err
		}
	}
	c := &Client{
		subscribedChannels: make([]uint32, 0),
		conn:               conn,
		messageMap:         make(map[uint32]*messageMapEntry),
		stubCallbacks: map[uint32]MessageHandlerFunc{
			// 0 is Reserved
			0: func(_ *Client, _ uint32, _ Message) {},
		},
	}

	c.SetMessageEntry(uint32(proto.MessageType_AUTH), &proto.AuthResultMessage{}, defaultMessageHandler)
	c.SetMessageEntry(uint32(proto.MessageType_CREATE_CHANNEL), &proto.CreateChannelMessage{}, defaultMessageHandler)
	c.SetMessageEntry(uint32(proto.MessageType_REMOVE_CHANNEL), &proto.RemoveChannelMessage{}, defaultMessageHandler)
	c.SetMessageEntry(uint32(proto.MessageType_AUTH), &proto.AuthResultMessage{}, defaultMessageHandler)
	c.SetMessageEntry(uint32(proto.MessageType_SUB_TO_CHANNEL), &proto.SubscribedToChannelMessage{}, handleSubToChannel)
	c.SetMessageEntry(uint32(proto.MessageType_UNSUB_TO_CHANNEL), &proto.UnsubscribedToChannelMessage{}, handleUnsubToChannel)
	c.SetMessageEntry(uint32(proto.MessageType_LIST_CHANNEL), &proto.ListChannelResultMessage{}, defaultMessageHandler)
	c.SetMessageEntry(uint32(proto.MessageType_CHANNEL_DATA_UPDATE), &proto.ChannelDataUpdateMessage{}, defaultMessageHandler)

	return c, nil
}

func (client *Client) Disconnect() error {
	return client.conn.Close()
}

func (client *Client) SetMessageEntry(msgType uint32, msgTemplate Message, handlers ...MessageHandlerFunc) {
	client.messageMap[msgType] = &messageMapEntry{
		msg:      msgTemplate,
		handlers: handlers,
	}
}

func (client *Client) AddMessageHandler(msgType uint32, handlers ...MessageHandlerFunc) error {
	entry := client.messageMap[msgType]
	if entry != nil {
		entry.handlers = append(entry.handlers, handlers...)
		return nil
	} else {
		return fmt.Errorf("failed to add handler as the message entry not found, msgType: %d", msgType)
	}
}

func (client *Client) Auth(lt string, pit string) chan *proto.AuthResultMessage {
	result := make(chan *proto.AuthResultMessage)
	client.Send(0, proto.BroadcastType_NO, uint32(proto.MessageType_AUTH), &proto.AuthMessage{
		LoginToken:            lt,
		PlayerIdentifierToken: pit,
	}, func(_ *Client, channelId uint32, m Message) {
		msg := m.(*proto.AuthResultMessage)
		client.Id = msg.ConnId
		result <- msg
		if msg.Result == proto.AuthResultMessage_SUCCESSFUL {
			client.Send(0, proto.BroadcastType_NO, uint32(proto.MessageType_SUB_TO_CHANNEL), &proto.SubscribedToChannelMessage{
				ConnId: client.Id,
			}, nil)
		}
	})
	return result
}

func handleSubToChannel(client *Client, channelId uint32, m Message) {
	client.subscribedChannels = append(client.subscribedChannels, channelId)
}

func handleUnsubToChannel(c *Client, channelId uint32, m Message) {
	for i, chid := range c.subscribedChannels {
		if chid == channelId {
			c.subscribedChannels[i] = c.subscribedChannels[len(c.subscribedChannels)-1]
			c.subscribedChannels = c.subscribedChannels[:len(c.subscribedChannels)-1]
			return
		}
	}
}

func defaultMessageHandler(client *Client, channelId uint32, m Message) {
	//log.Printf("Client(%d) received message from channel %d: %s", client.Id, channelId, m)
}

func readBytes(conn net.Conn, len uint) ([]byte, error) {
	bytes := make([]byte, len)
	if _, err := io.ReadFull(conn, bytes); err != nil {
		switch err.(type) {
		case *net.OpError:
		case *websocket.CloseError:
			return nil, fmt.Errorf("WebSocket server disconnected: %s", conn.RemoteAddr())
		}

		if err == io.EOF {
			return nil, fmt.Errorf("server disconnected: %s", conn.RemoteAddr())
		}
		return nil, err
	}
	return bytes, nil
}

func (client *Client) Receive() {
	tag, err := readBytes(client.conn, 4)
	if err != nil || tag[0] != 67 {
		log.Println("Invalid tag:", tag, ", the packet will be dropped.", err)
		return
	}

	packetSize := int(tag[3])
	if tag[1] != 72 {
		packetSize = packetSize | int(tag[1])<<16 | int(tag[2])<<8
	} else if tag[2] != 78 {
		packetSize = packetSize | int(tag[2])<<8
	}

	bytes := make([]byte, packetSize)
	if _, err := io.ReadFull(client.conn, bytes); err != nil {
		log.Panic("Error reading packet: ", err)
	}

	var p proto.Packet
	if err := protobuf.Unmarshal(bytes, &p); err != nil {
		log.Panic("Error unmarshalling packet: ", err)
	}

	entry := client.messageMap[p.MsgType]
	if entry == nil {
		log.Printf("No message type registered: %d", p.MsgType)
		return
	}

	// Always make a clone!
	msg := protobuf.Clone(entry.msg)
	err = protobuf.Unmarshal(p.MsgBody, msg)
	if err != nil {
		log.Panicln(err)
	}

	for _, handler := range entry.handlers {
		handler(client, p.ChannelId, msg)
	}

	if p.StubId != 0 {
		callback := client.stubCallbacks[p.StubId]
		if callback != nil {
			callback(client, p.ChannelId, msg)
		}
	}
}

func (client *Client) Send(channelId uint32, broadcast proto.BroadcastType, msgType uint32, msg Message, callback MessageHandlerFunc) error {
	var stubId uint32 = 0
	if callback != nil {
		for client.stubCallbacks[stubId] != nil {
			stubId++
		}
		client.stubCallbacks[stubId] = callback
	}

	msgBody, err := protobuf.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message %d: %s. Error: %v", msgType, msg, err)
	}
	if len(msgBody) >= (1 << 24) {
		return fmt.Errorf("message body is too large, size: %d. Error: %v", len(msgBody), err)
	}

	bytes, err := protobuf.Marshal(&proto.Packet{
		ChannelId: channelId,
		Broadcast: broadcast,
		StubId:    stubId,
		MsgType:   msgType,
		MsgBody:   msgBody,
	})
	if err != nil {
		return fmt.Errorf("error marshalling packet: %v", err)
	}

	// 'CHNL' in ASCII
	tag := []byte{67, 72, 78, 76}
	len := len(bytes)
	tag[3] = byte(len & 0xff)
	if len > 0xff {
		tag[2] = byte((len >> 8) & 0xff)
	}
	if len > 0xffff {
		tag[1] = byte((len >> 16) & 0xff)
	}
	/* With WebSocket, every Write() sends a message.
	client.conn.Write(tag)
	client.conn.Write(bytes)
	*/
	client.conn.Write(append(tag, bytes...))
	return nil
}

type wsConn struct {
	conn    *websocket.Conn
	readBuf []byte
	readIdx int
}

func (c *wsConn) Read(b []byte) (n int, err error) {
	if c.readBuf == nil || c.readIdx >= len(c.readBuf) {
		_, c.readBuf, err = c.conn.ReadMessage()
		c.readIdx = 0
	}
	n = copy(b, c.readBuf[c.readIdx:])
	c.readIdx += n
	return n, err
}

func (c *wsConn) Write(b []byte) (n int, err error) {
	return len(b), c.conn.WriteMessage(websocket.BinaryMessage, b)
}

func (c *wsConn) Close() error {
	return c.conn.Close()
}

func (c *wsConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *wsConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *wsConn) SetDeadline(t time.Time) error {
	return c.conn.UnderlyingConn().SetDeadline(t)
}

func (c *wsConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *wsConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}
