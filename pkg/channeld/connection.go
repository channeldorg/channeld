package channeld

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sync/atomic"
	"time"

	"channeld.clewcat.com/channeld/pkg/fsm"
	"channeld.clewcat.com/channeld/proto"
	"github.com/gorilla/websocket"
	protobuf "google.golang.org/protobuf/proto"
)

type ConnectionId uint32

type ConnectionType uint8

const (
	SERVER ConnectionType = 1
	CLIENT ConnectionType = 2
)

// Add an interface before the underlying network layer for the test purpose.
type MessageSender interface {
	Send(c *Connection, ctx MessageContext) //(c *Connection, channelId ChannelId, msgType proto.MessageType, msg Message)
}

type queuedMessageSender struct {
	MessageSender
}

/*
func (s *queuedMessageSender) Send(c *Connection, channelId ChannelId, msgType proto.MessageType, msg Message) {
	c.sendQueue <- sendQueueMessage{
		channelId: channelId,
		msgType:   msgType,
		msg:       msg,
	}
}
*/
func (s *queuedMessageSender) Send(c *Connection, ctx MessageContext) {
	c.sendQueue <- ctx
}

type Connection struct {
	id             ConnectionId
	connectionType ConnectionType
	conn           net.Conn
	reader         *bufio.Reader
	writer         *bufio.Writer
	sender         MessageSender
	sendQueue      chan MessageContext
	fsm            *fsm.FiniteStateMachine
	removing       int32 // Don't put the removing into the FSM as 1) the FSM's states are user-defined. 2) the FSM doesn't have the race condition.
}

var allConnections map[ConnectionId]*Connection
var nextConnectionId uint64 = 0
var serverFsm fsm.FiniteStateMachine
var clientFsm fsm.FiniteStateMachine

func InitConnections(connSize int, serverFsmPath string, clientFsmPath string) {
	allConnections = make(map[ConnectionId]*Connection, connSize)

	bytes, err := os.ReadFile(serverFsmPath)
	if err == nil {
		serverFsm, err = fsm.Load(bytes)
	}
	if err != nil {
		log.Println("Failed to read server FSM", err)
	}

	bytes, err = os.ReadFile(clientFsmPath)
	if err == nil {
		clientFsm, err = fsm.Load(bytes)
	}
	if err != nil {
		log.Println("Failed to read client FSM", err)
	}

	/* Split each Connection.Flush into a goroutine (see AddConnection)
	go func() {
		for {
			t := time.Now()
			for _, c := range allConnections {
				if !<-c.removing {
					c.Flush()
				}
			}
			time.Sleep(FlushInterval - time.Since(t))
		}
	}()
	*/
}

func GetConnection(id ConnectionId) *Connection {
	c := allConnections[id]
	if c != nil && !c.IsRemoving() {
		return c
	} else {
		return nil
	}
}

func startGoroutines(connection *Connection) {
	go func() {
		for !connection.IsRemoving() {
			connection.ReceivePacket()
		}
	}()

	go func() {
		for !connection.IsRemoving() {
			connection.Flush()
			time.Sleep(time.Millisecond)
		}
	}()
}

func StartListening(t ConnectionType, network string, address string) {
	// TODO: add "kcp" network types
	if network == "ws" || network == "websocket" {
		startWebSocketServer(t, address)
	} else {

		listener, err := net.Listen(network, address)
		if err != nil {
			log.Fatal(err)
			return
		}

		defer listener.Close()

		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Println(err)
			} else {
				connection := AddConnection(conn, t)
				startGoroutines(connection)
			}
		}
	}
}

func AddConnection(c net.Conn, t ConnectionType) *Connection {
	atomic.AddUint64(&nextConnectionId, 1)
	connection := &Connection{
		id:             ConnectionId(nextConnectionId),
		connectionType: t,
		conn:           c,
		reader:         bufio.NewReader(c),
		writer:         bufio.NewWriter(c),
		sender:         &queuedMessageSender{},
		sendQueue:      make(chan MessageContext, 128),
		removing:       0,
	}
	var fsm fsm.FiniteStateMachine
	switch t {
	case SERVER:
		fsm = serverFsm
		fallthrough
	case CLIENT:
		fsm = clientFsm
		connection.fsm = &fsm
	}

	allConnections[connection.id] = connection

	return connection
}

func RemoveConnection(c *Connection) {
	atomic.AddInt32(&c.removing, 1)
	c.conn.Close()
	delete(allConnections, c.id)
}

func (c *Connection) IsRemoving() bool {
	return c.removing > 0
}

type closeError struct {
	source error
}

func (e *closeError) Error() string {
	return e.source.Error()
}

func readBytes(c *Connection, len uint) ([]byte, error) {
	bytes := make([]byte, len)
	if _, err := io.ReadFull(c.reader, bytes); err != nil {
		switch err.(type) {
		case *net.OpError:
		case *websocket.CloseError:
			log.Printf("%s disconnected: %s\n", c.String(), c.conn.RemoteAddr().String())
			RemoveConnection(c)
			return nil, &closeError{err}
		}

		if err == io.EOF {
			log.Printf("%s disconnected: %s\n", c.String(), c.conn.RemoteAddr().String())
			RemoveConnection(c)
			return nil, &closeError{err}
		}
		return nil, err
	}
	return bytes, nil
}

func readUint32(c *Connection) (uint32, error) {
	bytes, err := readBytes(c, 4)
	if err != nil {
		return 0, err
	} else {
		return binary.BigEndian.Uint32(bytes), nil
	}
}

// 'CHNL' in ASCII
const TAG_ID uint32 = 67<<24 | 72<<16 | 78<<8 | 76

func (c *Connection) ReceivePacket() {
	tag, err := readBytes(c, 4)
	if err != nil || tag[0] != 67 {
		log.Println("Invalid tag:", tag, ", the packet will be dropped.", err)
		_, isClosed := err.(*closeError)
		if !isClosed {
			// Drop the packet
			ioutil.ReadAll(c.reader)
		}
		return
	}

	packetSize := int(tag[3])
	if tag[1] != 72 {
		packetSize = packetSize | int(tag[1])<<16 | int(tag[2])<<8
	} else if tag[2] != 78 {
		packetSize = packetSize | int(tag[2])<<8
	}

	bytes := make([]byte, packetSize)
	if _, err := io.ReadFull(c.reader, bytes); err != nil {
		log.Panic("Error reading packet: ", err)
	}

	var p proto.Packet
	if err := protobuf.Unmarshal(bytes, &p); err != nil {
		log.Panic("Error unmarshalling packet: ", err)
	}

	channel := GetChannel(ChannelId(p.ChannelId))
	if channel == nil {
		log.Println("Can't find channel by ID: ", p.ChannelId)
		return
	}

	if p.StubId > 0 {
		RPC().SaveStub(p.StubId, c)
	}

	entry := MessageMap[proto.MessageType(p.MsgType)]
	if entry == nil && p.MsgType < uint32(proto.MessageType_USER_SPACE_START) {
		log.Printf("%s sent undefined message type: %d\n", c, p.MsgType)
		return
	}

	if !c.fsm.IsAllowed(p.MsgType) {
		log.Printf("%s doesn't allow message type %d for current state.\n", c, p.MsgType)
		return
	}

	var msg Message
	var handler MessageHandlerFunc
	if p.MsgType >= uint32(proto.MessageType_USER_SPACE_START) && entry == nil {
		// User-space message without handler won't be deserialized.
		msg = &proto.UserSpaceMessage{MsgBody: p.MsgBody}
		handler = handleUserSpaceMessage
	} else {
		handler = entry.handler
		// Always make a clone!
		msg = protobuf.Clone(entry.msg)
		err = protobuf.Unmarshal(p.MsgBody, msg)
		if err != nil {
			log.Panicln(err)
		}
	}

	c.fsm.OnReceived(p.MsgType)

	channel.PutMessage(msg, handler, c, &p)
}

func (c *Connection) ReceiveRaw() {
	if tag, err := readUint32(c); tag != TAG_ID {
		log.Println("Invalid tag:", tag, ", the packet will be dropped.", err)
		_, isClosed := err.(*closeError)
		if !isClosed {
			// Drop the packet
			ioutil.ReadAll(c.reader)
		}
		return
	}

	channelId, err := readUint32(c)
	if err != nil {
		log.Println("Error when reading ChannelId: ", err)
		return
	}
	channel := GetChannel(ChannelId(channelId))
	if channel == nil {
		log.Println("Can't find channel by ID: ", channelId)
		return
	}

	broadcast, err := c.reader.ReadByte()
	if err != nil {
		log.Println("Error when reading Broadcast: ", err)
		return
	}
	stubId, err := readUint32(c)
	if err != nil {
		log.Println("Error when reading StubId: ", err)
		return
	}
	if stubId > 0 {
		RPC().SaveStub(stubId, c)
	}

	msgType, err := readUint32(c)
	if err != nil {
		log.Println("Error when reading message type: ", err)
		return
	}
	entry := MessageMap[proto.MessageType(msgType)]
	if entry == nil && msgType < uint32(proto.MessageType_USER_SPACE_START) {
		log.Printf("%s sent undefined message type: %d\n", c, msgType)
		return
	}

	if !c.fsm.IsAllowed(msgType) {
		log.Printf("%s doesn't allow message type %d for current state.\n", c, msgType)
		return
	}

	bodySize, err := readUint32(c)
	if err != nil {
		log.Println("Error when reading body size: ", err)
		return
	}

	body, err := readBytes(c, uint(bodySize))
	if err != nil {
		log.Println("Error when reading message body: ", err)
		return
	}

	var msg Message
	var handler MessageHandlerFunc
	if msgType >= uint32(proto.MessageType_USER_SPACE_START) && entry == nil {
		// User-space message without handler won't be deserialized.
		msg = &proto.UserSpaceMessage{MsgBody: body}
		handler = handleUserSpaceMessage
	} else {
		handler = entry.handler
		// Always make a clone!
		msg = protobuf.Clone(entry.msg)
		err = protobuf.Unmarshal(body, msg)
		if err != nil {
			log.Panicln(err)
		}
	}

	c.fsm.OnReceived(msgType)

	channel.PutMessage(msg, handler, c, &proto.Packet{Broadcast: proto.BroadcastType(broadcast), StubId: stubId})
}

func (c *Connection) ForwardPacket(channelId ChannelId, msgType proto.MessageType, bodySize uint32, msgBody []byte) {

}

func (c *Connection) Send(ctx MessageContext) {
	if c.IsRemoving() {
		return
	}

	c.sender.Send(c, ctx)
}

func (c *Connection) Flush() {
	for len(c.sendQueue) > 0 {
		e := <-c.sendQueue
		msgBody, err := protobuf.Marshal(e.Msg)
		if err != nil {
			log.Printf("Failed to marshal message %d: %s\n", e.MsgType, e.Msg)
			continue
		}
		if len(msgBody) >= (1 << 24) {
			log.Printf("Message body is too large, size: %d\n", len(msgBody))
			continue
		}

		bytes, err := protobuf.Marshal(&proto.Packet{
			ChannelId: uint32(e.Channel.id),
			Broadcast: proto.BroadcastType_NO,
			StubId:    e.StubId,
			MsgType:   uint32(e.MsgType),
			MsgBody:   msgBody,
		})
		if err != nil {
			log.Println("Error marshalling packet: ", err)
			continue
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
		c.writer.Write(tag)
		c.writer.Write(bytes)
	}

	c.writer.Flush()
}

func (c *Connection) String() string {
	var typeName string
	if c.connectionType == SERVER {
		typeName = "SERVER"
	} else if c.connectionType == CLIENT {
		typeName = "CLIENT"
	}
	return fmt.Sprintf("Connection(%s %d %s)", typeName, c.id, c.fsm.CurrentState().Name)
}
