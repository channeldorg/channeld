package common

import "google.golang.org/protobuf/proto"

type ChannelDataMessage = proto.Message //protoreflect.Message

type SpatialInfo struct {
	X float64
	Y float64
	Z float64
}

type SpatialInfoChangedNotifier interface {
	Notify(oldInfo SpatialInfo, newInfo SpatialInfo, handoverDataProvider func() ChannelDataMessage)
	//IsNotified() bool
}
