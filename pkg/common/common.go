package common

import (
	"fmt"
	"math"

	"google.golang.org/protobuf/proto"
)

// Each channel uses a goroutine and we can have at most millions of goroutines at the same time.
// So we won't use 64-bit channel ID unless we use a distributed architecture for channeld itself.
type ChannelId uint32

type Message = proto.Message //protoreflect.ProtoMessage

type ChannelDataMessage = proto.Message //protoreflect.Message

// channeldpb.SpatialInfo is heavy with mutex lock and other allocations.
// We need a light struct for frequent value copy.
type SpatialInfo struct {
	X float64
	Y float64
	Z float64
}

// Notifies the spatial/entity channel from the ChannelDataUpdate that the spatial info of an entity has changed,
// so the channel's SpatialController will check the handover. If the handover happpens, `handoverDataProvider`
// will be called to provide the data for the ChannelDataHandoverMessage.
type SpatialInfoChangedNotifier interface {
	// The handover data provider has three parameters:
	// srcChannelId: the channel that an object is handed over from.
	// dstChannelId: the channel that an object is handed over to.
	// handoverData: the data wrapped in ChannelDataHandoverMessage to be sent to the interested parties. If the chan signals nil, no handover will happen.
	Notify(oldInfo SpatialInfo, newInfo SpatialInfo, handoverDataProvider func(ChannelId, ChannelId, chan Message))
}

func (s *SpatialInfo) String() string {
	return fmt.Sprintf("(%.4f, %.4f, %.4f)", s.X, s.Y, s.Z)
}

func (info1 *SpatialInfo) Dist2D(info2 *SpatialInfo) float64 {
	return math.Sqrt((info1.X-info2.X)*(info1.X-info2.X) + (info1.Z-info2.Z)*(info1.Z-info2.Z))
}

func (info1 *SpatialInfo) Dot2D(info2 *SpatialInfo) float64 {
	return info1.X*info2.X + info1.Z*info2.Z
}

func (info1 *SpatialInfo) Magnitude2D() float64 {
	return math.Sqrt(info1.X*info1.X + info1.Z*info1.Z)
}

func (info *SpatialInfo) Normalize2D() {
	mag := info.Magnitude2D()
	info.X /= mag
	info.Z /= mag
}

func (info *SpatialInfo) Unit2D() SpatialInfo {
	mag := info.Magnitude2D()
	return SpatialInfo{X: info.X / mag, Y: info.Y, Z: info.Z / mag}
}
