
package main

import (
	"channeld.clewcat.com/channeld/examples/channeld-ue-tps/channeldgenpb"
	"channeld.clewcat.com/channeld/pkg/channeld"
	"channeld.clewcat.com/channeld/pkg/channeldpb"
)

func InitChannelDataTypes() {
	channeld.RegisterChannelDataType(channeldpb.ChannelType_GLOBAL, &channeldgenpb.DefaultChannelData{})
}
