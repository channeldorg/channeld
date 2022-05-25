// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.18.0
// source: unity_common.proto

package channeldpb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Vector3F struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	X float32 `protobuf:"fixed32,1,opt,name=x,proto3" json:"x,omitempty"`
	Y float32 `protobuf:"fixed32,2,opt,name=y,proto3" json:"y,omitempty"`
	Z float32 `protobuf:"fixed32,3,opt,name=z,proto3" json:"z,omitempty"`
}

func (x *Vector3F) Reset() {
	*x = Vector3F{}
	if protoimpl.UnsafeEnabled {
		mi := &file_unity_common_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Vector3F) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Vector3F) ProtoMessage() {}

func (x *Vector3F) ProtoReflect() protoreflect.Message {
	mi := &file_unity_common_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Vector3F.ProtoReflect.Descriptor instead.
func (*Vector3F) Descriptor() ([]byte, []int) {
	return file_unity_common_proto_rawDescGZIP(), []int{0}
}

func (x *Vector3F) GetX() float32 {
	if x != nil {
		return x.X
	}
	return 0
}

func (x *Vector3F) GetY() float32 {
	if x != nil {
		return x.Y
	}
	return 0
}

func (x *Vector3F) GetZ() float32 {
	if x != nil {
		return x.Z
	}
	return 0
}

type Vector4F struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	X float32 `protobuf:"fixed32,1,opt,name=x,proto3" json:"x,omitempty"`
	Y float32 `protobuf:"fixed32,2,opt,name=y,proto3" json:"y,omitempty"`
	Z float32 `protobuf:"fixed32,3,opt,name=z,proto3" json:"z,omitempty"`
	W float32 `protobuf:"fixed32,4,opt,name=w,proto3" json:"w,omitempty"`
}

func (x *Vector4F) Reset() {
	*x = Vector4F{}
	if protoimpl.UnsafeEnabled {
		mi := &file_unity_common_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Vector4F) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Vector4F) ProtoMessage() {}

func (x *Vector4F) ProtoReflect() protoreflect.Message {
	mi := &file_unity_common_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Vector4F.ProtoReflect.Descriptor instead.
func (*Vector4F) Descriptor() ([]byte, []int) {
	return file_unity_common_proto_rawDescGZIP(), []int{1}
}

func (x *Vector4F) GetX() float32 {
	if x != nil {
		return x.X
	}
	return 0
}

func (x *Vector4F) GetY() float32 {
	if x != nil {
		return x.Y
	}
	return 0
}

func (x *Vector4F) GetZ() float32 {
	if x != nil {
		return x.Z
	}
	return 0
}

func (x *Vector4F) GetW() float32 {
	if x != nil {
		return x.W
	}
	return 0
}

type TransformState struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Marks that the state should be removed from the containing map
	Removed  bool      `protobuf:"varint,1,opt,name=removed,proto3" json:"removed,omitempty"`
	Position *Vector3F `protobuf:"bytes,2,opt,name=position,proto3" json:"position,omitempty"`
	Rotation *Vector4F `protobuf:"bytes,3,opt,name=rotation,proto3" json:"rotation,omitempty"`
	Scale    *Vector3F `protobuf:"bytes,4,opt,name=scale,proto3" json:"scale,omitempty"`
}

func (x *TransformState) Reset() {
	*x = TransformState{}
	if protoimpl.UnsafeEnabled {
		mi := &file_unity_common_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TransformState) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TransformState) ProtoMessage() {}

func (x *TransformState) ProtoReflect() protoreflect.Message {
	mi := &file_unity_common_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TransformState.ProtoReflect.Descriptor instead.
func (*TransformState) Descriptor() ([]byte, []int) {
	return file_unity_common_proto_rawDescGZIP(), []int{2}
}

func (x *TransformState) GetRemoved() bool {
	if x != nil {
		return x.Removed
	}
	return false
}

func (x *TransformState) GetPosition() *Vector3F {
	if x != nil {
		return x.Position
	}
	return nil
}

func (x *TransformState) GetRotation() *Vector4F {
	if x != nil {
		return x.Rotation
	}
	return nil
}

func (x *TransformState) GetScale() *Vector3F {
	if x != nil {
		return x.Scale
	}
	return nil
}

var File_unity_common_proto protoreflect.FileDescriptor

var file_unity_common_proto_rawDesc = []byte{
	0x0a, 0x12, 0x75, 0x6e, 0x69, 0x74, 0x79, 0x5f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x63, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x64, 0x70, 0x62,
	0x22, 0x34, 0x0a, 0x08, 0x56, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x33, 0x66, 0x12, 0x0c, 0x0a, 0x01,
	0x78, 0x18, 0x01, 0x20, 0x01, 0x28, 0x02, 0x52, 0x01, 0x78, 0x12, 0x0c, 0x0a, 0x01, 0x79, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x02, 0x52, 0x01, 0x79, 0x12, 0x0c, 0x0a, 0x01, 0x7a, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x02, 0x52, 0x01, 0x7a, 0x22, 0x42, 0x0a, 0x08, 0x56, 0x65, 0x63, 0x74, 0x6f, 0x72,
	0x34, 0x66, 0x12, 0x0c, 0x0a, 0x01, 0x78, 0x18, 0x01, 0x20, 0x01, 0x28, 0x02, 0x52, 0x01, 0x78,
	0x12, 0x0c, 0x0a, 0x01, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x02, 0x52, 0x01, 0x79, 0x12, 0x0c,
	0x0a, 0x01, 0x7a, 0x18, 0x03, 0x20, 0x01, 0x28, 0x02, 0x52, 0x01, 0x7a, 0x12, 0x0c, 0x0a, 0x01,
	0x77, 0x18, 0x04, 0x20, 0x01, 0x28, 0x02, 0x52, 0x01, 0x77, 0x22, 0xba, 0x01, 0x0a, 0x0e, 0x54,
	0x72, 0x61, 0x6e, 0x73, 0x66, 0x6f, 0x72, 0x6d, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x18, 0x0a,
	0x07, 0x72, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07,
	0x72, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x64, 0x12, 0x30, 0x0a, 0x08, 0x70, 0x6f, 0x73, 0x69, 0x74,
	0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x63, 0x68, 0x61, 0x6e,
	0x6e, 0x65, 0x6c, 0x64, 0x70, 0x62, 0x2e, 0x56, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x33, 0x66, 0x52,
	0x08, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x30, 0x0a, 0x08, 0x72, 0x6f, 0x74,
	0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x63, 0x68,
	0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x64, 0x70, 0x62, 0x2e, 0x56, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x34,
	0x66, 0x52, 0x08, 0x72, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x2a, 0x0a, 0x05, 0x73,
	0x63, 0x61, 0x6c, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x63, 0x68, 0x61,
	0x6e, 0x6e, 0x65, 0x6c, 0x64, 0x70, 0x62, 0x2e, 0x56, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x33, 0x66,
	0x52, 0x05, 0x73, 0x63, 0x61, 0x6c, 0x65, 0x42, 0x39, 0x5a, 0x2c, 0x63, 0x68, 0x61, 0x6e, 0x6e,
	0x65, 0x6c, 0x64, 0x2e, 0x63, 0x6c, 0x65, 0x77, 0x63, 0x61, 0x74, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x63, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x64, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x63, 0x68, 0x61,
	0x6e, 0x6e, 0x65, 0x6c, 0x64, 0x70, 0x62, 0xaa, 0x02, 0x08, 0x43, 0x68, 0x61, 0x6e, 0x6e, 0x65,
	0x6c, 0x64, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_unity_common_proto_rawDescOnce sync.Once
	file_unity_common_proto_rawDescData = file_unity_common_proto_rawDesc
)

func file_unity_common_proto_rawDescGZIP() []byte {
	file_unity_common_proto_rawDescOnce.Do(func() {
		file_unity_common_proto_rawDescData = protoimpl.X.CompressGZIP(file_unity_common_proto_rawDescData)
	})
	return file_unity_common_proto_rawDescData
}

var file_unity_common_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_unity_common_proto_goTypes = []interface{}{
	(*Vector3F)(nil),       // 0: channeldpb.Vector3f
	(*Vector4F)(nil),       // 1: channeldpb.Vector4f
	(*TransformState)(nil), // 2: channeldpb.TransformState
}
var file_unity_common_proto_depIdxs = []int32{
	0, // 0: channeldpb.TransformState.position:type_name -> channeldpb.Vector3f
	1, // 1: channeldpb.TransformState.rotation:type_name -> channeldpb.Vector4f
	0, // 2: channeldpb.TransformState.scale:type_name -> channeldpb.Vector3f
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_unity_common_proto_init() }
func file_unity_common_proto_init() {
	if File_unity_common_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_unity_common_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Vector3F); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_unity_common_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Vector4F); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_unity_common_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TransformState); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_unity_common_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_unity_common_proto_goTypes,
		DependencyIndexes: file_unity_common_proto_depIdxs,
		MessageInfos:      file_unity_common_proto_msgTypes,
	}.Build()
	File_unity_common_proto = out.File
	file_unity_common_proto_rawDesc = nil
	file_unity_common_proto_goTypes = nil
	file_unity_common_proto_depIdxs = nil
}