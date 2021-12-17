package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
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
type messageQueueEntry struct {
	msg       Message
	channelId uint32
	stubId    uint32
	handlers  []MessageHandlerFunc
}

type Client struct {
	Id                 uint32
	subscribedChannels map[uint32]struct{}
	conn               net.Conn
	incomingQueue      chan messageQueueEntry
	outgoingQueue      chan *proto.MessagePack
	messageMap         map[uint32]*messageMapEntry
	stubCallbacks      map[uint32]MessageHandlerFunc
	writeMutex         sync.Mutex
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
		subscribedChannels: make(map[uint32]struct{}),
		conn:               conn,
		incomingQueue:      make(chan messageQueueEntry, 128),
		outgoingQueue:      make(chan *proto.MessagePack, 32),
		messageMap:         make(map[uint32]*messageMapEntry),
		stubCallbacks: map[uint32]MessageHandlerFunc{
			// 0 is Reserved
			0: func(_ *Client, _ uint32, _ Message) {},
		},
	}

	c.SetMessageEntry(uint32(proto.MessageType_AUTH), &proto.AuthResultMessage{}, defaultMessageHandler)
	c.SetMessageEntry(uint32(proto.MessageType_CREATE_CHANNEL), &proto.CreateChannelMessage{}, defaultMessageHandler)
	c.SetMessageEntry(uint32(proto.MessageType_REMOVE_CHANNEL), &proto.RemoveChannelMessage{}, handleRemoveChannel)
	c.SetMessageEntry(uint32(proto.MessageType_AUTH), &proto.AuthResultMessage{}, defaultMessageHandler)
	c.SetMessageEntry(uint32(proto.MessageType_SUB_TO_CHANNEL), &proto.SubscribedToChannelMessage{}, handleSubToChannel)
	c.SetMessageEntry(uint32(proto.MessageType_UNSUB_FROM_CHANNEL), &proto.UnsubscribedFromChannelMessage{}, handleUnsubToChannel)
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

func (client *Client) Auth(lt string, pit string) {
	//result := make(chan *proto.AuthResultMessage)
	client.Send(0, proto.BroadcastType_NO, uint32(proto.MessageType_AUTH), &proto.AuthMessage{
		LoginToken:            lt,
		PlayerIdentifierToken: pit,
	}, func(_ *Client, channelId uint32, m Message) {
		msg := m.(*proto.AuthResultMessage)
		client.Id = msg.ConnId
		//result <- msg
		if msg.Result == proto.AuthResultMessage_SUCCESSFUL {
			client.Send(0, proto.BroadcastType_NO, uint32(proto.MessageType_SUB_TO_CHANNEL), &proto.SubscribedToChannelMessage{
				ConnId: client.Id,
			}, nil)
		}
	})
	//return result
}

func handleRemoveChannel(client *Client, channelId uint32, m Message) {
	msg := m.(*proto.RemoveChannelMessage)
	delete(client.subscribedChannels, msg.ChannelId)
}

func handleSubToChannel(client *Client, channelId uint32, m Message) {
	client.subscribedChannels[channelId] = struct{}{}
}

func handleUnsubToChannel(c *Client, channelId uint32, m Message) {
	delete(c.subscribedChannels, channelId)
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

func (client *Client) Receive() error {

	tag, err := readBytes(client.conn, 4)
	if err != nil || tag[0] != 67 {
		return fmt.Errorf("invalid tag: %s, the packet will be dropped: %w", tag, err)
	}

	packetSize := int(tag[3])
	if tag[1] != 72 {
		packetSize = packetSize | int(tag[1])<<16 | int(tag[2])<<8
	} else if tag[2] != 78 {
		packetSize = packetSize | int(tag[2])<<8
	}

	bytes := make([]byte, packetSize)
	if _, err := io.ReadFull(client.conn, bytes); err != nil {
		return fmt.Errorf("error reading packet: %w", err)
	}

	var p proto.Packet
	if err := protobuf.Unmarshal(bytes, &p); err != nil {
		return fmt.Errorf("error unmarshalling packet: %w", err)
	}

	for _, mp := range p.Messages {
		entry := client.messageMap[mp.MsgType]
		if entry == nil {
			return fmt.Errorf("no message type registered: %d", mp.MsgType)
		}

		// Always make a clone!
		msg := protobuf.Clone(entry.msg)
		err = protobuf.Unmarshal(mp.MsgBody, msg)
		if err != nil {
			return fmt.Errorf("failed to unmarshal message: %w", err)
		}

		client.incomingQueue <- messageQueueEntry{msg, mp.ChannelId, mp.StubId, entry.handlers}
	}

	return nil
}

func (client *Client) Tick() error {
	for len(client.incomingQueue) > 0 {
		entry := <-client.incomingQueue

		for _, handler := range entry.handlers {
			handler(client, entry.channelId, entry.msg)
		}

		if entry.stubId > 0 {
			callback := client.stubCallbacks[entry.stubId]
			if callback != nil {
				callback(client, entry.channelId, entry.msg)
			}
		}
	}

	if len(client.outgoingQueue) == 0 {
		return nil
	}

	p := proto.Packet{Messages: make([]*proto.MessagePack, 0, len(client.outgoingQueue))}
	size := 0
	for len(client.outgoingQueue) > 0 {
		mp := <-client.outgoingQueue
		if size+protobuf.Size(mp) >= 0xfffff0 {
			break
		}
		p.Messages = append(p.Messages, mp)
	}
	return client.writePacket(&p)
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
		return fmt.Errorf("failed to marshal message %d: %s. Error: %w", msgType, msg, err)
	}

	client.outgoingQueue <- &proto.MessagePack{
		ChannelId: channelId,
		Broadcast: broadcast,
		StubId:    stubId,
		MsgType:   msgType,
		MsgBody:   msgBody,
	}
	return nil
}

func (client *Client) writePacket(p *proto.Packet) error {
	bytes, err := protobuf.Marshal(p)
	if err != nil {
		return fmt.Errorf("error marshalling packet: %w", err)
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

	client.writeMutex.Lock()
	defer client.writeMutex.Unlock()
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
	//c.SetReadDeadline(time.Now().Add(30 * time.Second))
	if c.readBuf == nil || c.readIdx >= len(c.readBuf) {
		defer func() {
			if recover() != nil {
				err = errors.New("read on failed connection")
			}
		}()
		_, c.readBuf, err = c.conn.ReadMessage()
		if err != nil {
			return 0, err
		}
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
