package main

import (
	"github.com/channeldorg/channeld/examples/channeld-ue-tps/tpspb"
	"github.com/channeldorg/channeld/pkg/channeld"
	"github.com/channeldorg/channeld/pkg/channeldpb"
	"github.com/channeldorg/channeld/pkg/unrealpb"
)

func InitTpsChannelDataTypes() {
	channeld.RegisterChannelDataType(channeldpb.ChannelType_GLOBAL, &tpspb.TestRepChannelData{})
	channeld.RegisterChannelDataType(channeldpb.ChannelType_SUBWORLD, &tpspb.TestRepChannelData{})
	channeld.RegisterChannelDataType(channeldpb.ChannelType_SPATIAL, &unrealpb.SpatialChannelData{})
	channeld.RegisterChannelDataType(channeldpb.ChannelType_ENTITY, &tpspb.EntityChannelData{})
}
