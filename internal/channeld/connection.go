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

	"clewcat.com/channeld/internal/fsm"
	"clewcat.com/channeld/proto"
	protobuf "google.golang.org/protobuf/proto"
)

type ConnectionId uint32

type ConnectionType uint8

const (
	SERVER        ConnectionType = 1
	CLIENT        ConnectionType = 2
	FlushInterval time.Duration  = time.Millisecond * 10
)

type sendQueueMessage struct {
	channelId ChannelId
	msgType   proto.MessageType
	msg       Message
}

// Add an interface before the underlying network layer for the test code.
type MessageSender interface {
	Send(c *Connection, channelId ChannelId, msgType proto.MessageType, msg Message)
}

type QueuedMessageSender struct{}

func (s *QueuedMessageSender) Send(c *Connection, channelId ChannelId, msgType proto.MessageType, msg Message) {
	c.sendQueue <- sendQueueMessage{
		channelId: channelId,
		msgType:   msgType,
		msg:       msg,
	}
}

type Connection struct {
	id             ConnectionId
	connectionType ConnectionType
	conn           net.Conn
	reader         *bufio.Reader
	writer         *bufio.Writer
	sender         MessageSender
	sendQueue      chan sendQueueMessage
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

func StartListening(t ConnectionType, network string, address string) {
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

			go func() {
				for !connection.IsRemoving() {
					connection.Receive()
				}
			}()

			go func() {
				for !connection.IsRemoving() {
					connection.Flush()
					time.Sleep(time.Millisecond)
				}
			}()
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
		sender:         &QueuedMessageSender{},
		sendQueue:      make(chan sendQueueMessage, 128),
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

func readBytes(c *Connection, len uint) ([]byte, error) {
	bytes := make([]byte, len)
	if _, err := io.ReadFull(c.reader, bytes); err != nil {
		switch err.(type) {
		case *net.OpError:
			log.Printf("%s disconnected: %s\n", c.String(), c.conn.RemoteAddr().String())
			RemoveConnection(c)
		}

		if err == io.EOF {
			log.Printf("%s disconnected: %s\n", c.String(), c.conn.RemoteAddr().String())
			RemoveConnection(c)
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
		return binary.LittleEndian.Uint32(bytes), nil
	}
}

// CHNL in ASCII
const TAG_ID uint32 = 67<<24 | 72<<16 | 78<<8 | 76

// TODO: read the packet using protobuf
func (c *Connection) Receive() {
	if tag, err := readUint32(c); tag != TAG_ID {
		log.Println("Invalid tag:", tag, ", the packet will be dropped.", err)
		// Drop the packet
		ioutil.ReadAll(c.reader)
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

	stubId, err := readUint32(c)
	if err != nil {
		log.Println("Error when reading StubId: ", err)
		return
	}
	if stubId > 0 {
		RPC().SaveStub(stubId, c)
	}

	// TODO: forward and broadcast user-space messages

	msgType, err := readUint32(c)
	if err != nil {
		log.Println("Error when reading message type: ", err)
		return
	}
	entry := MessageMap[proto.MessageType(msgType)]
	if entry == nil {
		log.Println("Undeinfed message type: ", msgType)
	}

	if !c.fsm.IsAllowed(msgType) {
		log.Printf("Message Type %d is not allow in state %s\n", msgType, c.fsm.CurrentState().Name)
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
	// Always make a clone!
	msg := protobuf.Clone(entry.msg)
	err = protobuf.Unmarshal(body, msg)
	if err != nil {
		log.Panicln(err)
	}

	c.fsm.OnReceived(msgType)

	channel.PutMessage(msg, entry.handler, c)
}

func (c *Connection) Send(channelId ChannelId, msgType proto.MessageType, msg Message) {
	if c.IsRemoving() {
		return
	}

	c.sender.Send(c, channelId, msgType, msg)
}

func (c *Connection) SendWithGlobalChannel(msgType proto.MessageType, msg Message) {
	if c.IsRemoving() {
		return
	}

	c.Send(0, msgType, msg)
}

func (c *Connection) Flush() {
	for len(c.sendQueue) > 0 {
		e := <-c.sendQueue
		bytes, err := protobuf.Marshal(e.msg)
		if err != nil {
			log.Printf("Failed to marshal message %d: %s\n", e.msgType, e.msg)
			continue
		}
		if len(bytes) >= (1 << 32) {
			log.Printf("Message body is too large, size: %d\n", len(bytes))
			continue
		}

		binary.Write(c.writer, binary.LittleEndian, TAG_ID)
		binary.Write(c.writer, binary.LittleEndian, uint32(e.channelId))
		binary.Write(c.writer, binary.LittleEndian, uint32(0))
		binary.Write(c.writer, binary.LittleEndian, uint32(e.msgType))
		binary.Write(c.writer, binary.LittleEndian, uint32(len(bytes)))
		binary.Write(c.writer, binary.LittleEndian, bytes)
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
