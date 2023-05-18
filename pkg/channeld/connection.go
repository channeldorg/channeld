package channeld

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/golang/snappy"
	"github.com/gorilla/websocket"
	"github.com/metaworking/channeld/pkg/channeldpb"
	"github.com/metaworking/channeld/pkg/common"
	"github.com/metaworking/channeld/pkg/fsm"
	"github.com/metaworking/channeld/pkg/replaypb"
	"github.com/puzpuzpuz/xsync/v2"
	"github.com/xtaci/kcp-go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type ConnectionId uint32

const MaxPacketSize int = 0x00ffff
const PacketHeaderSize int = 5

//type ConnectionState int32

const (
	ConnectionState_UNAUTHENTICATED int32 = 0
	ConnectionState_AUTHENTICATED   int32 = 1
	ConnectionState_CLOSING         int32 = 2
)

// Add an interface before the underlying network layer for the test purpose.
type MessageSender interface {
	Send(c *Connection, ctx MessageContext) //(c *Connection, channelId ChannelId, msgType channeldpb.MessageType, msg Message)
}

/*
type queuedMessageCtxSender struct {
	MessageSender
}

func (s *queuedMessageCtxSender) Send(c *Connection, ctx MessageContext) {
	c.sendQueue <- ctx
}
*/

type queuedMessagePackSender struct {
	MessageSender
}

func (s *queuedMessagePackSender) Send(c *Connection, ctx MessageContext) {
	msgBody, err := proto.Marshal(ctx.Msg)
	if err != nil {
		c.logger.Error("failed to marshal message", zap.Error(err), zap.Uint32("msgType", uint32(ctx.MsgType)))
		return
	}

	c.sendQueue <- &channeldpb.MessagePack{
		ChannelId: ctx.ChannelId,
		Broadcast: ctx.Broadcast,
		StubId:    ctx.StubId,
		MsgType:   uint32(ctx.MsgType),
		MsgBody:   msgBody,
	}
}

type Connection struct {
	ConnectionInChannel
	id              ConnectionId
	connectionType  channeldpb.ConnectionType
	compressionType channeldpb.CompressionType
	conn            net.Conn
	readBuffer      []byte
	readPos         int
	// reader          *bufio.Reader
	// writer          *bufio.Writer
	sender               MessageSender
	sendQueue            chan *channeldpb.MessagePack //MessageContext
	pit                  string
	fsm                  *fsm.FiniteStateMachine
	fsmDisallowedCounter int
	logger               *Logger
	state                int32 // Don't put the connection state into the FSM as 1) the FSM's states are user-defined. 2) the FSM is not goroutine-safe.
	connTime             time.Time
	closeHandlers        []func()
	replaySession        *replaypb.ReplaySession
	spatialSubscriptions *xsync.MapOf[common.ChannelId, *channeldpb.ChannelSubscriptionOptions]
}

var allConnections *xsync.MapOf[ConnectionId, *Connection]
var nextConnectionId uint32 = 0
var serverFsm *fsm.FiniteStateMachine
var clientFsm *fsm.FiniteStateMachine

func InitConnections(serverFsmPath string, clientFsmPath string) {
	if allConnections != nil {
		return
	}

	allConnections = xsync.NewTypedMapOf[ConnectionId, *Connection](UintIdHasher[ConnectionId]())

	bytes, err := os.ReadFile(serverFsmPath)
	if err == nil {
		serverFsm, err = fsm.Load(bytes)
	}
	if err != nil {
		rootLogger.Panic("failed to read server FSM",
			zap.Error(err),
		)
	} else {
		rootLogger.Info("loaded server FSM",
			zap.String("path", serverFsmPath),
			zap.String("currentState", serverFsm.CurrentState().Name),
		)
	}

	bytes, err = os.ReadFile(clientFsmPath)
	if err == nil {
		clientFsm, err = fsm.Load(bytes)
	}
	if err != nil {
		rootLogger.Panic("failed to read client FSM", zap.Error(err))
	} else {
		rootLogger.Info("loaded client FSM",
			zap.String("path", clientFsmPath),
			zap.String("currentState", clientFsm.CurrentState().Name),
		)
	}
}

func GetConnection(id ConnectionId) *Connection {
	c, ok := allConnections.Load(id)
	if ok {
		if c.IsClosing() {
			return nil
		}
		return c
	} else {
		return nil
	}
}

func startGoroutines(connection *Connection) {
	// receive goroutine
	go func() {
		for !connection.IsClosing() {
			connection.receive()
		}
	}()

	// tick & flush goroutine
	go func() {
		for !connection.IsClosing() {
			connection.flush()
			time.Sleep(time.Millisecond)
		}
	}()
}

func StartListening(t channeldpb.ConnectionType, network string, address string) {
	rootLogger.Info("start listenning",
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
		rootLogger.Panic("failed to listen", zap.Error(err))
		return
	}

	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			rootLogger.Error("failed to accept connection", zap.Error(err))
		} else {
			if network == "tcp" {
				tcpConn := conn.(*net.TCPConn)
				if err := tcpConn.SetReadBuffer(0x0fffff); err != nil {
					rootLogger.Error("failed to set read buffer size", zap.Error(err))
				}
				if err := tcpConn.SetWriteBuffer(0x0fffff); err != nil {
					rootLogger.Error("failed to set write buffer size", zap.Error(err))
				}
				tcpConn.SetNoDelay(true)
			}

			// Check if the IP address is banned.
			ip := GetIP(conn.RemoteAddr())
			_, banned := ipBlacklist[ip]
			if banned {
				securityLogger.Info("refused connection of banned IP address", zap.String("ip", ip))
				conn.Close()
				continue
			}

			connection := AddConnection(conn, t)
			connection.Logger().Debug("accepted connection")
			startGoroutines(connection)
		}
	}
}

func generateNextConnId(c net.Conn, maxConnId uint32) {
	if GlobalSettings.Development {
		atomic.AddUint32(&nextConnectionId, 1)
		if nextConnectionId >= maxConnId {
			// For now, we don't consider re-using the ConnectionId. Even if there are 100 incoming connections per sec, channeld can run over a year.
			rootLogger.Panic("connectionId reached the limit", zap.Uint32("maxConnId", maxConnId))
		}
	} else {
		// In non-dev mode, hash the (remote address + timestamp) to get a less guessable ID
		hash := HashString(c.RemoteAddr().String())
		hash = hash ^ uint32(time.Now().UnixNano())
		nextConnectionId = hash & maxConnId
	}
}

// NOT goroutine-safe. NEVER call AddConnection in different goroutines.
func AddConnection(c net.Conn, t channeldpb.ConnectionType) *Connection {
	var readerSize int
	// var writerSize int
	if t == channeldpb.ConnectionType_SERVER {
		readerSize = GlobalSettings.ServerReadBufferSize
		// writerSize = GlobalSettings.ServerWriteBufferSize
	} else if t == channeldpb.ConnectionType_CLIENT {
		readerSize = GlobalSettings.ClientReadBufferSize
		// writerSize = GlobalSettings.ClientWriteBufferSize
	} else {
		rootLogger.Panic("invalid connection type", zap.Int32("connType", int32(t)))
	}
	if readerSize < MaxPacketSize+PacketHeaderSize {
		readerSize = MaxPacketSize + PacketHeaderSize
	}
	maxConnId := uint32(1)<<GlobalSettings.MaxConnectionIdBits - 1

	for tries := 0; ; tries++ {
		generateNextConnId(c, maxConnId)
		if _, exists := allConnections.Load(ConnectionId(nextConnectionId)); !exists {
			break
		}

		rootLogger.Warn("there's a same connId existing, will try to generate a new one", zap.Uint32("connId", nextConnectionId))
		if tries >= 100 {
			rootLogger.Panic("could not find non-duplicate connId")
		}
	}

	connection := &Connection{
		id:              ConnectionId(nextConnectionId),
		connectionType:  t,
		compressionType: channeldpb.CompressionType_NO_COMPRESSION,
		conn:            c,
		readBuffer:      make([]byte, readerSize),
		readPos:         0,
		// reader:    bufio.NewReaderSize(c, readerSize),
		// writer:    bufio.NewWriterSize(c, writerSize),
		sender:               &queuedMessagePackSender{},
		sendQueue:            make(chan *channeldpb.MessagePack, 128),
		fsmDisallowedCounter: 0,
		logger: &Logger{rootLogger.With(
			zap.String("connType", t.String()),
			zap.Uint32("connId", nextConnectionId),
		)},
		state:                ConnectionState_UNAUTHENTICATED,
		connTime:             time.Now(),
		closeHandlers:        make([]func(), 0),
		spatialSubscriptions: xsync.NewTypedMapOf[common.ChannelId, *channeldpb.ChannelSubscriptionOptions](UintIdHasher[common.ChannelId]()),
	}

	if connection.isPacketRecordingEnabled() {
		connection.replaySession = &replaypb.ReplaySession{
			Packets: make([]*replaypb.ReplayPacket, 0, 1024),
		}
	}

	switch t {
	case channeldpb.ConnectionType_SERVER:
		if serverFsm != nil {
			// IMPORTANT: always make a value copy
			fsm := *serverFsm
			connection.fsm = &fsm
		}
	case channeldpb.ConnectionType_CLIENT:
		if clientFsm != nil {
			// IMPORTANT: always make a value copy
			fsm := *clientFsm
			connection.fsm = &fsm
		}
	}

	if connection.fsm == nil {
		rootLogger.Panic("cannot set the FSM for connection", zap.String("connType", t.String()))
	}

	allConnections.Store(connection.id, connection)

	if GlobalSettings.ConnectionAuthTimeoutMs > 0 {
		unauthenticatedConnections.Store(connection.id, connection)
	}

	connectionNum.WithLabelValues(t.String()).Inc()

	return connection
}

func (c *Connection) AddCloseHandler(handlerFunc func()) {
	c.closeHandlers = append(c.closeHandlers, handlerFunc)
}

func (c *Connection) Close() {
	defer func() {
		recover()
	}()
	if c.IsClosing() {
		c.Logger().Debug("connection is already closed")
		return
	}

	if c.isPacketRecordingEnabled() {
		c.persistReplaySession()
	}

	for _, handlerFunc := range c.closeHandlers {
		handlerFunc()
	}

	atomic.StoreInt32(&c.state, ConnectionState_CLOSING)
	c.conn.Close()
	close(c.sendQueue)
	allConnections.Delete(c.id)
	unauthenticatedConnections.Delete(c.id)

	c.Logger().Info("closed connection")
	connectionNum.WithLabelValues(c.connectionType.String()).Dec()
}

func (c *Connection) IsClosing() bool {
	return c.state > ConnectionState_AUTHENTICATED
}

func (c *Connection) receive() {
	// Read all bytes into the buffer at once
	readPtr := c.readBuffer[c.readPos:]
	bytesRead, err := c.conn.Read(readPtr)
	if err != nil {
		switch err := err.(type) {
		case *net.OpError:
			c.Logger().Warn("read bytes",
				zap.String("op", err.Op),
				zap.String("remoteAddr", c.conn.RemoteAddr().String()),
				zap.Error(err),
			)
		case *websocket.CloseError:
			c.Logger().Info("disconnected",
				zap.String("remoteAddr", c.conn.RemoteAddr().String()),
			)
		}

		if err == io.EOF {
			c.Logger().Info("disconnected",
				zap.String("remoteAddr", c.conn.RemoteAddr().String()),
			)
		}
		c.Close()
		return
	}
	c.readPos += bytesRead
	if c.readPos < PacketHeaderSize {
		// Unfinished header
		fragmentedPacketCount.WithLabelValues(c.connectionType.String()).Inc()
		return
	}

	bufPos := 0
	for bufPos = 0; bufPos < c.readPos; {
		if c.readPacket(&bufPos) == nil {
			break
		}
		if bufPos < c.readPos {
			combinedPacketCount.WithLabelValues(c.connectionType.String()).Inc()
		}
	}

	if c.IsClosing() {
		return
	}

	if bufPos < c.readPos {
		// Move unhandled content to the front
		copy(c.readBuffer, c.readBuffer[bufPos:c.readPos])
	}

	// Move read position
	c.readPos -= bufPos
}

func readSize(tag []byte) int {
	if tag[0] != 67 || tag[1] != 72 {
		return 0
	}

	size := int(tag[3]) | int(tag[2])<<8

	return size
}

func (c *Connection) readPacket(bufPos *int) *channeldpb.Packet {
	tag := c.readBuffer[*bufPos : *bufPos+PacketHeaderSize]

	packetSize := readSize(tag)
	if packetSize == 0 {
		c.readPos = 0
		connectionClosed.WithLabelValues(c.connectionType.String()).Inc()
		c.Logger().Warn("invalid tag, the connection will be closed",
			zap.Binary("tag", tag),
		)
		c.Close()
		return nil
	}

	if packetSize > MaxPacketSize {
		c.readPos = 0
		connectionClosed.WithLabelValues(c.connectionType.String()).Inc()
		c.Logger().Warn("packet size exceeds the limit, the connection will be closed", zap.Int("packetSize", packetSize), zap.Int("bufferSize", len(c.readBuffer)))
		c.Close()
		return nil
	}

	fullSize := PacketHeaderSize + packetSize

	if c.readPos < *bufPos+fullSize {
		// Unfinished packet
		fragmentedPacketCount.WithLabelValues(c.connectionType.String()).Inc()
		// this is a normal case, turn off the logs
		//c.Logger().Info("read part of package", zap.Int("readpos", c.readPos), zap.Int("full size", fullSize))
		return nil
	}

	bytes := c.readBuffer[*bufPos+PacketHeaderSize : *bufPos+fullSize]

	bytesReceived.WithLabelValues(c.connectionType.String()).Add(float64(fullSize))

	// Apply the decompression from the 5th byte in the header
	ct := tag[4]
	_, valid := channeldpb.CompressionType_name[int32(ct)]
	if valid && ct != 0 {
		c.compressionType = channeldpb.CompressionType(ct)
		if c.compressionType == channeldpb.CompressionType_SNAPPY {
			len, err := snappy.DecodedLen(bytes)
			if err != nil {
				c.Logger().Error("snappy.DecodedLen", zap.Error(err))
				return nil
			}
			dst := make([]byte, len)
			bytes, err = snappy.Decode(dst, bytes)
			if err != nil {
				c.Logger().Error("snappy.Decode", zap.Error(err))
				return nil
			}
		}
	}

	var p channeldpb.Packet
	if err := proto.Unmarshal(bytes, &p); err != nil {
		c.Logger().Error("failed to unmarshall packet, the connection will be closed", zap.Error(err),
			zap.Uint32("size", uint32(packetSize)),
			zap.Binary("tag", tag),
		)
		//if c.connectionType == channeldpb.ConnectionType_CLIENT {
		connectionClosed.WithLabelValues(c.connectionType.String()).Inc()
		c.Close()
		/*
			} else {
				// Drop the packet sent from server
				*bufPos += fullSize
			}
		*/
		return nil
	}

	packetReceived.WithLabelValues(c.connectionType.String()).Inc()

	if c.isPacketRecordingEnabled() {
		c.recordPacket(&p)
	}

	for _, mp := range p.Messages {
		c.receiveMessage(mp)
	}

	*bufPos += fullSize
	return &p
}

func (c *Connection) isPacketRecordingEnabled() bool {
	return c.connectionType == channeldpb.ConnectionType_CLIENT && GlobalSettings.EnableRecordPacket
}

func (c *Connection) receiveMessage(mp *channeldpb.MessagePack) {
	channel := GetChannel(common.ChannelId(mp.ChannelId))
	if channel == nil {
		c.Logger().Warn("can't find channel",
			zap.Uint32("channelId", mp.ChannelId),
			zap.Uint32("msgType", mp.MsgType),
		)
		return
	}

	entry := MessageMap[channeldpb.MessageType(mp.MsgType)]
	if entry == nil && mp.MsgType < uint32(channeldpb.MessageType_USER_SPACE_START) {
		c.Logger().Error("undefined message type", zap.Uint32("msgType", mp.MsgType))
		return
	}

	if !c.fsm.IsAllowed(mp.MsgType) {
		Event_FsmDisallowed.Broadcast(c)
		c.Logger().Warn("message is not allowed for current state",
			zap.Uint32("msgType", mp.MsgType),
			zap.String("connState", c.fsm.CurrentState().Name),
		)
		return
	}

	var msg common.Message
	var handler MessageHandlerFunc
	if mp.MsgType >= uint32(channeldpb.MessageType_USER_SPACE_START) && entry == nil {
		// client -> channeld -> server
		if c.connectionType == channeldpb.ConnectionType_CLIENT {
			// User-space message without handler won't be deserialized.
			msg = &channeldpb.ServerForwardMessage{ClientConnId: uint32(c.id), Payload: mp.MsgBody}
			handler = handleClientToServerUserMessage
		} else {
			// server -> channeld -> client/server
			msg = &channeldpb.ServerForwardMessage{}
			err := proto.Unmarshal(mp.MsgBody, msg)
			if err != nil {
				c.Logger().Error("unmarshalling ServerForwardMessage", zap.Error(err))
				return
			}
			handler = HandleServerToClientUserMessage
		}
	} else {
		handler = entry.handler
		// Always make a clone!
		msg = proto.Clone(entry.msg)
		err := proto.Unmarshal(mp.MsgBody, msg)
		if err != nil {
			c.Logger().Error("unmarshalling message", zap.Error(err))
			return
		}
	}

	c.fsm.OnReceived(mp.MsgType)

	channel.PutMessage(msg, handler, c, mp)

	c.Logger().VeryVerbose("received message", zap.Uint32("msgType", mp.MsgType), zap.Int("size", len(mp.MsgBody)))
	//c.Logger().Debug("received message", zap.Uint32("msgType", mp.MsgType), zap.Int("size", len(mp.MsgBody)))

	msgReceived.WithLabelValues(c.connectionType.String()).Inc() /*.WithLabelValues(
		strconv.FormatUint(uint64(p.ChannelId), 10),
		strconv.FormatUint(uint64(p.MsgType), 10),
	)*/
}

func (c *Connection) Send(ctx MessageContext) {
	if c.IsClosing() {
		return
	}

	c.sender.Send(c, ctx)
}

// Should NOT be called outside the flush goroutine!
func (c *Connection) flush() {
	if len(c.sendQueue) == 0 {
		return
	}

	p := channeldpb.Packet{Messages: make([]*channeldpb.MessagePack, 0, len(c.sendQueue))}
	size := 0

	// For now we don't limit the message numbers per packet
	for len(c.sendQueue) > 0 {
		mp := <-c.sendQueue
		p.Messages = append(p.Messages, mp)
		size = proto.Size(&p)
		if size > MaxPacketSize {
			c.Logger().Info("packet is going to be oversized",
				zap.Int("packetSize", size),
				zap.Uint32("msgType", uint32(mp.MsgType)),
				zap.Int("msgSize", len(mp.MsgBody)),
				zap.Int("msgNum", len(p.Messages)),
				zap.Int("msgInQueue", len(c.sendQueue)),
			)

			// Revert adding the message that causes the oversize
			p.Messages = p.Messages[:len(p.Messages)-1]

			// Put the message back to the queue
			// FIXME: order may matter
			c.sendQueue <- mp
			break
		}

		c.Logger().VeryVerbose("sent message", zap.Uint32("msgType", uint32(mp.MsgType)), zap.Int("size", len(mp.MsgBody)))

		msgSent.WithLabelValues(c.connectionType.String()).Inc() /*.WithLabelValues(
			strconv.FormatUint(uint64(e.Channel.id), 10),
			strconv.FormatUint(uint64(e.MsgType), 10),
		)*/
	}

	bytes, err := proto.Marshal(&p)
	if err != nil {
		c.Logger().Error("failed to marshal packet", zap.Error(err))
		return
	}

	// Apply the compression
	if c.compressionType == channeldpb.CompressionType_SNAPPY {
		dst := make([]byte, snappy.MaxEncodedLen(len(bytes)))
		bytes = snappy.Encode(dst, bytes)
	}

	// 'CHNL' in ASCII
	tag := []byte{67, 72, 78, 76, byte(c.compressionType)}
	len := len(bytes)
	tag[3] = byte(len & 0xff)
	tag[2] = byte((len >> 8) & 0xff)
	if len > MaxPacketSize {
		// Should never happen, but log it just in case
		c.Logger().Error("packet is oversized", zap.Int("size", len))
		return
	}

	/* Avoid writing multple times. With WebSocket, every Write() sends a message.
	writer.Write(tag)
	*/
	bytes = append(tag, bytes...)
	/*
		_, err = c.writer.Write(bytes)
		if err != nil {
			c.Logger().Error("error writing packet", zap.Error(err))
			return
		}

		c.writer.Flush()
	*/
	len, err = c.conn.Write(bytes)
	if err != nil {
		c.Logger().Error("error writing packet", zap.Error(err))
	}

	packetSent.WithLabelValues(c.connectionType.String()).Inc()
	bytesSent.WithLabelValues(c.connectionType.String()).Add(float64(len))
}

func (c *Connection) Disconnect() error {
	return c.conn.Close()
}

func (c *Connection) Id() ConnectionId {
	return c.id
}

func (c *Connection) GetConnectionType() channeldpb.ConnectionType {
	return c.connectionType
}

func (c *Connection) OnAuthenticated(pit string) {
	if c.IsClosing() {
		return
	}

	atomic.StoreInt32(&c.state, ConnectionState_AUTHENTICATED)

	unauthenticatedConnections.Delete(c.id)

	c.pit = pit

	if !c.fsm.MoveToNextState() {
		c.Logger().Error("no state found after the authenticated state")
	}
}

func (c *Connection) String() string {
	return fmt.Sprintf("Connection(%s %d %s)", c.connectionType, c.id, c.fsm.CurrentState().Name)
}

func (c *Connection) Logger() *Logger {
	return c.logger
}

func (c *Connection) RemoteAddr() net.Addr {
	/* The address should still be available even after the connection is closed.
	 * In this way, the anit-DDoS can save the address to the blacklist.
	if c.IsClosing() {
		return nil
	}
	*/
	return c.conn.RemoteAddr()
}

func (c *Connection) recordPacket(p *channeldpb.Packet) {

	recordedPacket := &channeldpb.Packet{
		Messages: make([]*channeldpb.MessagePack, 0, len(p.Messages)),
	}
	proto.Merge(recordedPacket, p)

	c.replaySession.Packets = append(c.replaySession.Packets, &replaypb.ReplayPacket{
		OffsetTime: time.Now().UnixNano(),
		Packet:     recordedPacket,
	})
}

func (c *Connection) persistReplaySession() {

	var prevPacketTime int64
	if len(c.replaySession.Packets) > 0 {
		prevPacketTime = c.replaySession.Packets[0].OffsetTime
	} else {
		c.Logger().Error("replay session is empty")
		return
	}

	for _, packet := range c.replaySession.Packets {
		t := packet.OffsetTime
		packet.OffsetTime -= prevPacketTime
		prevPacketTime = t
	}

	data, err := proto.Marshal(c.replaySession)
	if err != nil {
		c.Logger().Error("failed to marshal replay session", zap.Error(err))
		return
	}

	var dir string
	if GlobalSettings.ReplaySessionPersistenceDir != "" {
		dir = GlobalSettings.ReplaySessionPersistenceDir
	} else {
		dir = "replays"
	}

	_, err = os.Stat(dir)
	if err == nil || !os.IsExist(err) {
		os.MkdirAll(dir, 0777)
	}

	path := filepath.Join(dir, fmt.Sprintf("session_%d_%s.cpr", c.id, time.Now().Local().Format("06-01-02_15-04-03")))
	err = os.WriteFile(path, data, 0777)
	if err != nil {
		c.Logger().Error("failed to write replay session to location", zap.Error(err))
	}

}
