package channeld

import (
	"log"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/xtaci/kcp-go"
)

func TestKCPConnection(t *testing.T) {
	InitLogsAndMetrics()
	const addr string = "localhost:12108"
	go func() {
		StartListening(CLIENT, "kcp", addr)
	}()
	_, err := kcp.Dial(addr)
	assert.NoError(t, err)
}

func TestWebSocketConnection(t *testing.T) {
	InitLogsAndMetrics()
	const addr string = "ws://localhost:8080"
	go func() {
		StartListening(CLIENT, "ws", addr)
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
			AddConnection(nil, CLIENT)
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
