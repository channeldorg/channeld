package fsm

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func loadServerFSM(t *testing.T) FiniteStateMachine {
	bytes, err := ioutil.ReadFile("../../config/server_conn_fsm_test.json")
	if err != nil {
		t.Error(err)
	}
	serverFSM, err := Load(bytes)
	if err != nil {
		t.Error(err)
	}
	return serverFSM
}

func TestLoad(t *testing.T) {
	serverFSM := loadServerFSM(t)
	assert.Equal(t, 3, len(serverFSM.States))
	assert.Equal(t, 3, len(serverFSM.Transitions))
	assert.Equal(t, &serverFSM.States[0], serverFSM.CurrentState())
}

func TestTransitionAndMsgAllowence(t *testing.T) {
	serverFSM := loadServerFSM(t)
	assert.Equal(t, "INIT", serverFSM.CurrentState().Name)
	assert.True(t, serverFSM.IsAllowed(1))
	assert.False(t, serverFSM.IsAllowed(2))
	assert.False(t, serverFSM.IsAllowed(9))
	assert.False(t, serverFSM.IsAllowed(21))

	serverFSM.OnReceived(1)
	assert.Equal(t, "OPEN", serverFSM.CurrentState().Name)
	assert.False(t, serverFSM.IsAllowed(1))
	assert.True(t, serverFSM.IsAllowed(2))
	assert.False(t, serverFSM.IsAllowed(9))
	assert.True(t, serverFSM.IsAllowed(20))
	assert.False(t, serverFSM.IsAllowed(21))

	serverFSM.OnReceived(20)
	assert.Equal(t, "HANDOVER", serverFSM.CurrentState().Name)
	assert.False(t, serverFSM.IsAllowed(1))
	assert.False(t, serverFSM.IsAllowed(2))
	assert.True(t, serverFSM.IsAllowed(21))
	assert.False(t, serverFSM.IsAllowed(100))

	serverFSM.OnReceived(1)
	assert.Equal(t, "HANDOVER", serverFSM.CurrentState().Name)

	serverFSM.OnReceived(21)
	assert.Equal(t, "HANDOVER", serverFSM.CurrentState().Name)

	serverFSM.OnReceived(22)
	assert.Equal(t, "OPEN", serverFSM.CurrentState().Name)
}
