package channeld

import (
	"math/rand"
	"sync"
	"testing"

	"github.com/channeldorg/channeld/pkg/channeldpb"
	"github.com/channeldorg/channeld/pkg/common"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func init() {
	InitLogs()
	InitChannels()
}

func TestSubscribeToChannel(t *testing.T) {
	c1 := addTestConnection(channeldpb.ConnectionType_SERVER)

	assert.NotNil(t, globalChannel)
	// Can't create the GLOBAL channel
	_, err := CreateChannel(channeldpb.ChannelType_GLOBAL, nil)
	assert.Error(t, err)
	// By default, the GLOBAL channel has no owner
	assert.True(t, !globalChannel.HasOwner())

	globalChannel.SetOwner(c1)
	shouldSend := false
	_, shouldSend = c1.SubscribeToChannel(globalChannel, nil)
	assert.Contains(t, globalChannel.subscribedConnections, c1)
	assert.True(t, shouldSend)

	// Subscribe again
	_, shouldSend = c1.SubscribeToChannel(globalChannel, nil)
	assert.Contains(t, globalChannel.subscribedConnections, c1)
	assert.False(t, shouldSend)

	// Subscribe with different DataAccess
	_, shouldSend = c1.SubscribeToChannel(globalChannel, &channeldpb.ChannelSubscriptionOptions{
		DataAccess: Pointer(channeldpb.ChannelDataAccess_WRITE_ACCESS),
	})
	assert.True(t, shouldSend)
}

func randomChannelId(maxChId int) common.ChannelId {
	// [0, 999]
	chId := rand.Intn(maxChId) //time.Now().Nanosecond() % (maxChId - 1)
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

/*
Result:
BenchmarkHandoverEntitySub-20    	      81	  12666246 ns/op	 7495425 B/op	  100522 allocs/op
1000 clients handing over at the same time takes 13ms - Acceptable
7.5MB / 100K allocs - A bit too much
*/

func BenchmarkHandoverEntitySubAsync(b *testing.B) {
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
		wg := sync.WaitGroup{}
		for s := 0; s < SPATIAL_CHANNEL_NUM; s++ {
			gridIndex := s
			wg.Add(1)
			go func() {
				for nConn := 0; nConn < CONN_PER_GRID; nConn++ {
					conn := conns[gridIndex*CONN_PER_GRID+nConn]
					// Every client subscribes to 3 spatial channels and unsubscribes from 3 spatial channels
					for nSub := 0; nSub < 3*CONN_PER_GRID; nSub++ {
						conn.SubscribeToChannel(GetChannel(randomChannelId(ENTITY_CHANNEL_NUM)), nil)
					}
					for nUnsub := 0; nUnsub < 3*CONN_PER_GRID; nUnsub++ {
						conn.UnsubscribeFromChannel(GetChannel(randomChannelId(ENTITY_CHANNEL_NUM)))
					}
				}
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

/*
// Result:
BenchmarkHandoverEntitySubAsync-20    	     147	   9056136 ns/op	 5960059 B/op	   82705 allocs/op
1000 clients handing over at the same time takes 9ms - Acceptable
6MB / 82K allocs - A bit too much
*/
