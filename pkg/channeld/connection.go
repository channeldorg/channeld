package channeld

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"channeld.clewcat.com/channeld/pkg/fsm"
	"channeld.clewcat.com/channeld/proto"
	"github.com/gorilla/websocket"
	"github.com/xtaci/kcp-go"
	"go.uber.org/zap"
	protobuf "google.golang.org/protobuf/proto"
)

type ConnectionId uint32

type ConnectionType uint8

const (
	SERVER ConnectionType = 1
	CLIENT ConnectionType = 2
)

func (t ConnectionType) String() string {
	if t == SERVER {
		return "SERVER"
	} else if t == CLIENT {
		return "CLIENT"
	} else {
		return "UNKNOW"
	}
}

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

var allConnections sync.Map // map[ConnectionId]*Connection
var nextConnectionId uint64 = 0
var serverFsm fsm.FiniteStateMachine
var clientFsm fsm.FiniteStateMachine

func InitConnections(serverFsmPath string, clientFsmPath string) {
	bytes, err := os.ReadFile(serverFsmPath)
	if err == nil {
		serverFsm, err = fsm.Load(bytes)
	}
	if err != nil {
		logger.Panic("failed to read server FSM",
			zap.Error(err),
		)
	} else {
		logger.Info("loaded server FSM",
			zap.String("currentState", serverFsm.CurrentState().Name),
		)
	}

	bytes, err = os.ReadFile(clientFsmPath)
	if err == nil {
		clientFsm, err = fsm.Load(bytes)
	}
	if err != nil {
		logger.Panic("failed to read client FSM", zap.Error(err))
	} else {
		logger.Info("loaded client FSM",
			zap.String("currentState", clientFsm.CurrentState().Name),
		)
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
	v, ok := allConnections.Load(id)
	if ok {
		c := v.(*Connection)
		if c.IsRemoving() {
			return nil
		}
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
	logger.Info("start listenning",
		zap.String("connType", t.String()),
		zap.String("network", network),
		zap.String("address", address),
	)

	var listener net.Listener
	var err error
	switch network {
	case "ws", "websocket":
		startWebSocketServer(t, address)
		return
	case "kcp":
		listener, err = kcp.Listen(address)
	default:
		listener, err = net.Listen(network, address)
	}

	if err != nil {
		logger.Panic("failed to listen", zap.Error(err))
		return
	}

	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Error("failed to accept connection", zap.Error(err))
		} else {
			connection := AddConnection(conn, t)
			connection.Logger().Debug("accepted connection")
			startGoroutines(connection)
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

	allConnections.Store(connection.id, connection)

	connectionNum.WithLabelValues(t.String()).Inc()

	return connection
}

func RemoveConnection(c *Connection) {
	atomic.AddInt32(&c.removing, 1)
	c.conn.Close()
	close(c.sendQueue)
	allConnections.Delete(c.id)

	connectionNum.WithLabelValues(c.connectionType.String()).Dec()
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
		switch err := err.(type) {
		case *net.OpError:
			c.Logger().Warn("read bytes",
				zap.String("op", err.Op),
				zap.String("remoteAddr", c.conn.RemoteAddr().String()),
				zap.Error(err),
			)
			RemoveConnection(c)
			return nil, &closeError{err}
		case *websocket.CloseError:
			c.Logger().Info("disconnected",
				zap.String("remoteAddr", c.conn.RemoteAddr().String()),
			)
			RemoveConnection(c)
			return nil, &closeError{err}
		}

		if err == io.EOF {
			c.Logger().Info("disconnected",
				zap.String("remoteAddr", c.conn.RemoteAddr().String()),
			)
			RemoveConnection(c)
			return nil, &closeError{err}
		}
		return nil, err
	}
	return bytes, nil
}

func _(c *Connection) (uint32, error) {
	bytes, err := readBytes(c, 4)
	if err != nil {
		return 0, err
	} else {
		return binary.BigEndian.Uint32(bytes), nil
	}
}

func (c *Connection) ReceivePacket() {
	// FIXME: read all bytes once into a buffer
	tag, err := readBytes(c, 4)
	if err != nil {
		return
	}
	if tag[0] != 67 {
		c.Logger().Warn("invalid tag, the packet will be dropped",
			zap.ByteString("tag", tag),
			zap.Error(err),
		)
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
		c.Logger().Error("reading packet", zap.Error(err))
		return
	}

	bytesReceived.Add(float64(packetSize + 4))

	var p proto.Packet
	if err := protobuf.Unmarshal(bytes, &p); err != nil {
		c.Logger().Error("unmarshalling packet", zap.Error(err))
		return
	}

	for _, mp := range p.Messages {
		c.receiveMessage(mp)
	}

	//c.Logger().Debug("received packet", zap.Int("size", packetSize))
	packetReceived.Inc()
}

func (c *Connection) receiveMessage(mp *proto.MessagePack) {
	var channel *Channel
	if mp.Broadcast == proto.BroadcastType_SINGLE_CONNECTION {
		// Single connection forwarding will be handled in the GLOBAL channel, as the channelId will be used as the target connId.
		channel = globalChannel
	} else {
		channel = GetChannel(ChannelId(mp.ChannelId))
		if channel == nil {
			c.Logger().Warn("can't find channel",
				zap.Uint32("channelId", mp.ChannelId),
				zap.Uint32("msgType", mp.MsgType),
			)
			return
		}
	}

	entry := MessageMap[proto.MessageType(mp.MsgType)]
	if entry == nil && mp.MsgType < uint32(proto.MessageType_USER_SPACE_START) {
		c.Logger().Error("undefined message type", zap.Uint32("msgType", mp.MsgType))
		return
	}

	if !c.fsm.IsAllowed(mp.MsgType) {
		c.Logger().Warn("message is not allowed for current state", zap.Uint32("msgType", mp.MsgType))
		return
	}

	var msg Message
	var handler MessageHandlerFunc
	if mp.MsgType >= uint32(proto.MessageType_USER_SPACE_START) && entry == nil {
		// User-space message without handler won't be deserialized.
		msg = &proto.UserSpaceMessage{SourceConnId: uint32(c.id), MsgBody: mp.MsgBody}
		handler = handleUserSpaceMessage
	} else {
		handler = entry.handler
		// Always make a clone!
		msg = protobuf.Clone(entry.msg)
		err := protobuf.Unmarshal(mp.MsgBody, msg)
		if err != nil {
			c.Logger().Error("unmarshalling message", zap.Error(err))
			return
		}
	}

	c.fsm.OnReceived(mp.MsgType)

	channel.PutMessage(msg, handler, c, mp)

	c.Logger().Debug("received message", zap.Uint32("msgType", mp.MsgType), zap.Int("size", len(mp.MsgBody)))

	msgReceived.Inc() /*.WithLabelValues(
		strconv.FormatUint(uint64(p.ChannelId), 10),
		strconv.FormatUint(uint64(p.MsgType), 10),
	)*/
}

func (c *Connection) Send(ctx MessageContext) {
	if c.IsRemoving() {
		return
	}

	c.sender.Send(c, ctx)
}

func (c *Connection) Flush() {
	if len(c.sendQueue) == 0 {
		return
	}

	p := proto.Packet{Messages: make([]*proto.MessagePack, 0, len(c.sendQueue))}
	size := 0

	// TODO: should we limit the message numbers per packet?
	for len(c.sendQueue) > 0 {
		mc := <-c.sendQueue
		// The packet size should not exceed the capacity of 3 bytes
		if size+protobuf.Size(mc.Msg) >= 0xfffff0 {
			c.Logger().Warn("packet is going to be oversized")
			break
		}
		msgBody, err := protobuf.Marshal(mc.Msg)
		if err != nil {
			c.Logger().Error("error marshalling message", zap.Error(err))
			continue
		}
		p.Messages = append(p.Messages, &proto.MessagePack{
			ChannelId: mc.ChannelId,
			Broadcast: mc.Broadcast,
			StubId:    mc.StubId,
			MsgType:   uint32(mc.MsgType),
			MsgBody:   msgBody,
		})
		size = protobuf.Size(&p)

		msgSent.Inc() /*.WithLabelValues(
			strconv.FormatUint(uint64(e.Channel.id), 10),
			strconv.FormatUint(uint64(e.MsgType), 10),
		)*/
	}

	bytes, err := protobuf.Marshal(&p)
	if err != nil {
		c.Logger().Error("error marshalling packet", zap.Error(err))
		return
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

	/* Avoid writing multple times. With WebSocket, every Write() sends a message.
	writer.Write(tag)
	*/
	_, err = c.writer.Write(append(tag, bytes...))
	if err != nil {
		c.Logger().Error("error writing packet", zap.Error(err))
		return
	}

	c.writer.Flush()

	packetSent.Inc()
	bytesSent.Add(float64(len))
}

func (c *Connection) Disconnect() error {
	return c.conn.Close()
}

func (c *Connection) String() string {
	return fmt.Sprintf("Connection(%s %d %s)", c.connectionType, c.id, c.fsm.CurrentState().Name)
}

// FIXME: every call clones the logger!
func (c *Connection) Logger() *zap.Logger {
	return logger.With(
		zap.String("connType", c.connectionType.String()),
		zap.Uint32("connId", uint32(c.id)),
		zap.String("connState", c.fsm.CurrentState().Name),
	)
}
