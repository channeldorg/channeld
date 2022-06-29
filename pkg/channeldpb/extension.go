package channeldpb

func (broadcast BroadcastType) Check(value uint32) bool {
	return value&uint32(broadcast) > 0
}
