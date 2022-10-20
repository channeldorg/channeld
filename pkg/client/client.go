package client

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"channeld.clewcat.com/channeld/pkg/channeld"
	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"github.com/golang/snappy"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

type Message = proto.Message
type MessageHandlerFunc func(client *ChanneldClient, channelId uint32, m Message)
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

// Go library for writing game client/server that interations with channeld.
type ChanneldClient struct {
	Id                 uint32
	CompressionType    channeldpb.CompressionType
	SubscribedChannels map[uint32]struct{}
	CreatedChannels    map[uint32]struct{}
	ListedChannels     map[uint32]struct{}
	Conn               net.Conn
	readBuffer         []byte
	readPos            int
	connected          bool
	incomingQueue      chan messageQueueEntry
	outgoingQueue      chan *channeldpb.MessagePack
	messageMap         map[uint32]*messageMapEntry
	stubCallbacks      map[uint32]MessageHandlerFunc
	writeMutex         sync.Mutex
}

func NewClient(addr string) (*ChanneldClient, error) {
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
	c := &ChanneldClient{
		CompressionType:    channeldpb.CompressionType_NO_COMPRESSION,
		SubscribedChannels: make(map[uint32]struct{}),
		CreatedChannels:    make(map[uint32]struct{}),
		ListedChannels:     make(map[uint32]struct{}),
		Conn:               conn,
		readBuffer:         make([]byte, channeld.MaxPacketSize),
		readPos:            0,
		connected:          true,
		incomingQueue:      make(chan messageQueueEntry, 128),
		outgoingQueue:      make(chan *channeldpb.MessagePack, 32),
		messageMap:         make(map[uint32]*messageMapEntry),
		stubCallbacks: map[uint32]MessageHandlerFunc{
			// 0 is Reserved
			0: func(_ *ChanneldClient, _ uint32, _ Message) {},
		},
	}

	c.SetMessageEntry(uint32(channeldpb.MessageType_AUTH), &channeldpb.AuthResultMessage{}, handleAuth)
	c.SetMessageEntry(uint32(channeldpb.MessageType_CREATE_CHANNEL), &channeldpb.CreateChannelResultMessage{}, handleCreateChannel)
	c.SetMessageEntry(uint32(channeldpb.MessageType_REMOVE_CHANNEL), &channeldpb.RemoveChannelMessage{}, handleRemoveChannel)
	c.SetMessageEntry(uint32(channeldpb.MessageType_SUB_TO_CHANNEL), &channeldpb.SubscribedToChannelResultMessage{}, handleSubToChannel)
	c.SetMessageEntry(uint32(channeldpb.MessageType_UNSUB_FROM_CHANNEL), &channeldpb.UnsubscribedFromChannelResultMessage{}, handleUnsubToChannel)
	c.SetMessageEntry(uint32(channeldpb.MessageType_LIST_CHANNEL), &channeldpb.ListChannelResultMessage{}, handleListChannel)
	c.SetMessageEntry(uint32(channeldpb.MessageType_CHANNEL_DATA_UPDATE), &channeldpb.ChannelDataUpdateMessage{}, defaultMessageHandler)

	return c, nil
}

func (client *ChanneldClient) Disconnect() error {
	return client.Conn.Close()
}

func (client *ChanneldClient) SetMessageEntry(msgType uint32, msgTemplate Message, handlers ...MessageHandlerFunc) {
	client.messageMap[msgType] = &messageMapEntry{
		msg:      msgTemplate,
		handlers: handlers,
	}
}

func (client *ChanneldClient) AddMessageHandler(msgType uint32, handlers ...MessageHandlerFunc) error {
	entry := client.messageMap[msgType]
	if entry != nil {
		entry.handlers = append(entry.handlers, handlers...)
		return nil
	} else {
		return fmt.Errorf("failed to add handler as the message entry not found, msgType: %d", msgType)
	}
}

func (client *ChanneldClient) Auth(lt string, pit string) {
	//result := make(chan *channeldpb.AuthResultMessage)
	client.Send(0, channeldpb.BroadcastType_NO_BROADCAST, uint32(channeldpb.MessageType_AUTH), &channeldpb.AuthMessage{
		LoginToken:            lt,
		PlayerIdentifierToken: pit,
	}, nil)
	//return result
}

func handleAuth(client *ChanneldClient, channelId uint32, m Message) {
	msg := m.(*channeldpb.AuthResultMessage)

	if msg.Result == channeldpb.AuthResultMessage_SUCCESSFUL {
		if client.Id == 0 {
			client.Id = msg.ConnId
			client.CompressionType = msg.CompressionType
		}

		// client.Send(0, channeldpb.BroadcastType_NO_BROADCAST, uint32(channeldpb.MessageType_SUB_TO_CHANNEL), &channeldpb.SubscribedToChannelMessage{
		// 	ConnId: client.Id,
		// }, nil)
	}
}

func handleCreateChannel(c *ChanneldClient, channelId uint32, m Message) {
	c.CreatedChannels[channelId] = struct{}{}
}

func handleRemoveChannel(client *ChanneldClient, channelId uint32, m Message) {
	msg := m.(*channeldpb.RemoveChannelMessage)
	delete(client.SubscribedChannels, msg.ChannelId)
	delete(client.CreatedChannels, msg.ChannelId)
	delete(client.ListedChannels, msg.ChannelId)
}

func handleSubToChannel(client *ChanneldClient, channelId uint32, m Message) {
	client.SubscribedChannels[channelId] = struct{}{}
}

func handleUnsubToChannel(c *ChanneldClient, channelId uint32, m Message) {
	delete(c.SubscribedChannels, channelId)
}

func handleListChannel(c *ChanneldClient, channelId uint32, m Message) {
	c.ListedChannels = map[uint32]struct{}{}
	for _, info := range m.(*channeldpb.ListChannelResultMessage).Channels {
		c.ListedChannels[info.ChannelId] = struct{}{}
	}
}

func defaultMessageHandler(client *ChanneldClient, channelId uint32, m Message) {
	//log.Printf("Client(%d) received message from channel %d: %s", client.Id, channelId, m)
}

func (client *ChanneldClient) IsConnected() bool {
	return client.connected
}

func (client *ChanneldClient) Receive() error {
	readPtr := client.readBuffer[client.readPos:]
	bytesRead, err := client.Conn.Read(readPtr)
	if err != nil {
		return err
	}

	client.readPos += bytesRead
	if client.readPos < 5 {
		// Unfinished header
		return nil
	}

	tag := client.readBuffer[:5]
	if tag[0] != 67 {
		return fmt.Errorf("invalid tag: %s, the packet will be dropped: %w", tag, err)
	}

	packetSize := int(tag[3])
	if tag[1] != 72 {
		packetSize = packetSize | int(tag[1])<<16 | int(tag[2])<<8
	} else if tag[2] != 78 {
		packetSize = packetSize | int(tag[2])<<8
	}

	fullSize := 5 + packetSize
	if client.readPos < fullSize {
		// Unfinished packet
		return nil
	}

	bytes := client.readBuffer[5:fullSize]

	// Apply the decompression from the 5th byte in the header
	// Apply the decompression from the 5th byte in the header
	if tag[4] == byte(channeldpb.CompressionType_SNAPPY) {
		len, err := snappy.DecodedLen(bytes)
		if err != nil {
			return fmt.Errorf("snappy.DecodedLen: %w", err)
		}
		dst := make([]byte, len)
		bytes, err = snappy.Decode(dst, bytes)
		if err != nil {
			return fmt.Errorf("snappy.Decode: %w", err)
		}
	}

	var p channeldpb.Packet
	if err := proto.Unmarshal(bytes, &p); err != nil {
		return fmt.Errorf("error unmarshalling packet: %w", err)
	}

	for _, mp := range p.Messages {
		entry := client.messageMap[mp.MsgType]
		if entry == nil {
			return fmt.Errorf("no message type registered: %d", mp.MsgType)
		}

		// Always make a clone!
		msg := proto.Clone(entry.msg)
		err = proto.Unmarshal(mp.MsgBody, msg)
		if err != nil {
			return fmt.Errorf("failed to unmarshal message: %w", err)
		}

		client.incomingQueue <- messageQueueEntry{msg, mp.ChannelId, mp.StubId, entry.handlers}
	}

	client.readPos = 0

	return nil
}

func (client *ChanneldClient) Tick() error {
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

	p := channeldpb.Packet{Messages: make([]*channeldpb.MessagePack, 0, len(client.outgoingQueue))}
	size := 0
	for len(client.outgoingQueue) > 0 {
		mp := <-client.outgoingQueue
		if size+proto.Size(mp) >= 0xfffff0 {
			break
		}
		p.Messages = append(p.Messages, mp)
	}
	return client.writePacket(&p)
}

func (client *ChanneldClient) Send(channelId uint32, broadcast channeldpb.BroadcastType, msgType uint32, msg Message, callback MessageHandlerFunc) error {
	var stubId uint32 = 0
	if callback != nil {
		for client.stubCallbacks[stubId] != nil {
			stubId++
		}
		client.stubCallbacks[stubId] = callback
	}

	msgBody, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message %d: %s. Error: %w", msgType, msg, err)
	}

	client.outgoingQueue <- &channeldpb.MessagePack{
		ChannelId: channelId,
		Broadcast: uint32(broadcast),
		StubId:    stubId,
		MsgType:   msgType,
		MsgBody:   msgBody,
	}
	return nil
}

func (client *ChanneldClient) SendRaw(channelId uint32, broadcast channeldpb.BroadcastType, msgType uint32, msgBody *[]byte, callback MessageHandlerFunc) error {
	var stubId uint32 = 0
	if callback != nil {
		for client.stubCallbacks[stubId] != nil {
			stubId++
		}
		client.stubCallbacks[stubId] = callback
	}

	client.outgoingQueue <- &channeldpb.MessagePack{
		ChannelId: channelId,
		Broadcast: uint32(broadcast),
		StubId:    stubId,
		MsgType:   msgType,
		MsgBody:   *msgBody,
	}
	return nil
}

func (client *ChanneldClient) writePacket(p *channeldpb.Packet) error {
	bytes, err := proto.Marshal(p)
	if err != nil {
		return fmt.Errorf("error marshalling packet: %w", err)
	}

	// Apply the compression
	if client.CompressionType == channeldpb.CompressionType_SNAPPY {
		dst := make([]byte, snappy.MaxEncodedLen(len(bytes)))
		bytes = snappy.Encode(dst, bytes)
	}

	// 'CHNL' in ASCII
	tag := []byte{67, 72, 78, 76, byte(client.CompressionType)}
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
	client.Conn.Write(append(tag, bytes...))
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
