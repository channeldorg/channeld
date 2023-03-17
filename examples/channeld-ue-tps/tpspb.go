package main

import (
	"github.com/metaworking/channeld/examples/channeld-ue-tps/tpspb"
	"github.com/metaworking/channeld/pkg/channeld"
	"github.com/metaworking/channeld/pkg/channeldpb"
)

func InitTpsChannelDataTypes() {
	channeld.RegisterChannelDataType(channeldpb.ChannelType_GLOBAL, &tpspb.TestRepChannelData{})
	channeld.RegisterChannelDataType(channeldpb.ChannelType_SUBWORLD, &tpspb.TestRepChannelData{})
	channeld.RegisterChannelDataType(channeldpb.ChannelType_SPATIAL, &tpspb.TestRepChannelData{})
}
