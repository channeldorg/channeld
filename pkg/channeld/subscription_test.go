package channeld

import (
	"testing"
	"time"

	"github.com/metaworking/channeld/pkg/channeldpb"
	"github.com/metaworking/channeld/pkg/common"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func init() {
	InitLogs()
	InitChannels()
}

func TestSubscribeToChannel(t *testing.T) {
	c1 := &Connection{id: 1, connectionType: channeldpb.ConnectionType_SERVER}
	//c2 := &Connection{id: 2, connectionType: SERVER}
	//c3 := &Connection{id: 3, connectionType: CLIENT}

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

func randomChannelId(maxChId int) common.ChannelId {
	// [0, 999]
	chId := time.Now().Nanosecond() % (maxChId - 1)
	// [1, 1000]
	return common.ChannelId(chId + 1)
}

func BenchmarkHandoverEntitySub(b *testing.B) {
	// Disable logging
	SetLogLevel(zap.FatalLevel)

	CONN_NUM := 1000
	ENTITY_CHANNEL_NUM := 1000
	SPATIAL_CHANNEL_NUM := 15 * 15
	CONN_PER_GRID := CONN_NUM / SPATIAL_CHANNEL_NUM

	conns := make([]*Connection, CONN_NUM)
	for i := 0; i < CONN_NUM; i++ {
		conns[i] = addTestConnection(channeldpb.ConnectionType_CLIENT) //&Connection{id: ConnectionId(i), connectionType: channeldpb.ConnectionType_CLIENT}

	}

	for i := 0; i < ENTITY_CHANNEL_NUM; i++ {
		CreateChannel(channeldpb.ChannelType_ENTITY, conns[i])
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, conn := range conns {
			// Every client subscribes to 3 spatial channels and unsubscribes from 3 spatial channels
			for nSub := 0; nSub < 3*CONN_PER_GRID; nSub++ {
				conn.SubscribeToChannel(GetChannel(randomChannelId(ENTITY_CHANNEL_NUM)), nil)
			}
			for nUnsub := 0; nUnsub < 3*CONN_PER_GRID; nUnsub++ {
				conn.UnsubscribeFromChannel(GetChannel(randomChannelId(ENTITY_CHANNEL_NUM)))
			}
		}
	}
}

// Result:
// BenchmarkHandoverEntitySub-20    	     645	   2498163 ns/op	  985149 B/op	   22501 allocs/op
// 1000 clients handing over at the same time takes 2.5ms - Acceptable
// 985KB / 22K allocs - A bit too much
