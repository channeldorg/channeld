package unrealpb

// Implement [channeld.HandoverDataWithPayload]
func (data *HandoverData) ClearPayload() {
	data.ChannelData = nil
}
