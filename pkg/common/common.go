package common

import "google.golang.org/protobuf/proto"

type ChannelDataMessage = proto.Message //protoreflect.Message

// channeldpb.SpatialInfo is heavy with mutex lock and other allocations.
// We need a light struct for frequent value copy.
type SpatialInfo struct {
	X float64
	Y float64
	Z float64
}

type SpatialInfoChangedNotifier interface {
	Notify(oldInfo SpatialInfo, newInfo SpatialInfo, handoverDataProvider func() ChannelDataMessage)
}
