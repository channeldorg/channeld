package channeld

import (
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/metaworking/channeld/pkg/channeldpb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestUnauthTimeout(t *testing.T) {
	InitLogs()
	InitAntiDDoS()
	// InitChannels()
	InitConnections("../../config/server_conn_fsm_test.json", "../../config/client_non_authoratative_fsm.json")

	GlobalSettings.ConnectionAuthTimeoutMs = 1000

	// go StartListening(channeldpb.ConnectionType_SERVER, "tcp", ":31288")
	go StartListening(channeldpb.ConnectionType_CLIENT, "tcp", ":32108")
	time.Sleep(time.Millisecond * 100)

	conn, err := net.Dial("tcp", "127.0.0.1:32108")
	assert.NoError(t, err, "Error connecting to server")

	assert.True(t, checkConnOpen(conn), "Connection should not have been closed yet")

	time.Sleep(time.Millisecond * time.Duration(GlobalSettings.ConnectionAuthTimeoutMs))
	assert.True(t, checkConnClosed(conn), "Connection should have been closed by now")

	// IP blacklisted. Should still be able to connect, but will be soon be disconnected
	conn, _ = net.Dial("tcp", "127.0.0.1:32108")
	time.Sleep(time.Millisecond * 100)
	assert.True(t, checkConnClosed(conn), "Connection should have been closed by now")
}

func TestInvalidUsername(t *testing.T) {
	InitLogs()
	InitAntiDDoS()
	InitChannels()
	InitConnections("../../config/server_conn_fsm_test.json", "../../config/client_non_authoratative_fsm.json")

	// Turn off dev mode to force authentication
	GlobalSettings.Development = false
	GlobalSettings.MaxFailedAuthAttempts = 2
	SetAuthProvider(&AlwaysFailAuthProvider{})

	go StartListening(channeldpb.ConnectionType_CLIENT, "tcp", ":32108")
	time.Sleep(time.Millisecond * 100)

	conn, err := net.Dial("tcp", "127.0.0.1:32108")
	assert.NoError(t, err, "Error connecting to server")

	sendMessage(conn, uint32(channeldpb.MessageType_AUTH), &channeldpb.AuthMessage{
		PlayerIdentifierToken: "user1",
	})
	time.Sleep(time.Millisecond * 100)
	assert.Contains(t, failedAuthCounters, "127.0.0.1", "Failed auth counter should contain 127.0.0.1")
	assert.True(t, checkConnOpen(conn), "Connection should not have been closed yet")

	sendMessage(conn, uint32(channeldpb.MessageType_AUTH), &channeldpb.AuthMessage{
		PlayerIdentifierToken: "user2",
	})
	time.Sleep(time.Millisecond * 100)
	assert.EqualValues(t, 2, failedAuthCounters["127.0.0.1"], "Failed auth counter should be 2")
	// Read all bytes to clear the read buffer, so checkConnOpen/Closed can work
	readAll(conn)
	assert.False(t, checkConnOpen(conn), "Connection should have been closed now")
	/*
		_, err = conn.Write([]byte{0, 0, 0, 0})
		assert.Error(t, err, "Connection should have been closed now")
	*/

	// IP blacklisted. Should still be able to connect, but will be soon be disconnected
	conn, _ = net.Dial("tcp", "127.0.0.1:32108")
	time.Sleep(time.Millisecond * 100)
	assert.True(t, checkConnClosed(conn), "Connection should have been closed now")
}

func TestWrongPassword(t *testing.T) {
	InitLogs()
	InitAntiDDoS()
	InitChannels()
	InitConnections("../../config/server_conn_fsm_test.json", "../../config/client_non_authoratative_fsm.json")

	// Turn off dev mode to force authentication
	GlobalSettings.Development = false
	GlobalSettings.MaxFailedAuthAttempts = 3
	SetAuthProvider(&FixedPasswordAuthProvider{"rightpassword"})

	go StartListening(channeldpb.ConnectionType_CLIENT, "tcp", ":32108")
	time.Sleep(time.Millisecond * 100)

	conn, err := net.Dial("tcp", "127.0.0.1:32108")
	assert.NoError(t, err, "Error connecting to server")

	sendMessage(conn, uint32(channeldpb.MessageType_AUTH), &channeldpb.AuthMessage{
		PlayerIdentifierToken: "user1",
		LoginToken:            "wrongpassword",
	})
	time.Sleep(time.Millisecond * 100)
	assert.Contains(t, failedAuthCounters, "user1", "Failed auth counter should contain user1")
	assert.True(t, checkConnOpen(conn), "Connection should not have been closed yet")

	sendMessage(conn, uint32(channeldpb.MessageType_AUTH), &channeldpb.AuthMessage{
		PlayerIdentifierToken: "user1",
		LoginToken:            "wrongpassword",
	})
	time.Sleep(time.Millisecond * 100)
	assert.EqualValues(t, 2, failedAuthCounters["user1"], "Failed auth counter should be 2")
	assert.True(t, checkConnOpen(conn), "Connection should not have been closed yet")

	sendMessage(conn, uint32(channeldpb.MessageType_AUTH), &channeldpb.AuthMessage{
		PlayerIdentifierToken: "user1",
		LoginToken:            "rightpassword",
	})
	time.Sleep(time.Millisecond * 100)
	assert.EqualValues(t, 2, failedAuthCounters["user1"], "Failed auth counter should still be 2")
	assert.True(t, checkConnOpen(conn), "Connection should not have been closed yet")

	conn.Close()
	time.Sleep(time.Millisecond * 100)
	// Re-open connection as the FSM only allows valid AuthMessage once
	conn, err = net.Dial("tcp", "127.0.0.1:32108")
	assert.NoError(t, err, "Error connecting to server")

	sendMessage(conn, uint32(channeldpb.MessageType_AUTH), &channeldpb.AuthMessage{
		PlayerIdentifierToken: "user1",
		LoginToken:            "wrongpassword",
	})
	time.Sleep(time.Millisecond * 100)
	assert.False(t, checkConnOpen(conn), "Connection should have been closed by now")

	// PIT blacklisted. Should still be able to connect, but can't login anymore
	conn, _ = net.Dial("tcp", "127.0.0.1:32108")
	time.Sleep(time.Millisecond * 100)
	assert.True(t, checkConnOpen(conn), "Connection should have been closed by now")
	sendMessage(conn, uint32(channeldpb.MessageType_AUTH), &channeldpb.AuthMessage{
		PlayerIdentifierToken: "user1",
		LoginToken:            "rightpassword",
	})
	time.Sleep(time.Millisecond * 100)
	// Event the right password won't work anymore
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

func readAll(conn net.Conn) []byte {
	buff := make([]byte, 1024)
	n, _ := conn.Read(buff)
	return buff[:n]
}

func checkConnOpen(conn net.Conn) bool {
	buff := make([]byte, 1)
	conn.SetReadDeadline(time.Now().Add(time.Millisecond * 50))
	_, err := conn.Read(buff)
	if err != nil {
		return errors.Is(err, os.ErrDeadlineExceeded)
	}
	return true
}

func checkConnClosed(conn net.Conn) bool {
	buff := make([]byte, 1)
	conn.SetReadDeadline(time.Time{})
	_, err := conn.Read(buff)
	return err != nil
}
