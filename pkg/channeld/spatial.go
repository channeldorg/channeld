package channeld

import (
	"errors"
	"fmt"
	"math"

	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"channeld.clewcat.com/channeld/pkg/common"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
)

type SpatialController interface {
	// Notify() is called in the spatial channels (shared instance)
	common.SpatialInfoChangedNotifier
	// Called in GLOBAL and spatial channels
	GetChannelId(info common.SpatialInfo) (common.ChannelId, error)
	// Called in the spatials channel
	QueryChannelIds(query *channeldpb.SpatialInterestQuery) (map[common.ChannelId]uint, error)
	// Called in GLOBAL channel
	GetRegions() ([]*channeldpb.SpatialRegion, error)
	// Called in the spatials channel
	GetAdjacentChannels(spatialChannelId common.ChannelId) ([]common.ChannelId, error)
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

func GetSpatialController() SpatialController {
	return spatialController
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

	gridSize float64
}

func (ctl *StaticGrid2DSpatialController) GridSize() float64 {
	if ctl.gridSize == 0 && ctl.GridWidth > 0 && ctl.GridHeight > 0 {
		ctl.gridSize = math.Sqrt(ctl.GridWidth*ctl.GridWidth + ctl.GridHeight*ctl.GridHeight)
	}
	return ctl.gridSize
}

func (ctl *StaticGrid2DSpatialController) GetChannelId(info common.SpatialInfo) (common.ChannelId, error) {
	return ctl.GetChannelIdWithOffset(info, ctl.WorldOffsetX, ctl.WorldOffsetZ)
}

func (ctl *StaticGrid2DSpatialController) GetChannelIdNoOffset(info common.SpatialInfo) (common.ChannelId, error) {
	return ctl.GetChannelIdWithOffset(info, 0, 0)
}

func (ctl *StaticGrid2DSpatialController) GetChannelIdWithOffset(info common.SpatialInfo, offsetX float64, offsetZ float64) (common.ChannelId, error) {
	gridX := int(math.Floor((info.X - offsetX) / ctl.GridWidth))
	if gridX < 0 || gridX >= int(ctl.GridCols) {
		return 0, fmt.Errorf("gridX=%d when X=%f. GridX should be in [0,%d)", gridX, info.X, ctl.GridCols)
	}
	gridY := int(math.Floor((info.Z - offsetZ) / ctl.GridHeight))
	if gridY < 0 || gridY >= int(ctl.GridRows) {
		return 0, fmt.Errorf("gridY=%d when Z=%f. GridY should be in [0,%d)", gridY, info.Z, ctl.GridRows)
	}
	index := uint32(gridX) + uint32(gridY)*ctl.GridCols
	return common.ChannelId(index) + GlobalSettings.SpatialChannelIdStart, nil
}

func (ctl *StaticGrid2DSpatialController) QueryChannelIds(query *channeldpb.SpatialInterestQuery) (map[common.ChannelId]uint, error) {
	if query == nil {
		return nil, fmt.Errorf("query is nil")
	}

	result := make(map[common.ChannelId]uint)

	if query.SpotsAOI != nil {
		for _, spot := range query.SpotsAOI.Spots {
			chId, err := ctl.GetChannelId(common.SpatialInfo{X: spot.X, Y: spot.Y, Z: spot.Z})
			if err != nil {
				return nil, err
			}
			result[chId] = uint(math.Ceil(query.SpotsAOI.Center.Dist2D(spot) / ctl.GridSize()))
		}
	}

	if query.BoxAOI != nil {
		center := &common.SpatialInfo{X: (query.BoxAOI.Min.X + query.BoxAOI.Max.X) * 0.5, Y: 0, Z: (query.BoxAOI.Min.Z + query.BoxAOI.Max.Z) * 0.5}
		for z := query.BoxAOI.Min.Z; z < query.BoxAOI.Max.Z+ctl.GridHeight; z += ctl.GridHeight {
			for x := query.BoxAOI.Min.X; x < query.BoxAOI.Max.X+ctl.GridWidth; x += ctl.GridWidth {
				spot := common.SpatialInfo{X: x, Y: 0, Z: z}
				chId, err := ctl.GetChannelId(spot)
				if err != nil {
					return nil, err
				}
				result[chId] = uint(math.Ceil(center.Dist2D(&spot) / ctl.GridSize()))
			}
		}
	}

	if query.SphereAOI != nil {
		center := &common.SpatialInfo{X: query.SphereAOI.Center.X, Y: 0, Z: query.SphereAOI.Center.Z}
		for z := center.Z - query.SphereAOI.Radius; z < center.Z+query.SphereAOI.Radius+ctl.GridHeight; z += ctl.GridHeight {
			for x := center.X - query.SphereAOI.Radius; x < center.X+query.SphereAOI.Radius+ctl.GridWidth; x += ctl.GridWidth {
				if (x-center.X)*(x-center.X)+(z-center.Z)*(z-center.Z) > query.SphereAOI.Radius*query.SphereAOI.Radius {
					continue
				}
				spot := common.SpatialInfo{X: x, Y: 0, Z: z}
				chId, err := ctl.GetChannelId(spot)
				if err != nil {
					return nil, err
				}
				result[chId] = uint(math.Ceil(center.Dist2D(&spot) / ctl.GridSize()))
			}
		}
	}

	if query.ConeAOI != nil {
		center := &common.SpatialInfo{X: query.ConeAOI.Center.X, Y: 0, Z: query.ConeAOI.Center.Z}
		coneDir := &common.SpatialInfo{X: query.ConeAOI.Direction.X, Y: 0, Z: query.ConeAOI.Direction.Z}
		coneDir.Normalize2D()
		for z := center.Z - query.ConeAOI.Radius; z < center.Z+query.ConeAOI.Radius+ctl.GridHeight; z += ctl.GridHeight {
			for x := center.X - query.ConeAOI.Radius; x < center.X+query.ConeAOI.Radius+ctl.GridWidth; x += ctl.GridWidth {
				if (x-center.X)*(x-center.X)+(z-center.Z)*(z-center.Z) > query.ConeAOI.Radius*query.ConeAOI.Radius {
					continue
				}
				spot := common.SpatialInfo{X: x, Y: 0, Z: z}
				dir := common.SpatialInfo{X: spot.X - center.X, Y: 0, Z: spot.Z - center.Z}
				dir.Normalize2D()
				if dir.Dot2D(coneDir) < math.Cos(query.ConeAOI.Angle*0.5) {
					continue
				}

				chId, err := ctl.GetChannelId(spot)
				if err != nil {
					return nil, err
				}
				result[chId] = uint(math.Ceil(center.Dist2D(&spot) / ctl.GridSize()))
			}
		}
	}

	return result, nil
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

func (ctl *StaticGrid2DSpatialController) GetAdjacentChannels(spatialChannelId common.ChannelId) ([]common.ChannelId, error) {
	index := uint32(spatialChannelId - GlobalSettings.SpatialChannelIdStart)
	gridX := int32(index % ctl.GridCols)
	gridY := int32(index / ctl.GridCols)
	channelIds := make([]common.ChannelId, 0)
	for y := gridY - 1; y <= gridY+1; y++ {
		if y < 0 || y > int32(ctl.GridRows-1) {
			continue
		}

		for x := gridX - 1; x <= gridX+1; x++ {
			if x < 0 || x > int32(ctl.GridCols-1) {
				continue
			}
			if x == gridX && y == gridY {
				continue
			}

			channelIndex := uint32(x) + uint32(y)*ctl.GridCols
			channelIds = append(channelIds, common.ChannelId(channelIndex)+GlobalSettings.SpatialChannelIdStart)
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

	channelIds := make([]common.ChannelId, serverGridCols*serverGridRows)
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
func (ctl *StaticGrid2DSpatialController) Notify(oldInfo common.SpatialInfo, newInfo common.SpatialInfo, handoverDataProvider func(common.ChannelId, common.ChannelId, chan common.Message)) {
	srcChannelId, err := ctl.GetChannelId(oldInfo)
	if err != nil {
		rootLogger.Error("failed to calculate srcChannelId", zap.Error(err), zap.String("oldInfo", oldInfo.String()))
		return
	}
	dstChannelId, err := ctl.GetChannelId(newInfo)
	if err != nil {
		rootLogger.Error("failed to calculate dstChannelId", zap.Error(err), zap.String("newInfo", newInfo.String()))
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
	c := make(chan common.Message)
	go func() {
		handoverData := <-c
		if handoverData == nil {
			rootLogger.Info("handover will not happen as no data is provided", zap.Uint32("srcChannelId", uint32(srcChannelId)), zap.Uint32("dstChannelId", uint32(dstChannelId)))
			return
		}
		if rootLogger.Core().Enabled(zapcore.Level(VerboseLevel)) {
			rootLogger.Verbose("handover data", zap.Uint32("srcChannelId", uint32(srcChannelId)), zap.Uint32("dstChannelId", uint32(dstChannelId)),
				zap.String("data", dataMarshalOptions.Format(handoverData)))
		}

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

		// Use GetAllConnections() to avoid race condition
		srcChannelConns := srcChannel.GetAllConnections()
		dstChanenlConns := dstChannel.GetAllConnections()
		// Send the handover message to all connections in the srcChannel
		for conn := range srcChannelConns {
			// Avoid duplicate sending
			if _, exists := dstChanenlConns[conn]; !exists {
				conn.Send(MessageContext{
					MsgType:   channeldpb.MessageType_CHANNEL_DATA_HANDOVER,
					Msg:       handoverMsg,
					Broadcast: 0,
					StubId:    0,
					ChannelId: uint32(srcChannelId),
				})
			}
		}

		// Send the handover message to all connections in the dstChannel
		for conn := range dstChanenlConns {
			conn.Send(MessageContext{
				MsgType:   channeldpb.MessageType_CHANNEL_DATA_HANDOVER,
				Msg:       handoverMsg,
				Broadcast: 0,
				StubId:    0,
				ChannelId: uint32(dstChannelId),
			})
		}
	}()

	handoverDataProvider(srcChannelId, dstChannelId, c)
}

func (ctl *StaticGrid2DSpatialController) initServerConnections() {
	if ctl.serverConnections == nil {
		ctl.serverConnections = make([]ConnectionInChannel, ctl.ServerCols*ctl.ServerRows)
	}
}

func (ctl *StaticGrid2DSpatialController) nextServerIndex() uint32 {
	var i int = 0
	for i = 0; i < len(ctl.serverConnections); i++ {
		if ctl.serverConnections[i] == nil || ctl.serverConnections[i].IsClosing() {
			break
		}
	}
	return uint32(i)
}

func (ctl *StaticGrid2DSpatialController) Tick() {
	ctl.initServerConnections()
	for i := 0; i < len(ctl.serverConnections); i++ {
		if ctl.serverConnections[i] != nil && ctl.serverConnections[i].IsClosing() {
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
