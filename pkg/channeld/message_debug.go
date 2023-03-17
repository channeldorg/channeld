package channeld

import (
	"github.com/metaworking/channeld/pkg/channeldpb"
	"go.uber.org/zap"
)

func handleGetSpatialRegionsMessage(ctx MessageContext) {
	if !GlobalSettings.Development {
		ctx.Connection.Logger().Error("DebugGetSpatialRegionsMessage can only be handled in Development mode")
		return
	}

	if ctx.Channel != globalChannel {
		ctx.Connection.Logger().Error("illegal attemp to retrieve spatial regions outside the GLOBAL channel")
		return
	}

	_, ok := ctx.Msg.(*channeldpb.DebugGetSpatialRegionsMessage)
	if !ok {
		ctx.Connection.Logger().Error("mssage is not a DebugGetSpatialRegionsMessage, will not be handled.")
		return
	}

	if spatialController == nil {
		ctx.Connection.Logger().Error("illegal attemp to retrieve spatial regions as there's no controller")
		return
	}

	regions, err := spatialController.GetRegions()
	if err != nil {
		ctx.Connection.Logger().Error("failed to retrieve spatial regions", zap.Error(err))
	}
	ctx.MsgType = channeldpb.MessageType_SPATIAL_REGIONS_UPDATE
	ctx.Msg = &channeldpb.SpatialRegionsUpdateMessage{
		Regions: regions,
	}
	ctx.Connection.Send(ctx)
}
