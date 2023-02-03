package channeld

import (
	"errors"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func Init() {
	InitLogs()
	InitAntiDDoS()
}

func TestUnauthTimeout(t *testing.T) {
	GlobalSettings.ConnectionAuthTimeoutMs = 1000

	// go StartListening(channeldpb.ConnectionType_SERVER, "tcp", ":31288")
	go StartListening(channeldpb.ConnectionType_CLIENT, "tcp", ":32108")
	go checkUnauthConns()
	time.Sleep(time.Millisecond * 100)

	conn, err := net.Dial("tcp", "127.0.0.1:32108")
	assert.NoError(t, err, "Error connecting to server")

	buff := make([]byte, 1)
	conn.SetReadDeadline(time.Now().Add(time.Millisecond * 100))
	_, err = conn.Read(buff)
	assert.ErrorIs(t, err, os.ErrDeadlineExceeded, "Connection should not been closed yet")

	time.Sleep(time.Millisecond * time.Duration(GlobalSettings.ConnectionAuthTimeoutMs))
	// Set back to blocked reading
	conn.SetReadDeadline(time.Time{})
	_, err = conn.Read(buff)
	assert.ErrorIs(t, err, io.EOF, "Connection should already have been closed by now")

	// Should still be able to connect, but will be soon be disconnected
	conn, _ = net.Dial("tcp", "127.0.0.1:32108")
	time.Sleep(time.Millisecond * 100)
	_, err = conn.Read(buff)
	assert.ErrorIs(t, err, io.EOF, "Connection should already have been closed by now")
}

func TestAuthFailure(t *testing.T) {
	// Force authentication
	GlobalSettings.Development = false
	GlobalSettings.MaxFailedAuthAttempts = 3
	SetAuthProvider(&FixedPasswordAuthProvider{"rightpassword"})

	InitChannels()

	go StartListening(channeldpb.ConnectionType_CLIENT, "tcp", ":32108")
	time.Sleep(time.Millisecond * 100)

	conn, _ := net.Dial("tcp", "127.0.0.1:32108")
	sendMessage(conn, uint32(channeldpb.MessageType_AUTH), &channeldpb.AuthMessage{
		PlayerIdentifierToken: "user1",
		LoginToken:            "wrongpassword",
	})
	assert.Contains(t, failedAuthCounters, "user1", "Failed auth counter should contain user1")
	assert.True(t, checkConnOpen(conn), "Connection should not have been closed yet")

	time.Sleep(time.Millisecond * 100)
	sendMessage(conn, uint32(channeldpb.MessageType_AUTH), &channeldpb.AuthMessage{
		PlayerIdentifierToken: "user1",
		LoginToken:            "wrongpassword",
	})
	assert.EqualValues(t, 2, failedAuthCounters["user1"], "Failed auth counter should be 2")
	assert.True(t, checkConnOpen(conn), "Connection should not have been closed yet")

	time.Sleep(time.Millisecond * 100)
	sendMessage(conn, uint32(channeldpb.MessageType_AUTH), &channeldpb.AuthMessage{
		PlayerIdentifierToken: "user1",
		LoginToken:            "rightpassword",
	})
	assert.True(t, checkConnOpen(conn), "Connection should not have been closed yet")

	time.Sleep(time.Millisecond * 100)
	sendMessage(conn, uint32(channeldpb.MessageType_AUTH), &channeldpb.AuthMessage{
		PlayerIdentifierToken: "user1",
		LoginToken:            "wrongpassword",
	})
	assert.False(t, checkConnOpen(conn), "Connection should have been closed by now")
}

func sendMessage(conn net.Conn, msgType uint32, msg proto.Message) {
	msgBody, _ := proto.Marshal(msg)
	p := &channeldpb.Packet{
		Messages: []*channeldpb.MessagePack{
			{
				MsgType: msgType,
				MsgBody: msgBody,
			},
		},
	}
	bytes, _ := proto.Marshal(p)
	tag := []byte{67, 72, 78, byte(len(bytes)), 0}
	conn.Write(append(tag, bytes...))
}

func checkConnOpen(conn net.Conn) bool {
	buff := make([]byte, 1)
	conn.SetReadDeadline(time.Now().Add(time.Millisecond * 100))
	_, err := conn.Read(buff)
	return errors.Is(err, os.ErrDeadlineExceeded)
}
