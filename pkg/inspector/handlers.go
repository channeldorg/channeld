package inspector

import (
	"encoding/json"

	"github.com/channeldorg/channeld/pkg/channeld"
	"github.com/channeldorg/channeld/pkg/channeldpb"
	"github.com/channeldorg/channeld/pkg/common"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

// HandleDebugGetChannels handles DEBUG_GET_CHANNELS request
func HandleDebugGetChannels(ctx channeld.MessageContext) {
	req, ok := ctx.Msg.(*channeldpb.DebugGetChannelsRequest)
	if !ok {
		ctx.Connection.Logger().Error("message is not DebugGetChannelsRequest")
		sendError(ctx, 400, "Invalid request message type")
		return
	}

	// Get all channels
	allChannels := GetAllChannels()

	// Apply filters
	var filteredChannels []*ChannelInfo
	for _, ch := range allChannels {
		// Filter by type
		if req.FilterType != channeldpb.ChannelType_UNKNOWN && ch.ChannelType != req.FilterType {
			continue
		}
		// TODO: Apply search keyword filter
		filteredChannels = append(filteredChannels, ch)
	}

	// Apply pagination
	total := uint32(len(filteredChannels))
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 50 // Default page size
	}
	if pageSize > 1000 {
		pageSize = 1000 // Max page size
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= total {
		start = total
		end = total
	}
	if end > total {
		end = total
	}

	var pagedChannels []*ChannelInfo
	if start < total {
		pagedChannels = filteredChannels[start:end]
	}

	// Convert to protobuf messages
	pbChannels := make([]*channeldpb.DebugChannelInfo, len(pagedChannels))
	for i, ch := range pagedChannels {
		pbChannels[i] = &channeldpb.DebugChannelInfo{
			ChannelId:       uint32(ch.ChannelID),
			ChannelType:     ch.ChannelType,
			OwnerConnId:     uint32(ch.OwnerConnID),
			SubscriberCount: uint32(ch.SubscriberCount),
			Metadata:        ch.Metadata,
			CreatedTime:     ch.CreatedTime,
		}
	}

	// Send response
	resp := &channeldpb.DebugGetChannelsResponse{
		Channels: pbChannels,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}

	sendResponse(ctx, channeldpb.MessageType_DEBUG_GET_CHANNELS_RESULT, resp)
}

// HandleDebugGetChannel handles DEBUG_GET_CHANNEL request
func HandleDebugGetChannel(ctx channeld.MessageContext) {
	req, ok := ctx.Msg.(*channeldpb.DebugGetChannelRequest)
	if !ok {
		ctx.Connection.Logger().Error("message is not DebugGetChannelRequest")
		sendError(ctx, 400, "Invalid request message type")
		return
	}

	ch := channeld.GetChannel(common.ChannelId(req.ChannelId))
	if ch == nil {
		sendError(ctx, 404, "Channel not found")
		return
	}

	info := GetChannelInfo(common.ChannelId(req.ChannelId))
	if info == nil {
		sendError(ctx, 500, "Failed to get channel info")
		return
	}

	// Get subscriber connection IDs
	allConns := ch.GetAllConnections()
	subscriberIds := make([]uint32, 0, len(allConns))
	for conn := range allConns {
		subscriberIds = append(subscriberIds, uint32(conn.Id()))
	}

	pbInfo := &channeldpb.DebugChannelInfo{
		ChannelId:       uint32(info.ChannelID),
		ChannelType:     info.ChannelType,
		OwnerConnId:     uint32(info.OwnerConnID),
		SubscriberCount: uint32(info.SubscriberCount),
		Metadata:        info.Metadata,
		CreatedTime:     info.CreatedTime,
	}

	resp := &channeldpb.DebugGetChannelResponse{
		Channel:           pbInfo,
		SubscriberConnIds: subscriberIds,
	}

	sendResponse(ctx, channeldpb.MessageType_DEBUG_GET_CHANNEL_RESULT, resp)
}

// HandleDebugGetChannelData handles DEBUG_GET_CHANNEL_DATA request
func HandleDebugGetChannelData(ctx channeld.MessageContext) {
	req, ok := ctx.Msg.(*channeldpb.DebugGetChannelDataRequest)
	if !ok {
		ctx.Connection.Logger().Error("message is not DebugGetChannelDataRequest")
		sendError(ctx, 400, "Invalid request message type")
		return
	}

	ch := channeld.GetChannel(common.ChannelId(req.ChannelId))
	if ch == nil {
		sendError(ctx, 404, "Channel not found")
		return
	}

	// Access channel data in the channel's goroutine for thread safety
	var jsonData string
	var hasData bool
	var err error

	ch.Execute(func(ch *channeld.Channel) {
		jsonData, hasData, err = getChannelDataJSON(ch)
	})

	if err != nil {
		ctx.Connection.Logger().Error("failed to get channel data", zap.Error(err))
		sendError(ctx, 500, "Failed to get channel data: "+err.Error())
		return
	}

	resp := &channeldpb.DebugGetChannelDataResponse{
		ChannelId: req.ChannelId,
		JsonData:  jsonData,
		HasData:   hasData,
	}

	sendResponse(ctx, channeldpb.MessageType_DEBUG_GET_CHANNEL_DATA_RESULT, resp)
}

// HandleDebugUpdateChannelData handles DEBUG_UPDATE_CHANNEL_DATA request
func HandleDebugUpdateChannelData(ctx channeld.MessageContext) {
	req, ok := ctx.Msg.(*channeldpb.DebugUpdateChannelDataRequest)
	if !ok {
		ctx.Connection.Logger().Error("message is not DebugUpdateChannelDataRequest")
		sendError(ctx, 400, "Invalid request message type")
		return
	}

	ch := channeld.GetChannel(common.ChannelId(req.ChannelId))
	if ch == nil {
		sendError(ctx, 404, "Channel not found")
		return
	}

	// Update channel data in the channel's goroutine for thread safety
	var updateErr error
	ch.Execute(func(ch *channeld.Channel) {
		updateErr = updateChannelDataByPath(ch, req.JsonPath, req.Value, req.ValueType, ctx)
	})

	if updateErr != nil {
		ctx.Connection.Logger().Error("failed to update channel data", zap.Error(updateErr))
		resp := &channeldpb.DebugUpdateChannelDataResponse{
			ChannelId:    req.ChannelId,
			Success:      false,
			ErrorMessage: updateErr.Error(),
		}
		sendResponse(ctx, channeldpb.MessageType_DEBUG_UPDATE_CHANNEL_DATA_RESULT, resp)
		return
	}

	resp := &channeldpb.DebugUpdateChannelDataResponse{
		ChannelId: req.ChannelId,
		Success:   true,
	}

	sendResponse(ctx, channeldpb.MessageType_DEBUG_UPDATE_CHANNEL_DATA_RESULT, resp)
}

// updateChannelDataByPath updates channel data by JSON path
func updateChannelDataByPath(ch *channeld.Channel, jsonPath string, value string, valueType string, ctx channeld.MessageContext) error {
	data := ch.Data()
	if data == nil {
		return ErrChannelHasNoData
	}

	msg := ch.GetDataMessage()
	if msg == nil {
		return ErrChannelHasNoData
	}

	// Convert protobuf to JSON
	jsonBytes, err := protojson.Marshal(msg)
	if err != nil {
		return err
	}

	// Parse JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &jsonData); err != nil {
		return err
	}

	// Update value at path
	if err := updateJSONPath(jsonData, jsonPath, value, valueType); err != nil {
		return err
	}

	// Convert back to JSON
	updatedJSONBytes, err := json.Marshal(jsonData)
	if err != nil {
		return err
	}

	// Unmarshal back to protobuf message
	// msg is already a common.ChannelDataMessage, so New() will return the same type
	newMsg := msg.ProtoReflect().New().Interface()
	if err := protojson.Unmarshal(updatedJSONBytes, newMsg); err != nil {
		return err
	}

	// Update the channel data by calling OnUpdate
	// This will properly merge the data and trigger fan-out
	channelTime := ch.GetTime()
	senderConnId := ctx.Connection.Id()

	// Get spatial notifier if available (for spatial channels)
	// Note: spatialNotifier is stored in channel but not exposed as public method
	// For now, pass nil - the merge logic should handle it
	// TODO: Add public method to Channel to get spatialNotifier if needed
	var spatialNotifier common.SpatialInfoChangedNotifier = nil

	// newMsg is already common.ChannelDataMessage type (from msg.ProtoReflect().New())
	data.OnUpdate(newMsg, channelTime, senderConnId, spatialNotifier)

	return nil
}

// updateJSONPath updates a value in JSON object at the given path
func updateJSONPath(data map[string]interface{}, path string, value string, valueType string) error {
	// Simple implementation for top-level properties
	// TODO: Support nested paths like "field.subfield" and array indices

	var parsedValue interface{}
	switch valueType {
	case "string":
		parsedValue = value
	case "number":
		var num float64
		if err := json.Unmarshal([]byte(value), &num); err != nil {
			return err
		}
		parsedValue = num
	case "boolean":
		var b bool
		if err := json.Unmarshal([]byte(value), &b); err != nil {
			return err
		}
		parsedValue = b
	case "null":
		parsedValue = nil
	default:
		// Try to parse as JSON
		if err := json.Unmarshal([]byte(value), &parsedValue); err != nil {
			return err
		}
	}

	data[path] = parsedValue
	return nil
}

// sendResponse sends a response message
// The StubId from the original request is preserved so the client can match the response
func sendResponse(ctx channeld.MessageContext, msgType channeldpb.MessageType, msg common.Message) {
	ctx.MsgType = msgType
	ctx.Msg = msg
	ctx.ChannelId = 0 // Inspector messages use GLOBAL channel
	// StubId is preserved from the original request context
	ctx.Connection.Send(ctx)
}

// sendError sends an error response
func sendError(ctx channeld.MessageContext, code uint32, message string) {
	errResp := &channeldpb.DebugErrorResponse{
		Code:    code,
		Message: message,
	}
	sendResponse(ctx, channeldpb.MessageType_DEBUG_ERROR, errResp)
}

// getChannelDataJSON returns channel data as JSON string
// Must be called in the channel's goroutine for thread safety
func getChannelDataJSON(ch *channeld.Channel) (string, bool, error) {
	if ch == nil {
		return "", false, nil
	}

	data := ch.Data()
	if data == nil {
		return "", false, nil
	}

	msg := ch.GetDataMessage()
	if msg == nil {
		return "", false, nil
	}

	// Convert protobuf message to JSON
	jsonBytes, err := protojson.Marshal(msg)
	if err != nil {
		return "", true, err
	}

	return string(jsonBytes), true, nil
}
