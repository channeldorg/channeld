package channeld

import (
	"testing"

	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"github.com/stretchr/testify/assert"
)

func TestSubscribeToChannel(t *testing.T) {
	InitLogs()
	c1 := &Connection{id: 1, connectionType: channeldpb.ConnectionType_SERVER}
	//c2 := &Connection{id: 2, connectionType: SERVER}
	//c3 := &Connection{id: 3, connectionType: CLIENT}

	InitChannels()
	assert.NotNil(t, globalChannel)
	// Can't create the GLOBAL channel
	_, err := CreateChannel(channeldpb.ChannelType_GLOBAL, nil)
	assert.Error(t, err)
	// By default, the GLOBAL channel has no owner
	assert.True(t, !globalChannel.HasOwner())

	globalChannel.ownerConnection = c1
	c1.SubscribeToChannel(globalChannel, nil)
	assert.Contains(t, globalChannel.subscribedConnections, c1)

}
