package channeld

import (
	"math"
	"testing"

	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"channeld.clewcat.com/channeld/pkg/common"
	"github.com/stretchr/testify/assert"
)

func TestGetAdjacentChannels(t *testing.T) {
	InitLogs()

	// 1-by-1-grid world
	ctl1 := &StaticGrid2DSpatialController{
		WorldOffsetX:             0,
		WorldOffsetZ:             0,
		GridWidth:                10,
		GridHeight:               10,
		GridCols:                 1,
		GridRows:                 1,
		ServerCols:               1,
		ServerRows:               1,
		ServerInterestBorderSize: 1,
	}
	channelIds, err := ctl1.GetAdjacentChannels(GlobalSettings.SpatialChannelIdStart)
	assert.NoError(t, err)
	assert.Empty(t, channelIds)

	// 2-by-2-grid world
	ctl2 := &StaticGrid2DSpatialController{
		WorldOffsetX:             -5,
		WorldOffsetZ:             -5,
		GridWidth:                5,
		GridHeight:               5,
		GridCols:                 2,
		GridRows:                 2,
		ServerCols:               1,
		ServerRows:               1,
		ServerInterestBorderSize: 0,
	}
	channelIds, err = ctl2.GetAdjacentChannels(GlobalSettings.SpatialChannelIdStart)
	assert.NoError(t, err)
	assert.Len(t, channelIds, 3)

}

func TestCreateSpatialChannels3(t *testing.T) {
	InitLogs()

	// 2-by-2-grid world, 1:1 grid:server; servers don't have border
	ctl := &StaticGrid2DSpatialController{
		WorldOffsetX:             0,
		WorldOffsetZ:             0,
		GridWidth:                33,
		GridHeight:               77,
		GridCols:                 2,
		GridRows:                 2,
		ServerCols:               2,
		ServerRows:               2,
		ServerInterestBorderSize: 0,
	}

	testConn := createTestConnection()
	ctx := MessageContext{
		MsgType:    channeldpb.MessageType_CREATE_CHANNEL,
		Msg:        &channeldpb.CreateChannelMessage{},
		Connection: testConn,
	}

	for {
		channels, err := ctl.CreateChannels(ctx)
		if err != nil {
			break
		} else {
			assert.Len(t, channels, 1)
		}
	}
	assert.Empty(t, testConn.subscribedChannels)

	testConn.closing = true
	ctl.Tick()
	assert.EqualValues(t, 0, ctl.nextServerIndex())

	testConn.closing = false
	channels, err := ctl.CreateChannels(ctx)
	assert.NoError(t, err)
	assert.EqualValues(t, channels[0].id, GlobalSettings.SpatialChannelIdStart)
	assert.EqualValues(t, 1, ctl.nextServerIndex())
}

func TestCreateSpatialChannels2(t *testing.T) {
	InitLogs()

	// 1-by-1-grid world consists of 1-by-1-grid server
	ctl := &StaticGrid2DSpatialController{
		WorldOffsetX:             0,
		WorldOffsetZ:             0,
		GridWidth:                10,
		GridHeight:               10,
		GridCols:                 1,
		GridRows:                 1,
		ServerCols:               1,
		ServerRows:               1,
		ServerInterestBorderSize: 1,
	}

	testConn := createTestConnection()
	ctx := MessageContext{
		MsgType:    channeldpb.MessageType_CREATE_CHANNEL,
		Msg:        &channeldpb.CreateChannelMessage{},
		Connection: testConn,
	}

	channels, err := ctl.CreateChannels(ctx)
	assert.NoError(t, err)
	assert.Len(t, channels, 1)
	assert.Empty(t, testConn.subscribedChannels)

	testConn.closing = true
	ctl.Tick()
	assert.EqualValues(t, 0, ctl.nextServerIndex())

	testConn.closing = false
	channels, err = ctl.CreateChannels(ctx)
	assert.NoError(t, err)
	assert.EqualValues(t, channels[0].id, GlobalSettings.SpatialChannelIdStart)
	assert.EqualValues(t, 1, ctl.nextServerIndex())
	_, err = ctl.CreateChannels(ctx)
	assert.Error(t, err)
}

func TestCreateSpatialChannels1(t *testing.T) {
	InitLogs()

	// 4-by-3-grid world consists of 2-by-1-grid servers - there are 2x3=6 servers.
	ctl := &StaticGrid2DSpatialController{
		WorldOffsetX:             -40,
		WorldOffsetZ:             -60,
		GridWidth:                20,
		GridHeight:               40,
		GridCols:                 4,
		GridRows:                 3,
		ServerCols:               2,
		ServerRows:               3,
		ServerInterestBorderSize: 1,
	}

	conns := make([]*testConnection, 6)
	for i := range conns {
		conns[i] = createTestConnection()
	}

	ctx := MessageContext{
		MsgType: channeldpb.MessageType_CREATE_CHANNEL,
		Msg:     &channeldpb.CreateChannelMessage{},
	}

	ctx.Connection = conns[0]
	server0Channels, _ := ctl.CreateChannels(ctx)
	assert.Len(t, server0Channels, 2)
	assert.Equal(t, GlobalSettings.SpatialChannelIdStart+0, server0Channels[0].id)
	assert.Equal(t, GlobalSettings.SpatialChannelIdStart+1, server0Channels[1].id)

	for i := 1; i < 6; i++ {
		ctx.Connection = conns[i]
		channels, err := ctl.CreateChannels(ctx)
		assert.NoError(t, err)
		assert.Len(t, channels, 2)
	}
	assert.EqualValues(t, 6, ctl.nextServerIndex())

	/* Grids and Servers:
	3  2  |  1  0
	-------------
	7  6  |  5  4
	-------------
	11 10 |  9  8
	*/
	assert.Contains(t, conns[0].subscribedChannels, GlobalSettings.SpatialChannelIdStart+2)
	assert.Contains(t, conns[0].subscribedChannels, GlobalSettings.SpatialChannelIdStart+4)
	assert.Contains(t, conns[0].subscribedChannels, GlobalSettings.SpatialChannelIdStart+5)

	assert.Contains(t, conns[1].subscribedChannels, GlobalSettings.SpatialChannelIdStart+1)
	assert.Contains(t, conns[1].subscribedChannels, GlobalSettings.SpatialChannelIdStart+6)
	assert.Contains(t, conns[1].subscribedChannels, GlobalSettings.SpatialChannelIdStart+7)

	assert.Contains(t, conns[2].subscribedChannels, GlobalSettings.SpatialChannelIdStart+0)
	assert.Contains(t, conns[2].subscribedChannels, GlobalSettings.SpatialChannelIdStart+1)
	assert.Contains(t, conns[2].subscribedChannels, GlobalSettings.SpatialChannelIdStart+6)
	assert.Contains(t, conns[2].subscribedChannels, GlobalSettings.SpatialChannelIdStart+8)
	assert.Contains(t, conns[2].subscribedChannels, GlobalSettings.SpatialChannelIdStart+9)

	assert.Contains(t, conns[3].subscribedChannels, GlobalSettings.SpatialChannelIdStart+2)
	assert.Contains(t, conns[3].subscribedChannels, GlobalSettings.SpatialChannelIdStart+3)
	assert.Contains(t, conns[3].subscribedChannels, GlobalSettings.SpatialChannelIdStart+5)
	assert.Contains(t, conns[3].subscribedChannels, GlobalSettings.SpatialChannelIdStart+10)
	assert.Contains(t, conns[3].subscribedChannels, GlobalSettings.SpatialChannelIdStart+11)

	assert.Contains(t, conns[5].subscribedChannels, GlobalSettings.SpatialChannelIdStart+6)
	assert.Contains(t, conns[5].subscribedChannels, GlobalSettings.SpatialChannelIdStart+7)
	assert.Contains(t, conns[5].subscribedChannels, GlobalSettings.SpatialChannelIdStart+9)
}

type testConnection struct {
	sentMsgs           []MessageContext
	subscribedChannels map[ChannelId]*Channel
	closing            bool
}

func createTestConnection() *testConnection {
	return &testConnection{
		sentMsgs:           make([]MessageContext, 0),
		subscribedChannels: make(map[ChannelId]*Channel),
	}
}

func (c *testConnection) Id() ConnectionId {
	return 0
}

func (c *testConnection) GetConnectionType() channeldpb.ConnectionType {
	return channeldpb.ConnectionType_NO_CONNECTION
}

func (c *testConnection) OnAuthenticated() {

}

func (c *testConnection) HasAuthorityOver(ch *Channel) bool {
	return false
}

func (c *testConnection) Close() {
	c.closing = true
}

func (c *testConnection) IsClosing() bool {
	return c.closing
}

func (c *testConnection) Send(ctx MessageContext) {

}

func (c *testConnection) SubscribeToChannel(ch *Channel, options *channeldpb.ChannelSubscriptionOptions) *ChannelSubscription {
	c.subscribedChannels[ch.id] = ch
	return &ChannelSubscription{
		options: channeldpb.ChannelSubscriptionOptions{},
	}
}

func (c *testConnection) UnsubscribeFromChannel(ch *Channel) (*channeldpb.ChannelSubscriptionOptions, error) {
	delete(c.subscribedChannels, ch.id)
	return nil, nil
}

func (c *testConnection) sendSubscribed(ctx MessageContext, ch *Channel, connToSub ConnectionInChannel, stubId uint32, subOptions *channeldpb.ChannelSubscriptionOptions) {

}

func (c *testConnection) sendUnsubscribed(ctx MessageContext, ch *Channel, connToUnsub *Connection, stubId uint32) {

}

func (c *testConnection) Logger() *Logger {
	return rootLogger
}

func TestGetChannelId2(t *testing.T) {
	// 9-by-8-grid world consists of 3-by-2-grid servers - there are 3x4=12 servers.
	ctl := &StaticGrid2DSpatialController{
		WorldOffsetX:             0,
		WorldOffsetZ:             0,
		GridWidth:                100,
		GridHeight:               50,
		GridCols:                 9,
		GridRows:                 8,
		ServerCols:               3,
		ServerRows:               4,
		ServerInterestBorderSize: 2,
	}

	var channelId ChannelId
	var err error
	channelId, _ = ctl.GetChannelId(common.SpatialInfo{X: 0, Z: 0})
	assert.Equal(t, GlobalSettings.SpatialChannelIdStart+0, channelId)

	channelId, _ = ctl.GetChannelId(common.SpatialInfo{X: 100, Z: 0})
	assert.Equal(t, GlobalSettings.SpatialChannelIdStart+1, channelId)

	channelId, _ = ctl.GetChannelId(common.SpatialInfo{X: 0, Z: 50})
	assert.Equal(t, GlobalSettings.SpatialChannelIdStart+ChannelId(ctl.GridCols), channelId)

	channelId, _ = ctl.GetChannelId(common.SpatialInfo{X: 899.99, Z: 399.99})
	assert.Equal(t, GlobalSettings.SpatialChannelIdStart+9*8-1, channelId)

	_, err = ctl.GetChannelId(common.SpatialInfo{X: -1, Z: 0})
	assert.Error(t, err)

	_, err = ctl.GetChannelId(common.SpatialInfo{X: math.MaxFloat64, Z: 0})
	assert.Error(t, err)

	_, err = ctl.GetChannelId(common.SpatialInfo{X: 0, Z: -1})
	assert.Error(t, err)

	_, err = ctl.GetChannelId(common.SpatialInfo{X: 900, Z: 400})
	assert.Error(t, err)
}

func TestGetChannelId1(t *testing.T) {
	// 9-by-8-grid world consists of 3-by-2-grid servers - there are 3x4=12 servers.
	ctl := &StaticGrid2DSpatialController{
		WorldOffsetX:             -450,
		WorldOffsetZ:             -200,
		GridWidth:                100,
		GridHeight:               50,
		GridCols:                 9,
		GridRows:                 8,
		ServerCols:               3,
		ServerRows:               4,
		ServerInterestBorderSize: 2,
	}

	var channelId ChannelId
	var err error
	channelId, _ = ctl.GetChannelId(common.SpatialInfo{X: -450, Z: -200})
	assert.Equal(t, GlobalSettings.SpatialChannelIdStart+0, channelId)

	channelId, _ = ctl.GetChannelId(common.SpatialInfo{X: -350, Z: -200})
	assert.Equal(t, GlobalSettings.SpatialChannelIdStart+1, channelId)

	channelId, _ = ctl.GetChannelId(common.SpatialInfo{X: -450, Z: -150})
	assert.Equal(t, GlobalSettings.SpatialChannelIdStart+ChannelId(ctl.GridCols), channelId)

	channelId, _ = ctl.GetChannelId(common.SpatialInfo{X: 0, Z: 0})
	assert.Equal(t, GlobalSettings.SpatialChannelIdStart+9*4+4, channelId)

	channelId, _ = ctl.GetChannelId(common.SpatialInfo{X: 449.99, Z: 199.99})
	assert.Equal(t, GlobalSettings.SpatialChannelIdStart+9*8-1, channelId)

	_, err = ctl.GetChannelId(common.SpatialInfo{X: -500, Z: 0})
	assert.Error(t, err)

	_, err = ctl.GetChannelId(common.SpatialInfo{X: 500, Z: 0})
	assert.Error(t, err)

	_, err = ctl.GetChannelId(common.SpatialInfo{X: 0, Z: -300})
	assert.Error(t, err)

	_, err = ctl.GetChannelId(common.SpatialInfo{X: 0, Z: 300})
	assert.Error(t, err)

	_, err = ctl.GetChannelId(common.SpatialInfo{X: 450, Z: 200})
	assert.Error(t, err)
}
