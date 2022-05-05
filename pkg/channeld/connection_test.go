package channeld

import (
	"bufio"
	"io"
	"log"
	"sync"
	"testing"
	"time"

	"channeld.clewcat.com/channeld/proto"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/xtaci/kcp-go"
	protobuf "google.golang.org/protobuf/proto"
)

func TestDropPacket(t *testing.T) {
	GlobalSettings.Development = true
	InitLogsAndMetrics()
	pipeReader, pipeWriter := io.Pipe()
	c := &Connection{
		reader: bufio.NewReader(pipeReader),
		logger: logger,
	}

	go func() {
		for {
			c.ReceivePacket()
		}
	}()

	pipeWriter.Write([]byte{1, 2, 3, 4, 5, 6})
	time.Sleep(time.Millisecond * 100)

	msg := &proto.TestChannelDataMessage{Text: "abc", Num: 123}
	msgBody, _ := protobuf.Marshal(msg)
	p := &proto.Packet{
		Messages: []*proto.MessagePack{
			{
				MsgBody: msgBody,
			},
		},
	}
	bytes, _ := protobuf.Marshal(p)
	size := byte(len(bytes))
	pipeWriter.Write(append([]byte{67, 72, 78, size, 0}, bytes...))

	time.Sleep(time.Millisecond * 100)
	pipeWriter.Write([]byte{0})
}

func TestKCPConnection(t *testing.T) {
	InitLogsAndMetrics()
	const addr string = "localhost:12108"
	go func() {
		StartListening(proto.ConnectionType_CLIENT, "kcp", addr)
	}()
	_, err := kcp.Dial(addr)
	assert.NoError(t, err)
}

func TestWebSocketConnection(t *testing.T) {
	InitLogsAndMetrics()
	const addr string = "ws://localhost:8080"
	go func() {
		StartListening(proto.ConnectionType_CLIENT, "ws", addr)
	}()
	_, _, err := websocket.DefaultDialer.Dial(addr, nil)
	assert.NoError(t, err)
}

func TestConcurrentAccessConnections(t *testing.T) {
	InitLogsAndMetrics()
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		for i := 0; i < 100; i++ {
			AddConnection(nil, proto.ConnectionType_CLIENT)
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
