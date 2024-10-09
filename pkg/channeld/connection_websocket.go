package channeld

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/channeldorg/channeld/pkg/channeldpb"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type wsConn struct {
	conn *websocket.Conn
}

func (c *wsConn) Read(b []byte) (n int, err error) {
	_, body, err := c.conn.ReadMessage()
	return copy(b, body), err
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

var trustedOrigins []string

func SetWebSocketTrustedOrigins(addrs []string) {
	trustedOrigins = addrs
}

var upgrader websocket.Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		if trustedOrigins == nil {
			return true
		} else {
			for _, addr := range trustedOrigins {
				if addr == r.RemoteAddr {
					return true
				}
			}
			return false
		}
	},
}

func startWebSocketServer(t channeldpb.ConnectionType, address string) {
	if protocolIndex := strings.Index(address, "://"); protocolIndex >= 0 {
		address = address[protocolIndex+3:]
	}

	pattern := "/"
	if pathIndex := strings.Index(address, "/"); pathIndex >= 0 {
		pattern = address[pathIndex:]
		address = address[:pathIndex-1]
	}

	mux := http.NewServeMux()
	connsToAdd := make(chan *websocket.Conn, 128)
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			rootLogger.Panic("Upgrade to websocket connection", zap.Error(err))
		}
		// Add the websocket connection to a blocking queue instead of calling AddConnection() immediately,
		// as a new goroutines is created per request.
		connsToAdd <- conn
	})

	serverClosed := false
	// Call AddConnection() in a separate goroutine, to avoid the race condition.
	go func() {
		for !serverClosed {
			conn := <-connsToAdd
			c := AddConnection(&wsConn{conn}, t)
			startGoroutines(c)
		}
	}()

	server := http.Server{
		Addr:    address,
		Handler: mux,
	}

	defer server.Close()

	rootLogger.Error("stopped listening", zap.Error(server.ListenAndServe()))
	serverClosed = true
}
