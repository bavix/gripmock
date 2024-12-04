// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.35.2
// 	protoc        v4.24.4
// source: file1.proto

package multi_files

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

// The request message containing the user's name.
type Request1 struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
}

func (x *Request1) Reset() {
	*x = Request1{}
	mi := &file_file1_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Request1) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Request1) ProtoMessage() {}

func (x *Request1) ProtoReflect() protoreflect.Message {
	mi := &file_file1_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Request1.ProtoReflect.Descriptor instead.
func (*Request1) Descriptor() ([]byte, []int) {
	return file_file1_proto_rawDescGZIP(), []int{0}
}

func (x *Request1) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

// The response message containing the greetings
type Reply1 struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Message    string `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
	ReturnCode int32  `protobuf:"varint,2,opt,name=return_code,json=returnCode,proto3" json:"return_code,omitempty"`
}

func (x *Reply1) Reset() {
	*x = Reply1{}
	mi := &file_file1_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Reply1) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Reply1) ProtoMessage() {}

func (x *Reply1) ProtoReflect() protoreflect.Message {
	mi := &file_file1_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Reply1.ProtoReflect.Descriptor instead.
func (*Reply1) Descriptor() ([]byte, []int) {
	return file_file1_proto_rawDescGZIP(), []int{1}
}

func (x *Reply1) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

func (x *Reply1) GetReturnCode() int32 {
	if x != nil {
		return x.ReturnCode
	}
	return 0
}

var File_file1_proto protoreflect.FileDescriptor

var file_file1_proto_rawDesc = []byte{
	0x0a, 0x0b, 0x66, 0x69, 0x6c, 0x65, 0x31, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x6d,
	0x75, 0x6c, 0x74, 0x69, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x22, 0x1e, 0x0a, 0x08, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x31, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x22, 0x43, 0x0a, 0x06, 0x52, 0x65, 0x70,
	0x6c, 0x79, 0x31, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x1f, 0x0a,
	0x0b, 0x72, 0x65, 0x74, 0x75, 0x72, 0x6e, 0x5f, 0x63, 0x6f, 0x64, 0x65, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x05, 0x52, 0x0a, 0x72, 0x65, 0x74, 0x75, 0x72, 0x6e, 0x43, 0x6f, 0x64, 0x65, 0x32, 0x41,
	0x0a, 0x09, 0x47, 0x72, 0x69, 0x70, 0x6d, 0x6f, 0x63, 0x6b, 0x31, 0x12, 0x34, 0x0a, 0x08, 0x53,
	0x61, 0x79, 0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x12, 0x14, 0x2e, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x66,
	0x69, 0x6c, 0x65, 0x73, 0x2e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x31, 0x1a, 0x12, 0x2e,
	0x6d, 0x75, 0x6c, 0x74, 0x69, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x2e, 0x52, 0x65, 0x70, 0x6c, 0x79,
	0x31, 0x42, 0x38, 0x5a, 0x36, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x62, 0x61, 0x76, 0x69, 0x78, 0x2f, 0x67, 0x72, 0x69, 0x70, 0x6d, 0x6f, 0x63, 0x6b, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x67, 0x65, 0x6e, 0x2f, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2f,
	0x6d, 0x75, 0x6c, 0x74, 0x69, 0x2d, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_file1_proto_rawDescOnce sync.Once
	file_file1_proto_rawDescData = file_file1_proto_rawDesc
)

func file_file1_proto_rawDescGZIP() []byte {
	file_file1_proto_rawDescOnce.Do(func() {
		file_file1_proto_rawDescData = protoimpl.X.CompressGZIP(file_file1_proto_rawDescData)
	})
	return file_file1_proto_rawDescData
}

var file_file1_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_file1_proto_goTypes = []any{
	(*Request1)(nil), // 0: multifiles.Request1
	(*Reply1)(nil),   // 1: multifiles.Reply1
}
var file_file1_proto_depIdxs = []int32{
	0, // 0: multifiles.Gripmock1.SayHello:input_type -> multifiles.Request1
	1, // 1: multifiles.Gripmock1.SayHello:output_type -> multifiles.Reply1
	1, // [1:2] is the sub-list for method output_type
	0, // [0:1] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_file1_proto_init() }
func file_file1_proto_init() {
	if File_file1_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_file1_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_file1_proto_goTypes,
		DependencyIndexes: file_file1_proto_depIdxs,
		MessageInfos:      file_file1_proto_msgTypes,
	}.Build()
	File_file1_proto = out.File
	file_file1_proto_rawDesc = nil
	file_file1_proto_goTypes = nil
	file_file1_proto_depIdxs = nil
}
