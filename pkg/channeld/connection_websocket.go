package channeld

import (
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

type WSConn struct {
	conn *websocket.Conn
}

func (c *WSConn) Read(b []byte) (n int, err error) {
	_, body, err := c.conn.ReadMessage()
	return copy(b, body), err
}

func (c *WSConn) Write(b []byte) (n int, err error) {
	return len(b), c.conn.WriteMessage(websocket.BinaryMessage, b)
}

func (c *WSConn) Close() error {
	return c.conn.Close()
}

func (c *WSConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *WSConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *WSConn) SetDeadline(t time.Time) error {
	return c.conn.UnderlyingConn().SetDeadline(t)
}

func (c *WSConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *WSConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

var upgrader websocket.Upgrader = websocket.Upgrader{}

func startWebSocketServer(t ConnectionType, address string) {
	u, err := url.Parse(address)
	if err != nil {
		log.Panic("Invalid address: ", err)
	}

	pattern := u.Path
	if len(pattern) == 0 {
		pattern = "/"
	}
	http.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Panic(err)
		}
		c := AddConnection(&WSConn{conn}, t)
		startGoroutines(c)
	})
	http.ListenAndServe(address, nil)
}
