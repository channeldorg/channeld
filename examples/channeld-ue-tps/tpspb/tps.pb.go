// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.0
// 	protoc        v3.20.1
// source: tps.proto

package tpspb

import (
	unrealpb "channeld.clewcat.com/channeld/pkg/unrealpb"
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

type TestRepChannelData struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	SceneComponentStates map[uint32]*unrealpb.SceneComponentState `protobuf:"bytes,1,rep,name=sceneComponentStates,proto3" json:"sceneComponentStates,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *TestRepChannelData) Reset() {
	*x = TestRepChannelData{}
	if protoimpl.UnsafeEnabled {
		mi := &file_tps_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TestRepChannelData) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestRepChannelData) ProtoMessage() {}

func (x *TestRepChannelData) ProtoReflect() protoreflect.Message {
	mi := &file_tps_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestRepChannelData.ProtoReflect.Descriptor instead.
func (*TestRepChannelData) Descriptor() ([]byte, []int) {
	return file_tps_proto_rawDescGZIP(), []int{0}
}

func (x *TestRepChannelData) GetSceneComponentStates() map[uint32]*unrealpb.SceneComponentState {
	if x != nil {
		return x.SceneComponentStates
	}
	return nil
}

var File_tps_proto protoreflect.FileDescriptor

var file_tps_proto_rawDesc = []byte{
	0x0a, 0x09, 0x74, 0x70, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x05, 0x74, 0x70, 0x73,
	0x70, 0x62, 0x1a, 0x1c, 0x75, 0x6e, 0x72, 0x65, 0x61, 0x6c, 0x70, 0x62, 0x2f, 0x75, 0x6e, 0x72,
	0x65, 0x61, 0x6c, 0x5f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x22, 0xe5, 0x01, 0x0a, 0x12, 0x54, 0x65, 0x73, 0x74, 0x52, 0x65, 0x70, 0x43, 0x68, 0x61, 0x6e,
	0x6e, 0x65, 0x6c, 0x44, 0x61, 0x74, 0x61, 0x12, 0x67, 0x0a, 0x14, 0x73, 0x63, 0x65, 0x6e, 0x65,
	0x43, 0x6f, 0x6d, 0x70, 0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x53, 0x74, 0x61, 0x74, 0x65, 0x73, 0x18,
	0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x33, 0x2e, 0x74, 0x70, 0x73, 0x70, 0x62, 0x2e, 0x54, 0x65,
	0x73, 0x74, 0x52, 0x65, 0x70, 0x43, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x44, 0x61, 0x74, 0x61,
	0x2e, 0x53, 0x63, 0x65, 0x6e, 0x65, 0x43, 0x6f, 0x6d, 0x70, 0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x53,
	0x74, 0x61, 0x74, 0x65, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x14, 0x73, 0x63, 0x65, 0x6e,
	0x65, 0x43, 0x6f, 0x6d, 0x70, 0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x53, 0x74, 0x61, 0x74, 0x65, 0x73,
	0x1a, 0x66, 0x0a, 0x19, 0x53, 0x63, 0x65, 0x6e, 0x65, 0x43, 0x6f, 0x6d, 0x70, 0x6f, 0x6e, 0x65,
	0x6e, 0x74, 0x53, 0x74, 0x61, 0x74, 0x65, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a,
	0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12,
	0x33, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1d,
	0x2e, 0x75, 0x6e, 0x72, 0x65, 0x61, 0x6c, 0x70, 0x62, 0x2e, 0x53, 0x63, 0x65, 0x6e, 0x65, 0x43,
	0x6f, 0x6d, 0x70, 0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x42, 0x35, 0x5a, 0x33, 0x63, 0x68, 0x61, 0x6e,
	0x6e, 0x65, 0x6c, 0x64, 0x2e, 0x63, 0x6c, 0x65, 0x77, 0x63, 0x61, 0x74, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2f, 0x63, 0x68, 0x61, 0x6e, 0x6e, 0x65,
	0x6c, 0x64, 0x2d, 0x75, 0x65, 0x2d, 0x74, 0x70, 0x73, 0x2f, 0x74, 0x70, 0x73, 0x70, 0x62, 0x62,
	0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_tps_proto_rawDescOnce sync.Once
	file_tps_proto_rawDescData = file_tps_proto_rawDesc
)

func file_tps_proto_rawDescGZIP() []byte {
	file_tps_proto_rawDescOnce.Do(func() {
		file_tps_proto_rawDescData = protoimpl.X.CompressGZIP(file_tps_proto_rawDescData)
	})
	return file_tps_proto_rawDescData
}

var file_tps_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_tps_proto_goTypes = []interface{}{
	(*TestRepChannelData)(nil),           // 0: tpspb.TestRepChannelData
	nil,                                  // 1: tpspb.TestRepChannelData.SceneComponentStatesEntry
	(*unrealpb.SceneComponentState)(nil), // 2: unrealpb.SceneComponentState
}
var file_tps_proto_depIdxs = []int32{
	1, // 0: tpspb.TestRepChannelData.sceneComponentStates:type_name -> tpspb.TestRepChannelData.SceneComponentStatesEntry
	2, // 1: tpspb.TestRepChannelData.SceneComponentStatesEntry.value:type_name -> unrealpb.SceneComponentState
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_tps_proto_init() }
func file_tps_proto_init() {
	if File_tps_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_tps_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TestRepChannelData); i {
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
			RawDescriptor: file_tps_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_tps_proto_goTypes,
		DependencyIndexes: file_tps_proto_depIdxs,
		MessageInfos:      file_tps_proto_msgTypes,
	}.Build()
	File_tps_proto = out.File
	file_tps_proto_rawDesc = nil
	file_tps_proto_goTypes = nil
	file_tps_proto_depIdxs = nil
}
