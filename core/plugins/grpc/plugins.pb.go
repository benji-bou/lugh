// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.33.0
// 	protoc        v5.26.1
// source: core/plugins/grpc/plugins.proto

package grpc

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

type RunInputConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Config []byte `protobuf:"bytes,1,opt,name=config,proto3" json:"config,omitempty"`
}

func (x *RunInputConfig) Reset() {
	*x = RunInputConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_core_plugins_grpc_plugins_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RunInputConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RunInputConfig) ProtoMessage() {}

func (x *RunInputConfig) ProtoReflect() protoreflect.Message {
	mi := &file_core_plugins_grpc_plugins_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RunInputConfig.ProtoReflect.Descriptor instead.
func (*RunInputConfig) Descriptor() ([]byte, []int) {
	return file_core_plugins_grpc_plugins_proto_rawDescGZIP(), []int{0}
}

func (x *RunInputConfig) GetConfig() []byte {
	if x != nil {
		return x.Config
	}
	return nil
}

type InputSchema struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Config []byte `protobuf:"bytes,1,opt,name=config,proto3" json:"config,omitempty"`
}

func (x *InputSchema) Reset() {
	*x = InputSchema{}
	if protoimpl.UnsafeEnabled {
		mi := &file_core_plugins_grpc_plugins_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *InputSchema) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InputSchema) ProtoMessage() {}

func (x *InputSchema) ProtoReflect() protoreflect.Message {
	mi := &file_core_plugins_grpc_plugins_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use InputSchema.ProtoReflect.Descriptor instead.
func (*InputSchema) Descriptor() ([]byte, []int) {
	return file_core_plugins_grpc_plugins_proto_rawDescGZIP(), []int{1}
}

func (x *InputSchema) GetConfig() []byte {
	if x != nil {
		return x.Config
	}
	return nil
}

type DataStream struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Data       []byte `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	ParentSrc  string `protobuf:"bytes,2,opt,name=parentSrc,proto3" json:"parentSrc,omitempty"`
	Id         string `protobuf:"bytes,3,opt,name=id,proto3" json:"id,omitempty"`
	IsComplete bool   `protobuf:"varint,4,opt,name=isComplete,proto3" json:"isComplete,omitempty"`
	TotalLen   int64  `protobuf:"varint,5,opt,name=totalLen,proto3" json:"totalLen,omitempty"`
}

func (x *DataStream) Reset() {
	*x = DataStream{}
	if protoimpl.UnsafeEnabled {
		mi := &file_core_plugins_grpc_plugins_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DataStream) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DataStream) ProtoMessage() {}

func (x *DataStream) ProtoReflect() protoreflect.Message {
	mi := &file_core_plugins_grpc_plugins_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DataStream.ProtoReflect.Descriptor instead.
func (*DataStream) Descriptor() ([]byte, []int) {
	return file_core_plugins_grpc_plugins_proto_rawDescGZIP(), []int{2}
}

func (x *DataStream) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

func (x *DataStream) GetParentSrc() string {
	if x != nil {
		return x.ParentSrc
	}
	return ""
}

func (x *DataStream) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *DataStream) GetIsComplete() bool {
	if x != nil {
		return x.IsComplete
	}
	return false
}

func (x *DataStream) GetTotalLen() int64 {
	if x != nil {
		return x.TotalLen
	}
	return 0
}

type Empty struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *Empty) Reset() {
	*x = Empty{}
	if protoimpl.UnsafeEnabled {
		mi := &file_core_plugins_grpc_plugins_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Empty) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Empty) ProtoMessage() {}

func (x *Empty) ProtoReflect() protoreflect.Message {
	mi := &file_core_plugins_grpc_plugins_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Empty.ProtoReflect.Descriptor instead.
func (*Empty) Descriptor() ([]byte, []int) {
	return file_core_plugins_grpc_plugins_proto_rawDescGZIP(), []int{3}
}

type Error struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Message string `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
}

func (x *Error) Reset() {
	*x = Error{}
	if protoimpl.UnsafeEnabled {
		mi := &file_core_plugins_grpc_plugins_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Error) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Error) ProtoMessage() {}

func (x *Error) ProtoReflect() protoreflect.Message {
	mi := &file_core_plugins_grpc_plugins_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Error.ProtoReflect.Descriptor instead.
func (*Error) Descriptor() ([]byte, []int) {
	return file_core_plugins_grpc_plugins_proto_rawDescGZIP(), []int{4}
}

func (x *Error) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

var File_core_plugins_grpc_plugins_proto protoreflect.FileDescriptor

var file_core_plugins_grpc_plugins_proto_rawDesc = []byte{
	0x0a, 0x1f, 0x63, 0x6f, 0x72, 0x65, 0x2f, 0x70, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x73, 0x2f, 0x67,
	0x72, 0x70, 0x63, 0x2f, 0x70, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x04, 0x67, 0x72, 0x70, 0x63, 0x22, 0x28, 0x0a, 0x0e, 0x52, 0x75, 0x6e, 0x49, 0x6e,
	0x70, 0x75, 0x74, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x16, 0x0a, 0x06, 0x63, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x06, 0x63, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x22, 0x25, 0x0a, 0x0b, 0x49, 0x6e, 0x70, 0x75, 0x74, 0x53, 0x63, 0x68, 0x65, 0x6d, 0x61,
	0x12, 0x16, 0x0a, 0x06, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c,
	0x52, 0x06, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x22, 0x8a, 0x01, 0x0a, 0x0a, 0x44, 0x61, 0x74,
	0x61, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x12, 0x1c, 0x0a, 0x09, 0x70,
	0x61, 0x72, 0x65, 0x6e, 0x74, 0x53, 0x72, 0x63, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09,
	0x70, 0x61, 0x72, 0x65, 0x6e, 0x74, 0x53, 0x72, 0x63, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x1e, 0x0a, 0x0a, 0x69, 0x73, 0x43,
	0x6f, 0x6d, 0x70, 0x6c, 0x65, 0x74, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0a, 0x69,
	0x73, 0x43, 0x6f, 0x6d, 0x70, 0x6c, 0x65, 0x74, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x74, 0x6f, 0x74,
	0x61, 0x6c, 0x4c, 0x65, 0x6e, 0x18, 0x05, 0x20, 0x01, 0x28, 0x03, 0x52, 0x08, 0x74, 0x6f, 0x74,
	0x61, 0x6c, 0x4c, 0x65, 0x6e, 0x22, 0x07, 0x0a, 0x05, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x21,
	0x0a, 0x05, 0x45, 0x72, 0x72, 0x6f, 0x72, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61,
	0x67, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67,
	0x65, 0x32, 0xe8, 0x01, 0x0a, 0x0f, 0x49, 0x4f, 0x57, 0x6f, 0x72, 0x6b, 0x65, 0x72, 0x50, 0x6c,
	0x75, 0x67, 0x69, 0x6e, 0x73, 0x12, 0x30, 0x0a, 0x0e, 0x47, 0x65, 0x74, 0x49, 0x6e, 0x70, 0x75,
	0x74, 0x53, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x12, 0x0b, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x45,
	0x6d, 0x70, 0x74, 0x79, 0x1a, 0x11, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x49, 0x6e, 0x70, 0x75,
	0x74, 0x53, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x12, 0x2b, 0x0a, 0x06, 0x43, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x12, 0x14, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x52, 0x75, 0x6e, 0x49, 0x6e, 0x70, 0x75,
	0x74, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x1a, 0x0b, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x45,
	0x6d, 0x70, 0x74, 0x79, 0x12, 0x28, 0x0a, 0x05, 0x49, 0x6e, 0x70, 0x75, 0x74, 0x12, 0x10, 0x2e,
	0x67, 0x72, 0x70, 0x63, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x1a,
	0x0b, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x28, 0x01, 0x12, 0x29,
	0x0a, 0x06, 0x4f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x12, 0x0b, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e,
	0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x10, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x44, 0x61, 0x74,
	0x61, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x30, 0x01, 0x12, 0x21, 0x0a, 0x03, 0x52, 0x75, 0x6e,
	0x12, 0x0b, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x0b, 0x2e,
	0x67, 0x72, 0x70, 0x63, 0x2e, 0x45, 0x72, 0x72, 0x6f, 0x72, 0x30, 0x01, 0x42, 0x34, 0x5a, 0x32,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x62, 0x65, 0x6e, 0x6a, 0x69,
	0x2d, 0x62, 0x6f, 0x75, 0x2f, 0x53, 0x65, 0x63, 0x50, 0x69, 0x70, 0x65, 0x6c, 0x69, 0x6e, 0x65,
	0x2f, 0x63, 0x6f, 0x72, 0x65, 0x2f, 0x70, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x73, 0x2f, 0x67, 0x72,
	0x70, 0x63, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_core_plugins_grpc_plugins_proto_rawDescOnce sync.Once
	file_core_plugins_grpc_plugins_proto_rawDescData = file_core_plugins_grpc_plugins_proto_rawDesc
)

func file_core_plugins_grpc_plugins_proto_rawDescGZIP() []byte {
	file_core_plugins_grpc_plugins_proto_rawDescOnce.Do(func() {
		file_core_plugins_grpc_plugins_proto_rawDescData = protoimpl.X.CompressGZIP(file_core_plugins_grpc_plugins_proto_rawDescData)
	})
	return file_core_plugins_grpc_plugins_proto_rawDescData
}

var file_core_plugins_grpc_plugins_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_core_plugins_grpc_plugins_proto_goTypes = []interface{}{
	(*RunInputConfig)(nil), // 0: grpc.RunInputConfig
	(*InputSchema)(nil),    // 1: grpc.InputSchema
	(*DataStream)(nil),     // 2: grpc.DataStream
	(*Empty)(nil),          // 3: grpc.Empty
	(*Error)(nil),          // 4: grpc.Error
}
var file_core_plugins_grpc_plugins_proto_depIdxs = []int32{
	3, // 0: grpc.IOWorkerPlugins.GetInputSchema:input_type -> grpc.Empty
	0, // 1: grpc.IOWorkerPlugins.Config:input_type -> grpc.RunInputConfig
	2, // 2: grpc.IOWorkerPlugins.Input:input_type -> grpc.DataStream
	3, // 3: grpc.IOWorkerPlugins.Output:input_type -> grpc.Empty
	3, // 4: grpc.IOWorkerPlugins.Run:input_type -> grpc.Empty
	1, // 5: grpc.IOWorkerPlugins.GetInputSchema:output_type -> grpc.InputSchema
	3, // 6: grpc.IOWorkerPlugins.Config:output_type -> grpc.Empty
	3, // 7: grpc.IOWorkerPlugins.Input:output_type -> grpc.Empty
	2, // 8: grpc.IOWorkerPlugins.Output:output_type -> grpc.DataStream
	4, // 9: grpc.IOWorkerPlugins.Run:output_type -> grpc.Error
	5, // [5:10] is the sub-list for method output_type
	0, // [0:5] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_core_plugins_grpc_plugins_proto_init() }
func file_core_plugins_grpc_plugins_proto_init() {
	if File_core_plugins_grpc_plugins_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_core_plugins_grpc_plugins_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RunInputConfig); i {
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
		file_core_plugins_grpc_plugins_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*InputSchema); i {
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
		file_core_plugins_grpc_plugins_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DataStream); i {
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
		file_core_plugins_grpc_plugins_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Empty); i {
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
		file_core_plugins_grpc_plugins_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Error); i {
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
			RawDescriptor: file_core_plugins_grpc_plugins_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_core_plugins_grpc_plugins_proto_goTypes,
		DependencyIndexes: file_core_plugins_grpc_plugins_proto_depIdxs,
		MessageInfos:      file_core_plugins_grpc_plugins_proto_msgTypes,
	}.Build()
	File_core_plugins_grpc_plugins_proto = out.File
	file_core_plugins_grpc_plugins_proto_rawDesc = nil
	file_core_plugins_grpc_plugins_proto_goTypes = nil
	file_core_plugins_grpc_plugins_proto_depIdxs = nil
}
