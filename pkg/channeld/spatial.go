package channeld

import (
	"errors"
	"fmt"
	"math"

	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"channeld.clewcat.com/channeld/pkg/common"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
)

type SpatialController interface {
	// Notify() is called in the spatial channels (shared instance)
	common.SpatialInfoChangedNotifier
	// Called in GLOBAL and spatial channels
	GetChannelId(info common.SpatialInfo) (ChannelId, error)
	// Called in GLOBAL channel
	GetRegions() ([]*channeldpb.SpatialRegion, error)
	// Called in any spatial channel
	GetAdjacentChannels(spatialChannelId ChannelId) ([]ChannelId, error)
	// Create spatial channels for a spatial server.
	// Called in GLOBAL channel
	CreateChannels(ctx MessageContext) ([]*Channel, error)
	// Called in GLOBAL channel
	Tick()
}

// A channeld instance should have only one SpatialController
var spatialController SpatialController

func InitSpatialController(controller SpatialController) {
	spatialController = controller
}

const (
	MinY = -3.40282347e+38 / 2
	MaxY = 3.40282347e+38 / 2
)

// Divide the world into GridCols x GridRows static squares on the XZ plane. Each square(grid) represents a spatial channel.
// Typically, a player's view distance is 150m, so a grid is sized at 50x50m.
// A 100x100 km world has 2000x2000 grids, which needs about 2^22 spatial channels.
// By default, we support up to 2^32-2^16 grid-based spatial channels.
type StaticGrid2DSpatialController struct {
	SpatialController

	/* Defines how the world is divided into grids */
	// The width of a grid in simulation/engine units
	GridWidth float64
	// The heights of a grid in the simulation/engine units
	GridHeight float64
	// How many grids the world has in X axis. The width of the world = GridWidth x GridCols.
	GridCols uint32
	// How many grids the world has in Z axis. The height of the world = GridHeight x GridRows.
	GridRows uint32

	// WorldWidth  float64
	// WorldHeight float64

	// In the right-handed coordinate system, the difference between the world origin and the top-right corner of the first grid, in the simulation/engine units.
	// This is how we uses the offset value to calculate which grid a (x,z) point is in: gridX = Floor((x - OffsetX) / GridWidth), gridY = Floor((y - OffsetY) / GridHeight)
	// If the world origin is exactly in the middle of the world, the offset should be (-WorldWidth*0.5, -WorldHeight*0.5).
	WorldOffsetX float64
	WorldOffsetZ float64

	/* Defines the authority area of a spatial server, as well as the number of the servers (= ServerCols * ServerRows) */
	// How many servers the world has in X axis.
	ServerCols uint32
	// How many servers the world has in Z axis.
	ServerRows uint32

	/* Defines the extra interest area a spatial server has, adjacent to the authority area */
	// For each side of a server's grids (authority area), how many grids(spatial channels) the server subscribes to, as the extend of its interest area.
	// For example, ServerInterestBorderSize = 1 means a spatial server of 3x3 grids has interest area of 4x4 grids.
	// Remarks: the value should always be less than the size of the authority area (=Min(GridCols/ServerCols, GridRows/ServerRows))
	ServerInterestBorderSize uint32

	//serverIndex       uint32
	serverConnections []ConnectionInChannel
}

func (ctl *StaticGrid2DSpatialController) GetChannelId(info common.SpatialInfo) (ChannelId, error) {
	return ctl.GetChannelIdWithOffset(info, ctl.WorldOffsetX, ctl.WorldOffsetZ)
}

func (ctl *StaticGrid2DSpatialController) GetChannelIdNoOffset(info common.SpatialInfo) (ChannelId, error) {
	return ctl.GetChannelIdWithOffset(info, 0, 0)
}

func (ctl *StaticGrid2DSpatialController) GetChannelIdWithOffset(info common.SpatialInfo, offsetX float64, offsetZ float64) (ChannelId, error) {
	gridX := int(math.Floor((info.X - offsetX) / ctl.GridWidth))
	if gridX < 0 || gridX >= int(ctl.GridCols) {
		return 0, fmt.Errorf("gridX=%d when X=%f. GridX should be in [0,%d)", gridX, info.X, ctl.GridCols)
	}
	gridY := int(math.Floor((info.Z - offsetZ) / ctl.GridHeight))
	if gridY < 0 || gridY >= int(ctl.GridRows) {
		return 0, fmt.Errorf("gridY=%d when Z=%f. GridY should be in [0,%d)", gridY, info.Z, ctl.GridRows)
	}
	index := uint32(gridX) + uint32(gridY)*ctl.GridCols
	return ChannelId(index) + GlobalSettings.SpatialChannelIdStart, nil
}

func (ctl *StaticGrid2DSpatialController) GetRegions() ([]*channeldpb.SpatialRegion, error) {
	// How many grids a server has in X axis
	serverGridCols := ctl.GridCols / ctl.ServerCols
	if ctl.GridCols%ctl.ServerCols > 0 {
		serverGridCols++
	}
	// How many grids a server has in Z axis
	serverGridRows := ctl.GridRows / ctl.ServerRows
	if ctl.GridRows%ctl.ServerRows > 0 {
		serverGridRows++
	}

	regions := make([]*channeldpb.SpatialRegion, ctl.GridCols*ctl.GridRows)
	//var MinFloat64 = math.Inf(-1)
	for y := uint32(0); y < ctl.GridRows; y++ {
		for x := uint32(0); x < ctl.GridCols; x++ {
			index := x + y*ctl.GridCols
			serverX := x / serverGridCols
			serverY := y / serverGridRows

			regions[index] = &channeldpb.SpatialRegion{
				Min: &channeldpb.SpatialInfo{
					X: ctl.WorldOffsetX + ctl.GridWidth*float64(x),
					Y: MinY,
					Z: ctl.WorldOffsetZ + ctl.GridHeight*float64(y),
				},
				Max: &channeldpb.SpatialInfo{
					X: ctl.WorldOffsetX + ctl.GridWidth*float64(x+1),
					Y: MaxY,
					Z: ctl.WorldOffsetZ + ctl.GridHeight*float64(y+1),
				},
				ChannelId:   uint32(GlobalSettings.SpatialChannelIdStart) + index,
				ServerIndex: serverX + serverY*ctl.ServerCols,
			}
		}
	}
	return regions, nil
}

func (ctl *StaticGrid2DSpatialController) GetAdjacentChannels(spatialChannelId ChannelId) ([]ChannelId, error) {
	index := uint32(spatialChannelId - GlobalSettings.SpatialChannelIdStart)
	gridX := int32(index % ctl.GridCols)
	gridY := int32(index / ctl.GridCols)
	channelIds := make([]ChannelId, 0)
	for y := gridY - 1; y <= gridY+1; y++ {
		if y < 0 || y >= int32(ctl.GridRows-1) {
			continue
		}

		for x := gridX - 1; x <= gridX+1; x++ {
			if x < 0 || x >= int32(ctl.GridCols-1) {
				continue
			}
			if x == gridX && y == gridY {
				continue
			}

			channelIndex := uint32(x) + uint32(y)*ctl.GridCols
			channelIds = append(channelIds, ChannelId(channelIndex)+GlobalSettings.SpatialChannelIdStart)
		}
	}
	return channelIds, nil
}

func (ctl *StaticGrid2DSpatialController) CreateChannels(ctx MessageContext) ([]*Channel, error) {
	ctl.initServerConnections()
	serverIndex := ctl.nextServerIndex()
	if serverIndex >= ctl.ServerCols*ctl.ServerRows {
		return nil, fmt.Errorf("failed to create spatail channel as all %d grids are allocated to %d servers", ctl.GridCols*ctl.GridRows, ctl.ServerCols*ctl.ServerRows)
	}

	msg, ok := ctx.Msg.(*channeldpb.CreateChannelMessage)
	if !ok {
		return nil, errors.New("ctx.Msg is not a CreateChannelMessage, will not be handled")
	}

	// How many grids a server has in X axis
	serverGridCols := ctl.GridCols / ctl.ServerCols
	if ctl.GridCols%ctl.ServerCols > 0 {
		serverGridCols++
	}
	// How many grids a server has in Z axis
	serverGridRows := ctl.GridRows / ctl.ServerRows
	if ctl.GridRows%ctl.ServerRows > 0 {
		serverGridRows++
	}

	channelIds := make([]ChannelId, serverGridCols*serverGridRows)
	serverX := serverIndex % ctl.ServerCols
	serverY := serverIndex / ctl.ServerCols
	var spatialInfo common.SpatialInfo
	for y := uint32(0); y < serverGridRows; y++ {
		for x := uint32(0); x < serverGridCols; x++ {
			spatialInfo.X = float64(serverX*serverGridCols+x) * ctl.GridWidth
			spatialInfo.Z = float64(serverY*serverGridRows+y) * ctl.GridHeight
			channelId, err := ctl.GetChannelIdNoOffset(spatialInfo)
			if err != nil {
				return nil, err
			}
			channelIds[x+y*serverGridCols] = channelId
		}
	}

	channels := make([]*Channel, len(channelIds))
	for index, channelId := range channelIds {
		channel := createChannelWithId(channelId, channeldpb.ChannelType_SPATIAL, ctx.Connection)
		if msg.Data != nil {
			dataMsg, err := msg.Data.UnmarshalNew()
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal data message for the new channel: %v", err)
			} else {
				channel.InitData(dataMsg, msg.MergeOptions)
			}
		} else {
			// Channel data should always be initialized
			channel.InitData(nil, msg.MergeOptions)
		}

		channels[index] = channel
	}

	// Save the connection for later use
	ctl.serverConnections[serverIndex] = ctx.Connection
	//ctl.serverIndex++
	serverIndex = ctl.nextServerIndex()
	// When all spatial channels are created, subscribe each server to its adjacent grids(channels)
	if serverIndex == ctl.ServerCols*ctl.ServerRows {
		for i := uint32(0); i < serverIndex; i++ {
			err := ctl.subToAdjacentChannels(i, serverGridCols, serverGridRows, msg.SubOptions)
			if err != nil {
				return channels, fmt.Errorf("failed to sub to adjacent channels of server connection %d, err: %v", ctl.serverConnections[i].Id(), err)
			}
		}
	}

	return channels, nil
}

func (ctl *StaticGrid2DSpatialController) subToAdjacentChannels(serverIndex uint32, serverGridCols uint32, serverGridRows uint32, subOptions *channeldpb.ChannelSubscriptionOptions) error {
	serverConn := ctl.serverConnections[serverIndex]
	serverX := serverIndex % ctl.ServerCols
	serverY := serverIndex / ctl.ServerCols
	spatialInfo := common.SpatialInfo{
		X: float64(serverX*serverGridCols) * ctl.GridWidth,
		Z: float64(serverY*serverGridRows) * ctl.GridHeight,
	}
	serverChannelId, err := ctl.GetChannelIdNoOffset(spatialInfo)
	if err != nil {
		return err
	}
	serverChannel := GetChannel(serverChannelId)
	if serverChannel == nil {
		return fmt.Errorf("failed to subscribe to adjacent channels for  %d as it doesn't exist", serverChannelId)
	}

	// Right border
	if serverX > 0 {
		for y := uint32(0); y < serverGridRows; y++ {
			for x := uint32(1); x <= ctl.ServerInterestBorderSize; x++ {
				spatialInfo.X = float64(serverX*serverGridCols-x) * ctl.GridWidth
				spatialInfo.Z = float64(serverY*serverGridRows+y) * ctl.GridHeight
				channelId, err := ctl.GetChannelIdNoOffset(spatialInfo)
				if err != nil {
					return err
				}
				channelToSub := GetChannel(channelId)
				if channelToSub == nil {
					return fmt.Errorf("failed to subscribe border channel %d as it doesn't exist", channelId)
				}
				cs := serverConn.SubscribeToChannel(channelToSub, subOptions)
				if cs != nil {
					serverConn.sendSubscribed(MessageContext{}, channelToSub, serverConn, 0, &cs.options)
				}
			}
		}
	}

	// Left border
	if serverX < ctl.ServerCols-1 {
		for y := uint32(0); y < serverGridRows; y++ {
			for x := uint32(0); x < ctl.ServerInterestBorderSize; x++ {
				spatialInfo.X = float64((serverX+1)*serverGridCols+x) * ctl.GridWidth
				spatialInfo.Z = float64(serverY*serverGridRows+y) * ctl.GridHeight
				channelId, err := ctl.GetChannelIdNoOffset(spatialInfo)
				if err != nil {
					return err
				}
				channelToSub := GetChannel(channelId)
				if channelToSub == nil {
					return fmt.Errorf("failed to subscribe border channel %d as it doesn't exist", channelId)
				}
				cs := serverConn.SubscribeToChannel(channelToSub, subOptions)
				if cs != nil {
					serverConn.sendSubscribed(MessageContext{}, channelToSub, serverConn, 0, &cs.options)
				}
			}
		}
	}

	// Top border
	if serverY > 0 {
		for y := uint32(1); y <= ctl.ServerInterestBorderSize; y++ {
			for x := uint32(0); x < serverGridCols; x++ {
				spatialInfo.X = float64(serverX*serverGridCols+x) * ctl.GridWidth
				spatialInfo.Z = float64(serverY*serverGridRows-y) * ctl.GridHeight
				channelId, err := ctl.GetChannelIdNoOffset(spatialInfo)
				if err != nil {
					return err
				}
				channelToSub := GetChannel(channelId)
				if channelToSub == nil {
					return fmt.Errorf("failed to subscribe border channel %d as it doesn't exist", channelId)
				}
				cs := serverConn.SubscribeToChannel(channelToSub, subOptions)
				if cs != nil {
					serverConn.sendSubscribed(MessageContext{}, channelToSub, serverConn, 0, &cs.options)
				}
			}
		}
	}

	// Bottom border
	if serverY < ctl.ServerRows-1 {
		for y := uint32(0); y < ctl.ServerInterestBorderSize; y++ {
			for x := uint32(0); x < serverGridCols; x++ {
				spatialInfo.X = float64(serverX*serverGridCols+x) * ctl.GridWidth
				spatialInfo.Z = float64((serverY+1)*serverGridRows+y) * ctl.GridHeight
				channelId, err := ctl.GetChannelIdNoOffset(spatialInfo)
				if err != nil {
					return err
				}
				channelToSub := GetChannel(channelId)
				if channelToSub == nil {
					return fmt.Errorf("failed to subscribe border channel %d as it doesn't exist", channelId)
				}
				cs := serverConn.SubscribeToChannel(channelToSub, subOptions)
				if cs != nil {
					serverConn.sendSubscribed(MessageContext{}, channelToSub, serverConn, 0, &cs.options)
				}
			}
		}
	}
	return nil
}

var dataMarshalOptions = protojson.MarshalOptions{Multiline: false}

// Runs in the source spatial channel (shared instance)
func (ctl *StaticGrid2DSpatialController) Notify(oldInfo common.SpatialInfo, newInfo common.SpatialInfo, handoverDataProvider func() common.ChannelDataMessage) {
	srcChannelId, err := ctl.GetChannelId(oldInfo)
	if err != nil {
		rootLogger.Error("failed to calculate srcChannelId", zap.Error(err))
		return
	}
	dstChannelId, err := ctl.GetChannelId(newInfo)
	if err != nil {
		rootLogger.Error("failed to calculate dstChannelId", zap.Error(err))
		return
	}
	// No migration between channels
	if dstChannelId == srcChannelId {
		return
	}

	srcChannel := GetChannel(srcChannelId)
	if srcChannel == nil {
		rootLogger.Error("channel doesn't exist, failed to handover channel data", zap.Uint32("srcChannelId", uint32(srcChannelId)))
		return
	}
	if !srcChannel.HasOwner() {
		rootLogger.Error("channel doesn't have owner, failed to handover channel data", zap.Uint32("srcChannelId", uint32(srcChannelId)))
	}

	dstChannel := GetChannel(dstChannelId)
	if dstChannel == nil {
		rootLogger.Error("channel doesn't exist, failed to handover channel data", zap.Uint32("dstChannelId", uint32(dstChannelId)))
		return
	}
	if !srcChannel.HasOwner() {
		rootLogger.Error("channel doesn't have owner, failed to handover channel data", zap.Uint32("dstChannelId", uint32(dstChannelId)))
	}

	// Handover data is provider by the Merger [channeld.MergeableChannelData]
	handoverData := handoverDataProvider()
	if handoverData == nil {
		rootLogger.Error("failed to provider handover channel data", zap.Uint32("srcChannelId", uint32(srcChannelId)), zap.Uint32("dstChannelId", uint32(dstChannelId)))
		return
	}
	rootLogger.Debug("handover channel data", zap.Uint32("srcChannelId", uint32(srcChannelId)), zap.Uint32("dstChannelId", uint32(dstChannelId)),
		zap.String("data", dataMarshalOptions.Format(handoverData)))

	anyData, err := anypb.New(handoverData)
	if err != nil {
		rootLogger.Error("failed to marshall handover data", zap.Error(err))
		return
	}

	/*
		newChannel.PutMessage(&channeldpb.ChannelDataUpdateMessage{
			Data: anyData,
		}, handleChannelDataUpdate, internalDummyConnection, &channeldpb.MessagePack{
			MsgType:   uint32(channeldpb.MessageType_CHANNEL_DATA_UPDATE),
			Broadcast: channeldpb.BroadcastType_NO_BROADCAST,
			StubId:    0,
			ChannelId: uint32(dstChannelId),
		})
	*/

	handoverMsg := &channeldpb.ChannelDataHandoverMessage{
		SrcChannelId:  uint32(srcChannelId),
		DstChannelId:  uint32(dstChannelId),
		Data:          anyData,
		ContextConnId: uint32(srcChannel.latestDataUpdateConnId),
	}

	// Avoid duplicate sending
	// Race Condition: reading dstChannel's subscribedConnections in srcChannel
	conns := dstChannel.GetAllConnections()
	for conn := range srcChannel.subscribedConnections {
		if _, exists := conns[conn]; !exists {
			conn.Send(MessageContext{
				MsgType:   channeldpb.MessageType_CHANNEL_DATA_HANDOVER,
				Msg:       handoverMsg,
				Broadcast: 0,
				StubId:    0,
				ChannelId: uint32(srcChannelId),
			})
		}
	}
	for conn := range conns {
		conn.Send(MessageContext{
			MsgType:   channeldpb.MessageType_CHANNEL_DATA_HANDOVER,
			Msg:       handoverMsg,
			Broadcast: 0,
			StubId:    0,
			ChannelId: uint32(dstChannelId),
		})
	}

	/* Broadcast in both channels can cause duplicate sending
	srcChannel.Broadcast(MessageContext{
		MsgType:   channeldpb.MessageType_CHANNEL_DATA_HANDOVER,
		Msg:       handoverMsg,
		Broadcast: channeldpb.BroadcastType_ALL,
		StubId:    0,
		ChannelId: uint32(srcChannelId),
	})

	var broadcastType = channeldpb.BroadcastType_ALL
	// Don't send the spatial server twice (if it's the owner of both channels)
	if dstChannel.ownerConnection == srcChannel.ownerConnection {
		broadcastType = channeldpb.BroadcastType_ALL_BUT_OWNER
	}
	dstChannel.Broadcast(MessageContext{
		MsgType:   channeldpb.MessageType_CHANNEL_DATA_HANDOVER,
		Msg:       handoverMsg,
		Broadcast: broadcastType,
		StubId:    0,
		ChannelId: uint32(dstChannelId),
	})
	*/

	/* Unsub and Sub is controlled by server/client, and has nothing to do with src/dst channel in most cases.
	// Unsub from srcChannel & Sub to dstChannel
	clientConn := GetConnection(ConnectionId(ctl.contextConnId))
	if clientConn != nil {
		subOptions, err := clientConn.UnsubscribeFromChannel(srcChannel)
		if err != nil {
			rootLogger.Error("failed to unsub from channel",
				zap.String("channelType", srcChannel.channelType.String()),
				zap.Uint32("channelId", uint32(srcChannel.id)),
				zap.Error(err))
		}
		clientConn.sendUnsubscribed(MessageContext{}, srcChannel, clientConn, 0)

		clientConn.SubscribeToChannel(dstChannel, subOptions)
		clientConn.sendSubscribed(MessageContext{}, dstChannel, clientConn, 0, subOptions)
	}
	*/
}

func (ctl *StaticGrid2DSpatialController) initServerConnections() {
	if ctl.serverConnections == nil {
		ctl.serverConnections = make([]ConnectionInChannel, ctl.ServerCols*ctl.ServerRows)
	}
}

func (ctl *StaticGrid2DSpatialController) nextServerIndex() uint32 {
	var i int = 0
	for i = 0; i < len(ctl.serverConnections); i++ {
		if ctl.serverConnections[i] == nil || ctl.serverConnections[i].IsRemoving() {
			break
		}
	}
	return uint32(i)
}

func (ctl *StaticGrid2DSpatialController) Tick() {
	ctl.initServerConnections()
	for i := 0; i < len(ctl.serverConnections); i++ {
		if ctl.serverConnections[i] != nil && ctl.serverConnections[i].IsRemoving() {
			ctl.serverConnections[i] = nil
			rootLogger.Info("reset spatial server connection", zap.Int("serverIndex", i))
		}
	}
}

/*
// Used for sending message between channels
var internalDummyConnection = &Connection{
	id:              math.MaxUint32,
	connectionType:  channeldpb.ConnectionType_NO_CONNECTION,
	compressionType: channeldpb.CompressionType_NO_COMPRESSION,
	conn:            nil,
	reader:          nil,
	writer:          nil,
	sender:          nil, //&queuedMessageSender{},
	sendQueue:       nil, //make(chan MessageContext, 128),
	logger: logger.With(
		zap.String("connType", "Internal"),
	),
	removing: 0,
}
*/
