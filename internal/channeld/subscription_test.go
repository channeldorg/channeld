package channeld

import (
	"testing"

	"clewcat.com/channeld/proto"
	"github.com/stretchr/testify/assert"
)

func TestSubscribeToChannel(t *testing.T) {
	c1 := &Connection{id: 1, connectionType: SERVER}
	//c2 := &Connection{id: 2, connectionType: SERVER}
	//c3 := &Connection{id: 3, connectionType: CLIENT}

	InitChannels()
	assert.NotNil(t, globalChannel)
	// Can't create the GLOBAL channel
	assert.Panics(t, func() {
		CreateChannel(proto.ChannelType_GLOBAL, nil)
	})
	// By default, the GLOBAL channel has no owner
	assert.Nil(t, globalChannel.ownerConnection)

	globalChannel.ownerConnection = c1
	assert.NoError(t, c1.SubscribeToChannel(globalChannel, ChannelSubscriptionOptions{}))
	assert.Contains(t, globalChannel.subscribedConnections, c1.id)

}
