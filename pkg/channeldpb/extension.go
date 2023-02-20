package channeldpb

import "math"

func (broadcast BroadcastType) Check(value uint32) bool {
	return value&uint32(broadcast) > 0
}

func (info1 *SpatialInfo) Dist2D(info2 *SpatialInfo) float64 {
	return math.Sqrt((info1.X-info2.X)*(info1.X-info2.X) + (info1.Z-info2.Z)*(info1.Z-info2.Z))
}
