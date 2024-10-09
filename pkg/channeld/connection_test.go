package channeld

import (
	"io"
	"log"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/channeldorg/channeld/internal/testpb"
	"github.com/channeldorg/channeld/pkg/channeldpb"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/xtaci/kcp-go"
	"google.golang.org/protobuf/proto"
)

func init() {
	GlobalSettings.Development = true
	InitLogs()
	InitConnections("../../config/server_conn_fsm_test.json", "../../config/client_non_authoratative_fsm.json")
}

type pipelineConn struct {
	net.Conn
	r *io.PipeReader
	w *io.PipeWriter
}

func (c *pipelineConn) Read(b []byte) (n int, err error) {
	return c.r.Read(b)
}

func (c *pipelineConn) Write(b []byte) (n int, err error) {
	return c.w.Write(b)
}

func (c *pipelineConn) Close() error {
	err1 := c.r.Close()
	err2 := c.w.Close()
	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return nil
}

func (c *pipelineConn) LocalAddr() net.Addr {
	return nil
}

func (c *pipelineConn) RemoteAddr() net.Addr {
	return nil
}

func (c *pipelineConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *pipelineConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *pipelineConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func TestReadSize(t *testing.T) {
	var tag []byte

	tag = []byte{0, 0, 0, 0}
	assert.Equal(t, 0, readSize(tag))
	tag[3] = 1
	assert.Equal(t, 0, readSize(tag))

	tag = []byte{67, 0, 0, 0}
	assert.Equal(t, 0, readSize(tag))
	tag[3] = 1
	assert.Equal(t, 0, readSize(tag))

	tag = []byte{67, 72, 78, 0}
	assert.Equal(t, 78<<8, readSize(tag))
	tag[3] = 1
	assert.Equal(t, 78<<8+1, readSize(tag))

	tag = []byte{67, 72, 0, 0}
	assert.Equal(t, 0, readSize(tag))
	tag[3] = 1
	assert.Equal(t, 1, readSize(tag))
}

func TestDropPacket(t *testing.T) {
	pipeReader, pipeWriter := io.Pipe()
	c := &Connection{
		conn:       &pipelineConn{r: pipeReader, w: pipeWriter},
		readBuffer: make([]byte, 1024),
		logger:     rootLogger,
	}

	end := false
	go func() {
		for !end {
			c.receive()
		}
	}()

	pipeWriter.Write([]byte{1, 2, 3, 4, 5, 6})
	time.Sleep(time.Millisecond * 100)

	msg := &testpb.TestChannelDataMessage{Text: "abc", Num: 123}
	msgBody, _ := proto.Marshal(msg)
	p := &channeldpb.Packet{
		Messages: []*channeldpb.MessagePack{
			{
				MsgBody: msgBody,
			},
		},
	}
	bytes, _ := proto.Marshal(p)
	size := byte(len(bytes))
	pipeWriter.Write(append([]byte{67, 72, 78, size, 0}, bytes...))

	time.Sleep(time.Millisecond * 100)
	pipeWriter.Write([]byte{0})

	end = true
}

func TestKCPConnection(t *testing.T) {
	const addr string = "127.0.0.1:12108"
	go func() {
		StartListening(channeldpb.ConnectionType_CLIENT, "kcp", addr)
	}()
	sess, err := kcp.DialWithOptions(addr, nil, 0, 0)
	assert.NoError(t, err)
	_, err = sess.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.NoError(t, sess.Close())
}

func TestWebSocketConnection(t *testing.T) {
	const addr string = "ws://localhost:8080"
	go func() {
		StartListening(channeldpb.ConnectionType_CLIENT, "ws", addr)
	}()
	_, _, err := websocket.DefaultDialer.Dial(addr, nil)
	assert.NoError(t, err)

	_, _, err = websocket.DefaultDialer.Dial("ws://localhost:8081", nil)
	assert.Error(t, err)
}

func TestConcurrentAccessConnections(t *testing.T) {
	InitConnections("../../config/server_conn_fsm_test.json", "../../config/client_non_authoratative_fsm.json")

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		for i := 0; i < 100; i++ {
			AddConnection(nil, channeldpb.ConnectionType_CLIENT)
			time.Sleep(1 * time.Millisecond)
		}
		wg.Done()
	}()

	// Read-Write ratio = 100:1
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			counter := 0
			for i := 0; i < 100; i++ {
				if GetConnection(ConnectionId(i)) != nil {
					counter++
				}
				time.Sleep(1 * time.Millisecond)
			}
			log.Println(counter)
			wg.Done()
		}()
	}

	wg.Wait()

}
