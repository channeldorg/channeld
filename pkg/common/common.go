package common

import (
	"fmt"

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
