// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.18.0
// source: test.proto

package proto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	anypb "google.golang.org/protobuf/types/known/anypb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type TestChannelDataMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Text string `protobuf:"bytes,1,opt,name=text,proto3" json:"text,omitempty"`
	Num  uint32 `protobuf:"varint,2,opt,name=num,proto3" json:"num,omitempty"`
}

func (x *TestChannelDataMessage) Reset() {
	*x = TestChannelDataMessage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TestChannelDataMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestChannelDataMessage) ProtoMessage() {}

func (x *TestChannelDataMessage) ProtoReflect() protoreflect.Message {
	mi := &file_test_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestChannelDataMessage.ProtoReflect.Descriptor instead.
func (*TestChannelDataMessage) Descriptor() ([]byte, []int) {
	return file_test_proto_rawDescGZIP(), []int{0}
}

func (x *TestChannelDataMessage) GetText() string {
	if x != nil {
		return x.Text
	}
	return ""
}

func (x *TestChannelDataMessage) GetNum() uint32 {
	if x != nil {
		return x.Num
	}
	return 0
}

type TestAnyMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Msg  *anypb.Any   `protobuf:"bytes,1,opt,name=msg,proto3" json:"msg,omitempty"`
	List []*anypb.Any `protobuf:"bytes,2,rep,name=list,proto3" json:"list,omitempty"`
}

func (x *TestAnyMessage) Reset() {
	*x = TestAnyMessage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TestAnyMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestAnyMessage) ProtoMessage() {}

func (x *TestAnyMessage) ProtoReflect() protoreflect.Message {
	mi := &file_test_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestAnyMessage.ProtoReflect.Descriptor instead.
func (*TestAnyMessage) Descriptor() ([]byte, []int) {
	return file_test_proto_rawDescGZIP(), []int{1}
}

func (x *TestAnyMessage) GetMsg() *anypb.Any {
	if x != nil {
		return x.Msg
	}
	return nil
}

func (x *TestAnyMessage) GetList() []*anypb.Any {
	if x != nil {
		return x.List
	}
	return nil
}

type TestMergeMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	List []string                                  `protobuf:"bytes,1,rep,name=list,proto3" json:"list,omitempty"`
	Kv   map[int64]*TestMergeMessage_StringWrapper `protobuf:"bytes,2,rep,name=kv,proto3" json:"kv,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *TestMergeMessage) Reset() {
	*x = TestMergeMessage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TestMergeMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestMergeMessage) ProtoMessage() {}

func (x *TestMergeMessage) ProtoReflect() protoreflect.Message {
	mi := &file_test_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestMergeMessage.ProtoReflect.Descriptor instead.
func (*TestMergeMessage) Descriptor() ([]byte, []int) {
	return file_test_proto_rawDescGZIP(), []int{2}
}

func (x *TestMergeMessage) GetList() []string {
	if x != nil {
		return x.List
	}
	return nil
}

func (x *TestMergeMessage) GetKv() map[int64]*TestMergeMessage_StringWrapper {
	if x != nil {
		return x.Kv
	}
	return nil
}

type TestMapMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Kv  map[uint32]string                        `protobuf:"bytes,1,rep,name=kv,proto3" json:"kv,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Kv2 map[uint32]*TestMapMessage_StringWrapper `protobuf:"bytes,2,rep,name=kv2,proto3" json:"kv2,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *TestMapMessage) Reset() {
	*x = TestMapMessage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TestMapMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestMapMessage) ProtoMessage() {}

func (x *TestMapMessage) ProtoReflect() protoreflect.Message {
	mi := &file_test_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestMapMessage.ProtoReflect.Descriptor instead.
func (*TestMapMessage) Descriptor() ([]byte, []int) {
	return file_test_proto_rawDescGZIP(), []int{3}
}

func (x *TestMapMessage) GetKv() map[uint32]string {
	if x != nil {
		return x.Kv
	}
	return nil
}

func (x *TestMapMessage) GetKv2() map[uint32]*TestMapMessage_StringWrapper {
	if x != nil {
		return x.Kv2
	}
	return nil
}

type TestFieldMaskMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name string                                        `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Msg  *TestFieldMaskMessage_NestedMessage           `protobuf:"bytes,2,opt,name=msg,proto3" json:"msg,omitempty"`
	List []*TestFieldMaskMessage_NestedMessage         `protobuf:"bytes,3,rep,name=list,proto3" json:"list,omitempty"`
	Kv1  map[int64]*TestFieldMaskMessage_NestedMessage `protobuf:"bytes,4,rep,name=kv1,proto3" json:"kv1,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Kv2  map[int64]string                              `protobuf:"bytes,5,rep,name=kv2,proto3" json:"kv2,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *TestFieldMaskMessage) Reset() {
	*x = TestFieldMaskMessage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TestFieldMaskMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestFieldMaskMessage) ProtoMessage() {}

func (x *TestFieldMaskMessage) ProtoReflect() protoreflect.Message {
	mi := &file_test_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestFieldMaskMessage.ProtoReflect.Descriptor instead.
func (*TestFieldMaskMessage) Descriptor() ([]byte, []int) {
	return file_test_proto_rawDescGZIP(), []int{4}
}

func (x *TestFieldMaskMessage) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *TestFieldMaskMessage) GetMsg() *TestFieldMaskMessage_NestedMessage {
	if x != nil {
		return x.Msg
	}
	return nil
}

func (x *TestFieldMaskMessage) GetList() []*TestFieldMaskMessage_NestedMessage {
	if x != nil {
		return x.List
	}
	return nil
}

func (x *TestFieldMaskMessage) GetKv1() map[int64]*TestFieldMaskMessage_NestedMessage {
	if x != nil {
		return x.Kv1
	}
	return nil
}

func (x *TestFieldMaskMessage) GetKv2() map[int64]string {
	if x != nil {
		return x.Kv2
	}
	return nil
}

type TestAnyMessage_Type1 struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Value string `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *TestAnyMessage_Type1) Reset() {
	*x = TestAnyMessage_Type1{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TestAnyMessage_Type1) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestAnyMessage_Type1) ProtoMessage() {}

func (x *TestAnyMessage_Type1) ProtoReflect() protoreflect.Message {
	mi := &file_test_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestAnyMessage_Type1.ProtoReflect.Descriptor instead.
func (*TestAnyMessage_Type1) Descriptor() ([]byte, []int) {
	return file_test_proto_rawDescGZIP(), []int{1, 0}
}

func (x *TestAnyMessage_Type1) GetValue() string {
	if x != nil {
		return x.Value
	}
	return ""
}

type TestAnyMessage_Type2 struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Value int64 `protobuf:"varint,1,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *TestAnyMessage_Type2) Reset() {
	*x = TestAnyMessage_Type2{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TestAnyMessage_Type2) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestAnyMessage_Type2) ProtoMessage() {}

func (x *TestAnyMessage_Type2) ProtoReflect() protoreflect.Message {
	mi := &file_test_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestAnyMessage_Type2.ProtoReflect.Descriptor instead.
func (*TestAnyMessage_Type2) Descriptor() ([]byte, []int) {
	return file_test_proto_rawDescGZIP(), []int{1, 1}
}

func (x *TestAnyMessage_Type2) GetValue() int64 {
	if x != nil {
		return x.Value
	}
	return 0
}

type TestMergeMessage_StringWrapper struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Removed bool   `protobuf:"varint,1,opt,name=removed,proto3" json:"removed,omitempty"`
	Content string `protobuf:"bytes,2,opt,name=content,proto3" json:"content,omitempty"`
}

func (x *TestMergeMessage_StringWrapper) Reset() {
	*x = TestMergeMessage_StringWrapper{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TestMergeMessage_StringWrapper) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestMergeMessage_StringWrapper) ProtoMessage() {}

func (x *TestMergeMessage_StringWrapper) ProtoReflect() protoreflect.Message {
	mi := &file_test_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestMergeMessage_StringWrapper.ProtoReflect.Descriptor instead.
func (*TestMergeMessage_StringWrapper) Descriptor() ([]byte, []int) {
	return file_test_proto_rawDescGZIP(), []int{2, 0}
}

func (x *TestMergeMessage_StringWrapper) GetRemoved() bool {
	if x != nil {
		return x.Removed
	}
	return false
}

func (x *TestMergeMessage_StringWrapper) GetContent() string {
	if x != nil {
		return x.Content
	}
	return ""
}

type TestMapMessage_StringWrapper struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Content string `protobuf:"bytes,1,opt,name=content,proto3" json:"content,omitempty"`
	Num     int64  `protobuf:"varint,2,opt,name=num,proto3" json:"num,omitempty"`
}

func (x *TestMapMessage_StringWrapper) Reset() {
	*x = TestMapMessage_StringWrapper{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_proto_msgTypes[10]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TestMapMessage_StringWrapper) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestMapMessage_StringWrapper) ProtoMessage() {}

func (x *TestMapMessage_StringWrapper) ProtoReflect() protoreflect.Message {
	mi := &file_test_proto_msgTypes[10]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestMapMessage_StringWrapper.ProtoReflect.Descriptor instead.
func (*TestMapMessage_StringWrapper) Descriptor() ([]byte, []int) {
	return file_test_proto_rawDescGZIP(), []int{3, 1}
}

func (x *TestMapMessage_StringWrapper) GetContent() string {
	if x != nil {
		return x.Content
	}
	return ""
}

func (x *TestMapMessage_StringWrapper) GetNum() int64 {
	if x != nil {
		return x.Num
	}
	return 0
}

type TestFieldMaskMessage_NestedMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	P1 int64  `protobuf:"varint,1,opt,name=p1,proto3" json:"p1,omitempty"`
	P2 uint32 `protobuf:"varint,2,opt,name=p2,proto3" json:"p2,omitempty"`
}

func (x *TestFieldMaskMessage_NestedMessage) Reset() {
	*x = TestFieldMaskMessage_NestedMessage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_proto_msgTypes[12]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TestFieldMaskMessage_NestedMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestFieldMaskMessage_NestedMessage) ProtoMessage() {}

func (x *TestFieldMaskMessage_NestedMessage) ProtoReflect() protoreflect.Message {
	mi := &file_test_proto_msgTypes[12]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestFieldMaskMessage_NestedMessage.ProtoReflect.Descriptor instead.
func (*TestFieldMaskMessage_NestedMessage) Descriptor() ([]byte, []int) {
	return file_test_proto_rawDescGZIP(), []int{4, 0}
}

func (x *TestFieldMaskMessage_NestedMessage) GetP1() int64 {
	if x != nil {
		return x.P1
	}
	return 0
}

func (x *TestFieldMaskMessage_NestedMessage) GetP2() uint32 {
	if x != nil {
		return x.P2
	}
	return 0
}

var File_test_proto protoreflect.FileDescriptor

var file_test_proto_rawDesc = []byte{
	0x0a, 0x0a, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x08, 0x63, 0x68,
	0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x64, 0x1a, 0x19, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6e, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x22, 0x3e, 0x0a, 0x16, 0x54, 0x65, 0x73, 0x74, 0x43, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c,
	0x44, 0x61, 0x74, 0x61, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x74,
	0x65, 0x78, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x65, 0x78, 0x74, 0x12,
	0x10, 0x0a, 0x03, 0x6e, 0x75, 0x6d, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x03, 0x6e, 0x75,
	0x6d, 0x22, 0xa0, 0x01, 0x0a, 0x0e, 0x54, 0x65, 0x73, 0x74, 0x41, 0x6e, 0x79, 0x4d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x12, 0x26, 0x0a, 0x03, 0x6d, 0x73, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x14, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x41, 0x6e, 0x79, 0x52, 0x03, 0x6d, 0x73, 0x67, 0x12, 0x28, 0x0a, 0x04,
	0x6c, 0x69, 0x73, 0x74, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x41, 0x6e, 0x79,
	0x52, 0x04, 0x6c, 0x69, 0x73, 0x74, 0x1a, 0x1d, 0x0a, 0x05, 0x54, 0x79, 0x70, 0x65, 0x31, 0x12,
	0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x1a, 0x1d, 0x0a, 0x05, 0x54, 0x79, 0x70, 0x65, 0x32, 0x12, 0x14,
	0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x22, 0x80, 0x02, 0x0a, 0x10, 0x54, 0x65, 0x73, 0x74, 0x4d, 0x65, 0x72,
	0x67, 0x65, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x6c, 0x69, 0x73,
	0x74, 0x18, 0x01, 0x20, 0x03, 0x28, 0x09, 0x52, 0x04, 0x6c, 0x69, 0x73, 0x74, 0x12, 0x32, 0x0a,
	0x02, 0x6b, 0x76, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x22, 0x2e, 0x63, 0x68, 0x61, 0x6e,
	0x6e, 0x65, 0x6c, 0x64, 0x2e, 0x54, 0x65, 0x73, 0x74, 0x4d, 0x65, 0x72, 0x67, 0x65, 0x4d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x4b, 0x76, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x02, 0x6b,
	0x76, 0x1a, 0x43, 0x0a, 0x0d, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x57, 0x72, 0x61, 0x70, 0x70,
	0x65, 0x72, 0x12, 0x18, 0x0a, 0x07, 0x72, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x08, 0x52, 0x07, 0x72, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x64, 0x12, 0x18, 0x0a, 0x07,
	0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x63,
	0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x1a, 0x5f, 0x0a, 0x07, 0x4b, 0x76, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x03,
	0x6b, 0x65, 0x79, 0x12, 0x3e, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x28, 0x2e, 0x63, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x64, 0x2e, 0x54, 0x65,
	0x73, 0x74, 0x4d, 0x65, 0x72, 0x67, 0x65, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x53,
	0x74, 0x72, 0x69, 0x6e, 0x67, 0x57, 0x72, 0x61, 0x70, 0x70, 0x65, 0x72, 0x52, 0x05, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0xcb, 0x02, 0x0a, 0x0e, 0x54, 0x65, 0x73, 0x74,
	0x4d, 0x61, 0x70, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x30, 0x0a, 0x02, 0x6b, 0x76,
	0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x20, 0x2e, 0x63, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c,
	0x64, 0x2e, 0x54, 0x65, 0x73, 0x74, 0x4d, 0x61, 0x70, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65,
	0x2e, 0x4b, 0x76, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x02, 0x6b, 0x76, 0x12, 0x33, 0x0a, 0x03,
	0x6b, 0x76, 0x32, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x21, 0x2e, 0x63, 0x68, 0x61, 0x6e,
	0x6e, 0x65, 0x6c, 0x64, 0x2e, 0x54, 0x65, 0x73, 0x74, 0x4d, 0x61, 0x70, 0x4d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x2e, 0x4b, 0x76, 0x32, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x03, 0x6b, 0x76,
	0x32, 0x1a, 0x35, 0x0a, 0x07, 0x4b, 0x76, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03,
	0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14,
	0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a, 0x3b, 0x0a, 0x0d, 0x53, 0x74, 0x72, 0x69,
	0x6e, 0x67, 0x57, 0x72, 0x61, 0x70, 0x70, 0x65, 0x72, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x6f, 0x6e,
	0x74, 0x65, 0x6e, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x63, 0x6f, 0x6e, 0x74,
	0x65, 0x6e, 0x74, 0x12, 0x10, 0x0a, 0x03, 0x6e, 0x75, 0x6d, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03,
	0x52, 0x03, 0x6e, 0x75, 0x6d, 0x1a, 0x5e, 0x0a, 0x08, 0x4b, 0x76, 0x32, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x03,
	0x6b, 0x65, 0x79, 0x12, 0x3c, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x26, 0x2e, 0x63, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x64, 0x2e, 0x54, 0x65,
	0x73, 0x74, 0x4d, 0x61, 0x70, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x53, 0x74, 0x72,
	0x69, 0x6e, 0x67, 0x57, 0x72, 0x61, 0x70, 0x70, 0x65, 0x72, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0xf1, 0x03, 0x0a, 0x14, 0x54, 0x65, 0x73, 0x74, 0x46, 0x69,
	0x65, 0x6c, 0x64, 0x4d, 0x61, 0x73, 0x6b, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x12,
	0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61,
	0x6d, 0x65, 0x12, 0x3e, 0x0a, 0x03, 0x6d, 0x73, 0x67, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x2c, 0x2e, 0x63, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x64, 0x2e, 0x54, 0x65, 0x73, 0x74, 0x46,
	0x69, 0x65, 0x6c, 0x64, 0x4d, 0x61, 0x73, 0x6b, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e,
	0x4e, 0x65, 0x73, 0x74, 0x65, 0x64, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x52, 0x03, 0x6d,
	0x73, 0x67, 0x12, 0x40, 0x0a, 0x04, 0x6c, 0x69, 0x73, 0x74, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x2c, 0x2e, 0x63, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x64, 0x2e, 0x54, 0x65, 0x73, 0x74,
	0x46, 0x69, 0x65, 0x6c, 0x64, 0x4d, 0x61, 0x73, 0x6b, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65,
	0x2e, 0x4e, 0x65, 0x73, 0x74, 0x65, 0x64, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x52, 0x04,
	0x6c, 0x69, 0x73, 0x74, 0x12, 0x39, 0x0a, 0x03, 0x6b, 0x76, 0x31, 0x18, 0x04, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x27, 0x2e, 0x63, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x64, 0x2e, 0x54, 0x65, 0x73,
	0x74, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x4d, 0x61, 0x73, 0x6b, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67,
	0x65, 0x2e, 0x4b, 0x76, 0x31, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x03, 0x6b, 0x76, 0x31, 0x12,
	0x39, 0x0a, 0x03, 0x6b, 0x76, 0x32, 0x18, 0x05, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x27, 0x2e, 0x63,
	0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x64, 0x2e, 0x54, 0x65, 0x73, 0x74, 0x46, 0x69, 0x65, 0x6c,
	0x64, 0x4d, 0x61, 0x73, 0x6b, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x4b, 0x76, 0x32,
	0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x03, 0x6b, 0x76, 0x32, 0x1a, 0x2f, 0x0a, 0x0d, 0x4e, 0x65,
	0x73, 0x74, 0x65, 0x64, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x0e, 0x0a, 0x02, 0x70,
	0x31, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x02, 0x70, 0x31, 0x12, 0x0e, 0x0a, 0x02, 0x70,
	0x32, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x02, 0x70, 0x32, 0x1a, 0x64, 0x0a, 0x08, 0x4b,
	0x76, 0x31, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x03, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x42, 0x0a, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2c, 0x2e, 0x63, 0x68, 0x61, 0x6e, 0x6e,
	0x65, 0x6c, 0x64, 0x2e, 0x54, 0x65, 0x73, 0x74, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x4d, 0x61, 0x73,
	0x6b, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x4e, 0x65, 0x73, 0x74, 0x65, 0x64, 0x4d,
	0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38,
	0x01, 0x1a, 0x36, 0x0a, 0x08, 0x4b, 0x76, 0x32, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a,
	0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12,
	0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x42, 0x08, 0x5a, 0x06, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_test_proto_rawDescOnce sync.Once
	file_test_proto_rawDescData = file_test_proto_rawDesc
)

func file_test_proto_rawDescGZIP() []byte {
	file_test_proto_rawDescOnce.Do(func() {
		file_test_proto_rawDescData = protoimpl.X.CompressGZIP(file_test_proto_rawDescData)
	})
	return file_test_proto_rawDescData
}

var file_test_proto_msgTypes = make([]protoimpl.MessageInfo, 15)
var file_test_proto_goTypes = []interface{}{
	(*TestChannelDataMessage)(nil),         // 0: channeld.TestChannelDataMessage
	(*TestAnyMessage)(nil),                 // 1: channeld.TestAnyMessage
	(*TestMergeMessage)(nil),               // 2: channeld.TestMergeMessage
	(*TestMapMessage)(nil),                 // 3: channeld.TestMapMessage
	(*TestFieldMaskMessage)(nil),           // 4: channeld.TestFieldMaskMessage
	(*TestAnyMessage_Type1)(nil),           // 5: channeld.TestAnyMessage.Type1
	(*TestAnyMessage_Type2)(nil),           // 6: channeld.TestAnyMessage.Type2
	(*TestMergeMessage_StringWrapper)(nil), // 7: channeld.TestMergeMessage.StringWrapper
	nil,                                    // 8: channeld.TestMergeMessage.KvEntry
	nil,                                    // 9: channeld.TestMapMessage.KvEntry
	(*TestMapMessage_StringWrapper)(nil),   // 10: channeld.TestMapMessage.StringWrapper
	nil,                                    // 11: channeld.TestMapMessage.Kv2Entry
	(*TestFieldMaskMessage_NestedMessage)(nil), // 12: channeld.TestFieldMaskMessage.NestedMessage
	nil,               // 13: channeld.TestFieldMaskMessage.Kv1Entry
	nil,               // 14: channeld.TestFieldMaskMessage.Kv2Entry
	(*anypb.Any)(nil), // 15: google.protobuf.Any
}
var file_test_proto_depIdxs = []int32{
	15, // 0: channeld.TestAnyMessage.msg:type_name -> google.protobuf.Any
	15, // 1: channeld.TestAnyMessage.list:type_name -> google.protobuf.Any
	8,  // 2: channeld.TestMergeMessage.kv:type_name -> channeld.TestMergeMessage.KvEntry
	9,  // 3: channeld.TestMapMessage.kv:type_name -> channeld.TestMapMessage.KvEntry
	11, // 4: channeld.TestMapMessage.kv2:type_name -> channeld.TestMapMessage.Kv2Entry
	12, // 5: channeld.TestFieldMaskMessage.msg:type_name -> channeld.TestFieldMaskMessage.NestedMessage
	12, // 6: channeld.TestFieldMaskMessage.list:type_name -> channeld.TestFieldMaskMessage.NestedMessage
	13, // 7: channeld.TestFieldMaskMessage.kv1:type_name -> channeld.TestFieldMaskMessage.Kv1Entry
	14, // 8: channeld.TestFieldMaskMessage.kv2:type_name -> channeld.TestFieldMaskMessage.Kv2Entry
	7,  // 9: channeld.TestMergeMessage.KvEntry.value:type_name -> channeld.TestMergeMessage.StringWrapper
	10, // 10: channeld.TestMapMessage.Kv2Entry.value:type_name -> channeld.TestMapMessage.StringWrapper
	12, // 11: channeld.TestFieldMaskMessage.Kv1Entry.value:type_name -> channeld.TestFieldMaskMessage.NestedMessage
	12, // [12:12] is the sub-list for method output_type
	12, // [12:12] is the sub-list for method input_type
	12, // [12:12] is the sub-list for extension type_name
	12, // [12:12] is the sub-list for extension extendee
	0,  // [0:12] is the sub-list for field type_name
}

func init() { file_test_proto_init() }
func file_test_proto_init() {
	if File_test_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_test_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TestChannelDataMessage); i {
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
		file_test_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TestAnyMessage); i {
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
		file_test_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TestMergeMessage); i {
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
		file_test_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TestMapMessage); i {
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
		file_test_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TestFieldMaskMessage); i {
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
		file_test_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TestAnyMessage_Type1); i {
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
		file_test_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TestAnyMessage_Type2); i {
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
		file_test_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TestMergeMessage_StringWrapper); i {
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
		file_test_proto_msgTypes[10].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TestMapMessage_StringWrapper); i {
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
		file_test_proto_msgTypes[12].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TestFieldMaskMessage_NestedMessage); i {
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
			RawDescriptor: file_test_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   15,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_test_proto_goTypes,
		DependencyIndexes: file_test_proto_depIdxs,
		MessageInfos:      file_test_proto_msgTypes,
	}.Build()
	File_test_proto = out.File
	file_test_proto_rawDesc = nil
	file_test_proto_goTypes = nil
	file_test_proto_depIdxs = nil
}
