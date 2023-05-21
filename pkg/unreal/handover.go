package unreal

import (
	"github.com/metaworking/channeld/pkg/common"
	"github.com/metaworking/channeld/pkg/unrealpb"
)

func CheckEntityHandover(netId uint32, newLoc, oldLoc *unrealpb.FVector) (bool, *common.SpatialInfo, *common.SpatialInfo) { //, spatialNotifier common.SpatialInfoChangedNotifier) {
	var newX, newY, newZ float32

	if newLoc.X != nil {
		newX = *newLoc.X
	} else {
		// Use GetX/Y() to avoid violation memory access!
		newX = oldLoc.GetX()
	}

	if newLoc.Y != nil {
		newY = *newLoc.Y
	} else {
		newY = oldLoc.GetY()
	}

	if newLoc.Z != nil {
		newZ = *newLoc.Z
	} else {
		newZ = oldLoc.GetZ()
	}

	if newX != oldLoc.GetX() || newY != oldLoc.GetY() || newZ != oldLoc.GetZ() {
		// Swap the Y and Z as UE uses the Z-Up rule but channeld uses the Y-up rule.
		oldInfo := &common.SpatialInfo{
			X: float64(oldLoc.GetX()),
			Y: float64(oldLoc.GetZ()),
			Z: float64(oldLoc.GetY()),
		}
		newInfo := &common.SpatialInfo{
			X: float64(newX),
			Y: float64(newZ),
			Z: float64(newY),
		}

		return true, oldInfo, newInfo
	}

	return false, nil, nil
}
