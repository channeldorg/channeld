package channeld

import "google.golang.org/protobuf/reflect/protoreflect"

type EntityId uint32

// type EntityChannel struct {
// 	*Channel
// }

// Implements [protoreflect.ProtoMessage] so the EntityId can be provided to the SpatialInfoChangedNotifier.
func (id EntityId) ProtoReflect() protoreflect.Message {
	return nil
}
