package channeld

import (
	"errors"
	"fmt"
	"math"

	"channeld.clewcat.com/channeld/pkg/channeldpb"
	"channeld.clewcat.com/channeld/pkg/common"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/anypb"
)

type SpatialController interface {
	common.SpatialInfoChangedNotifier
	GetChannelId(info common.SpatialInfo) (ChannelId, error)
	CreateChannels(ctx MessageContext) ([]*Channel, error)
}

var spatialController SpatialController

func InitSpatialController(controller SpatialController) {
	spatialController = controller
}

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

	serverIndex       uint32
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

func (ctl *StaticGrid2DSpatialController) CreateChannels(ctx MessageContext) ([]*Channel, error) {
	if ctl.serverIndex >= ctl.ServerCols*ctl.ServerRows {
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
	serverX := ctl.serverIndex % ctl.ServerCols
	serverY := ctl.serverIndex / ctl.ServerCols
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

	if ctl.serverConnections == nil {
		ctl.serverConnections = make([]ConnectionInChannel, ctl.ServerCols*ctl.ServerRows)
	}
	// Save the connection for later use
	ctl.serverConnections[ctl.serverIndex] = ctx.Connection
	ctl.serverIndex++
	// When all spatial channels are created, subscribe each server to its adjacent grids(channels)
	if ctl.serverIndex == ctl.ServerCols*ctl.ServerRows {
		for i := uint32(0); i < ctl.serverIndex; i++ {
			ctl.subToAdjacentChannels(i, serverGridCols, serverGridRows, msg.SubOptions)
		}
		/*
			// right border
			if serverX > 0 {
				for y := uint32(0); y < serverGridRows; y++ {
					for x := uint32(0); x < ctl.ServerInterestBorderSize; x++ {
						spatialInfo.X = float64(serverX*serverGridCols-x) * ctl.GridWidth
						spatialInfo.Z = float64(serverY*serverGridRows+y) * ctl.GridHeight
						channelId, err := ctl.GetChannelIdNoOffset(spatialInfo)
						if err != nil {
							return nil, err
						}
						channelToSub := GetChannel(channelId)
						if channelToSub == nil {
							return nil, fmt.Errorf("failed to subscribe border channel %d as it doesn't exist", channelId)
						}
						ctx.Connection.SubscribeToChannel(channelToSub, msg.SubOptions)
						ctx.Connection.sendSubscribed(ctx, channelToSub, ctx.Connection, 0, msg.SubOptions)
					}
				}
			}

			// left border
			if serverX < ctl.ServerCols-1 {
				for y := uint32(0); y < serverGridRows; y++ {
					for x := uint32(0); x < ctl.ServerInterestBorderSize; x++ {
						spatialInfo.X = float64((serverX+1)*serverGridCols+x) * ctl.GridWidth
						spatialInfo.Z = float64(serverY*serverGridRows+y) * ctl.GridHeight
						channelId, err := ctl.GetChannelIdNoOffset(spatialInfo)
						if err != nil {
							return nil, err
						}
						channelToSub := GetChannel(channelId)
						if channelToSub == nil {
							return nil, fmt.Errorf("failed to subscribe border channel %d as it doesn't exist", channelId)
						}
						ctx.Connection.SubscribeToChannel(channelToSub, msg.SubOptions)
						ctx.Connection.sendSubscribed(ctx, channelToSub, ctx.Connection, 0, msg.SubOptions)
					}
				}
			}

			// top border
			if serverY > 0 {
				for y := uint32(0); y < ctl.ServerInterestBorderSize; y++ {
					for x := uint32(0); x < serverGridCols; x++ {
						spatialInfo.X = float64(serverX*serverGridCols+x) * ctl.GridWidth
						spatialInfo.Z = float64(serverY*serverGridRows-y) * ctl.GridHeight
						channelId, err := ctl.GetChannelIdNoOffset(spatialInfo)
						if err != nil {
							return nil, err
						}
						channelToSub := GetChannel(channelId)
						if channelToSub == nil {
							return nil, fmt.Errorf("failed to subscribe border channel %d as it doesn't exist", channelId)
						}
						ctx.Connection.SubscribeToChannel(channelToSub, msg.SubOptions)
						ctx.Connection.sendSubscribed(ctx, channelToSub, ctx.Connection, 0, msg.SubOptions)
					}
				}
			}

			// bottom border
			if serverY > 0 {
				for y := uint32(0); y < ctl.ServerInterestBorderSize; y++ {
					for x := uint32(0); x < serverGridCols; x++ {
						spatialInfo.X = float64(serverX*serverGridCols+x) * ctl.GridWidth
						spatialInfo.Z = float64((serverY+1)*serverGridRows+y) * ctl.GridHeight
						channelId, err := ctl.GetChannelIdNoOffset(spatialInfo)
						if err != nil {
							return nil, err
						}
						channelToSub := GetChannel(channelId)
						if channelToSub == nil {
							return nil, fmt.Errorf("failed to subscribe border channel %d as it doesn't exist", channelId)
						}
						ctx.Connection.SubscribeToChannel(channelToSub, msg.SubOptions)
					}
				}
			}
		*/
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
	serverChannelId, err := ctl.GetChannelId(spatialInfo)
	if err != nil {
		return err
	}
	serverChannel := GetChannel(serverChannelId)
	if serverChannel == nil {
		return fmt.Errorf("failed to subscribe to adjacent channels for  %d as it doesn't exist", serverChannelId)
	}

	// right border
	if serverX > 0 {
		for y := uint32(0); y < serverGridRows; y++ {
			for x := uint32(0); x < ctl.ServerInterestBorderSize; x++ {
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
				serverConn.SubscribeToChannel(channelToSub, subOptions)
				serverConn.sendSubscribed(MessageContext{}, channelToSub, serverConn, 0, subOptions)
			}
		}
	}

	// left border
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
				serverConn.SubscribeToChannel(channelToSub, subOptions)
				serverConn.sendSubscribed(MessageContext{}, channelToSub, serverConn, 0, subOptions)
			}
		}
	}

	// top border
	if serverY > 0 {
		for y := uint32(0); y < ctl.ServerInterestBorderSize; y++ {
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
				serverConn.SubscribeToChannel(channelToSub, subOptions)
				serverConn.sendSubscribed(MessageContext{}, channelToSub, serverConn, 0, subOptions)
			}
		}
	}

	// bottom border
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
				serverConn.SubscribeToChannel(channelToSub, subOptions)
				serverConn.sendSubscribed(MessageContext{}, channelToSub, serverConn, 0, subOptions)
			}
		}
	}
	return nil
}

func (ctl *StaticGrid2DSpatialController) Notify(oldInfo common.SpatialInfo, newInfo common.SpatialInfo, handoverDataProvider func() common.ChannelDataMessage) {
	oldChannelId, err := ctl.GetChannelId(oldInfo)
	if err != nil {
		logger.Error("failed to calculate oldChannelId", zap.Error(err))
		return
	}
	newChannelId, err := ctl.GetChannelId(newInfo)
	if err != nil {
		logger.Error("failed to calculate newChannelId", zap.Error(err))
		return
	}
	if newChannelId == oldChannelId {
		return
	}

	oldChannel := GetChannel(oldChannelId)
	if oldChannel == nil {
		logger.Error("channel doesn't exist, failed to handover channel data", zap.Uint32("oldChannelId", uint32(oldChannelId)))
		return
	}
	if !oldChannel.HasOwner() {
		logger.Error("channel doesn't have owner, failed to handover channel data", zap.Uint32("oldChannelId", uint32(oldChannelId)))
	}

	newChannel := GetChannel(newChannelId)
	if newChannel == nil {
		logger.Error("channel doesn't exist, failed to handover channel data", zap.Uint32("newChannelId", uint32(newChannelId)))
		return
	}
	if !oldChannel.HasOwner() {
		logger.Error("channel doesn't have owner, failed to handover channel data", zap.Uint32("newChannelId", uint32(newChannelId)))
	}

	handoverData := handoverDataProvider()
	if handoverData == nil {
		logger.Error("failed to provider handover channel data", zap.Uint32("oldChannelId", uint32(oldChannelId)), zap.Uint32("newChannelId", uint32(newChannelId)))
		return
	}

	anyData, err := anypb.New(handoverData)
	if err != nil {
		logger.Error("failed to marshall handover data", zap.Error(err))
		return
	}

	/*
		newChannel.PutMessage(&channeldpb.ChannelDataUpdateMessage{
			Data: anyData,
		}, handleChannelDataUpdate, internalDummyConnection, &channeldpb.MessagePack{
			MsgType:   uint32(channeldpb.MessageType_CHANNEL_DATA_UPDATE),
			Broadcast: channeldpb.BroadcastType_NO_BROADCAST,
			StubId:    0,
			ChannelId: uint32(newChannelId),
		})
	*/

	handoverMsg := &channeldpb.ChannelDataHandoverMessage{
		SrcChannelId: uint32(oldChannelId),
		DstChannelId: uint32(newChannelId),
		Data:         anyData,
	}
	oldChannel.ownerConnection.Send(MessageContext{
		MsgType:   channeldpb.MessageType_CHANNEL_DATA_HANDOVER,
		Msg:       handoverMsg,
		Broadcast: channeldpb.BroadcastType_NO_BROADCAST,
		StubId:    0,
		ChannelId: uint32(oldChannelId),
	})
	newChannel.ownerConnection.Send(MessageContext{
		MsgType:   channeldpb.MessageType_CHANNEL_DATA_HANDOVER,
		Msg:       handoverMsg,
		Broadcast: channeldpb.BroadcastType_NO_BROADCAST,
		StubId:    0,
		ChannelId: uint32(newChannelId),
	})
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
