package channeld

import (
	"fmt"
	"math"
)

type SpatialInfo struct {
	X float64
	Y float64
	Z float64
}

type SpatialController interface {
	GetChannelId(info SpatialInfo) ChannelId
}

type SpatialInfoChangedNotifier interface {
	Notify(newInfo SpatialInfo)
	//IsNotified() bool
}

// Divide the world into GridCols x GridRows static squares on the XZ plane. Each square(grid) represents a spatial channel.
// Typically, a player's view distance is 150m, so a grid is sized at 50x50m.
// A 100x100 km world has 2000x2000 grids, which needs about 2^22 spatial channels.
// By default, we support up to 2^32-2^16 grid-based spatial channels.
type StaticGrid2DSpatialController struct {
	// WorldWidth  float64
	// WorldHeight float64
	WorldOffsetX float64
	WorldOffsetZ float64
	GridWidth    float64
	GridHeight   float64
	GridCols     uint32
	GridRows     uint32

	channel *Channel
}

func (controller *StaticGrid2DSpatialController) GetChannelId(info SpatialInfo) (ChannelId, error) {
	gridX := int(math.Ceil((info.X - controller.WorldOffsetX) / controller.GridWidth))
	if gridX < 0 {
		return 0, fmt.Errorf("gridX=%d when X=%f. GridX should never be negative", gridX, info.X)
	}
	gridY := int(math.Ceil((info.Y - controller.WorldOffsetZ) / controller.GridHeight))
	if gridY < 0 {
		return 0, fmt.Errorf("gridY=%d when Z=%f. GridY should never be negative", gridY, info.Z)
	}
	index := uint32(gridX) + uint32(gridY)*controller.GridCols
	return ChannelId(index) + GlobalSettings.SpatialChannelIdStart, nil
}

func (controller *StaticGrid2DSpatialController) Notify(newInfo SpatialInfo) {

}
