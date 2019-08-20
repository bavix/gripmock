// Code generated by protoc-gen-go. DO NOT EDIT.
// source: pb/hello.proto

package main

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import empty "github.com/golang/protobuf/ptypes/empty"
import wrappers "github.com/golang/protobuf/ptypes/wrappers"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// The request message containing the user's name.
type Request struct {
	Name                 string   `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Request) Reset()         { *m = Request{} }
func (m *Request) String() string { return proto.CompactTextString(m) }
func (*Request) ProtoMessage()    {}
func (*Request) Descriptor() ([]byte, []int) {
	return fileDescriptor_hello_64609aa425866106, []int{0}
}
func (m *Request) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Request.Unmarshal(m, b)
}
func (m *Request) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Request.Marshal(b, m, deterministic)
}
func (dst *Request) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Request.Merge(dst, src)
}
func (m *Request) XXX_Size() int {
	return xxx_messageInfo_Request.Size(m)
}
func (m *Request) XXX_DiscardUnknown() {
	xxx_messageInfo_Request.DiscardUnknown(m)
}

var xxx_messageInfo_Request proto.InternalMessageInfo

func (m *Request) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

// The response message containing the greetings
type Reply struct {
	Message              string   `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Reply) Reset()         { *m = Reply{} }
func (m *Reply) String() string { return proto.CompactTextString(m) }
func (*Reply) ProtoMessage()    {}
func (*Reply) Descriptor() ([]byte, []int) {
	return fileDescriptor_hello_64609aa425866106, []int{1}
}
func (m *Reply) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Reply.Unmarshal(m, b)
}
func (m *Reply) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Reply.Marshal(b, m, deterministic)
}
func (dst *Reply) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Reply.Merge(dst, src)
}
func (m *Reply) XXX_Size() int {
	return xxx_messageInfo_Reply.Size(m)
}
func (m *Reply) XXX_DiscardUnknown() {
	xxx_messageInfo_Reply.DiscardUnknown(m)
}

var xxx_messageInfo_Reply proto.InternalMessageInfo

func (m *Reply) GetMessage() string {
	if m != nil {
		return m.Message
	}
	return ""
}

func init() {
	proto.RegisterType((*Request)(nil), "main.Request")
	proto.RegisterType((*Reply)(nil), "main.Reply")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// GripmockClient is the client API for Gripmock service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type GripmockClient interface {
	// standard grpc method
	SayHello(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Reply, error)
	// server to client sreaming
	ServerStream(ctx context.Context, in *Request, opts ...grpc.CallOption) (Gripmock_ServerStreamClient, error)
	// client to server streaming
	ClientStream(ctx context.Context, opts ...grpc.CallOption) (Gripmock_ClientStreamClient, error)
	// bidirectional streaming
	Bidirectional(ctx context.Context, opts ...grpc.CallOption) (Gripmock_BidirectionalClient, error)
	// using well-known-type of Empty
	HealthCheck(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*wrappers.StringValue, error)
}

type gripmockClient struct {
	cc *grpc.ClientConn
}

func NewGripmockClient(cc *grpc.ClientConn) GripmockClient {
	return &gripmockClient{cc}
}

func (c *gripmockClient) SayHello(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Reply, error) {
	out := new(Reply)
	err := c.cc.Invoke(ctx, "/main.Gripmock/SayHello", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gripmockClient) ServerStream(ctx context.Context, in *Request, opts ...grpc.CallOption) (Gripmock_ServerStreamClient, error) {
	stream, err := c.cc.NewStream(ctx, &_Gripmock_serviceDesc.Streams[0], "/main.Gripmock/serverStream", opts...)
	if err != nil {
		return nil, err
	}
	x := &gripmockServerStreamClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Gripmock_ServerStreamClient interface {
	Recv() (*Reply, error)
	grpc.ClientStream
}

type gripmockServerStreamClient struct {
	grpc.ClientStream
}

func (x *gripmockServerStreamClient) Recv() (*Reply, error) {
	m := new(Reply)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *gripmockClient) ClientStream(ctx context.Context, opts ...grpc.CallOption) (Gripmock_ClientStreamClient, error) {
	stream, err := c.cc.NewStream(ctx, &_Gripmock_serviceDesc.Streams[1], "/main.Gripmock/clientStream", opts...)
	if err != nil {
		return nil, err
	}
	x := &gripmockClientStreamClient{stream}
	return x, nil
}

type Gripmock_ClientStreamClient interface {
	Send(*Request) error
	CloseAndRecv() (*Reply, error)
	grpc.ClientStream
}

type gripmockClientStreamClient struct {
	grpc.ClientStream
}

func (x *gripmockClientStreamClient) Send(m *Request) error {
	return x.ClientStream.SendMsg(m)
}

func (x *gripmockClientStreamClient) CloseAndRecv() (*Reply, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(Reply)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *gripmockClient) Bidirectional(ctx context.Context, opts ...grpc.CallOption) (Gripmock_BidirectionalClient, error) {
	stream, err := c.cc.NewStream(ctx, &_Gripmock_serviceDesc.Streams[2], "/main.Gripmock/bidirectional", opts...)
	if err != nil {
		return nil, err
	}
	x := &gripmockBidirectionalClient{stream}
	return x, nil
}

type Gripmock_BidirectionalClient interface {
	Send(*Request) error
	Recv() (*Reply, error)
	grpc.ClientStream
}

type gripmockBidirectionalClient struct {
	grpc.ClientStream
}

func (x *gripmockBidirectionalClient) Send(m *Request) error {
	return x.ClientStream.SendMsg(m)
}

func (x *gripmockBidirectionalClient) Recv() (*Reply, error) {
	m := new(Reply)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *gripmockClient) HealthCheck(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*wrappers.StringValue, error) {
	out := new(wrappers.StringValue)
	err := c.cc.Invoke(ctx, "/main.Gripmock/HealthCheck", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GripmockServer is the server API for Gripmock service.
type GripmockServer interface {
	// standard grpc method
	SayHello(context.Context, *Request) (*Reply, error)
	// server to client sreaming
	ServerStream(*Request, Gripmock_ServerStreamServer) error
	// client to server streaming
	ClientStream(Gripmock_ClientStreamServer) error
	// bidirectional streaming
	Bidirectional(Gripmock_BidirectionalServer) error
	// using well-known-type of Empty
	HealthCheck(context.Context, *empty.Empty) (*wrappers.StringValue, error)
}

func RegisterGripmockServer(s *grpc.Server, srv GripmockServer) {
	s.RegisterService(&_Gripmock_serviceDesc, srv)
}

func _Gripmock_SayHello_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GripmockServer).SayHello(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/main.Gripmock/SayHello",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GripmockServer).SayHello(ctx, req.(*Request))
	}
	return interceptor(ctx, in, info, handler)
}

func _Gripmock_ServerStream_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(Request)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(GripmockServer).ServerStream(m, &gripmockServerStreamServer{stream})
}

type Gripmock_ServerStreamServer interface {
	Send(*Reply) error
	grpc.ServerStream
}

type gripmockServerStreamServer struct {
	grpc.ServerStream
}

func (x *gripmockServerStreamServer) Send(m *Reply) error {
	return x.ServerStream.SendMsg(m)
}

func _Gripmock_ClientStream_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(GripmockServer).ClientStream(&gripmockClientStreamServer{stream})
}

type Gripmock_ClientStreamServer interface {
	SendAndClose(*Reply) error
	Recv() (*Request, error)
	grpc.ServerStream
}

type gripmockClientStreamServer struct {
	grpc.ServerStream
}

func (x *gripmockClientStreamServer) SendAndClose(m *Reply) error {
	return x.ServerStream.SendMsg(m)
}

func (x *gripmockClientStreamServer) Recv() (*Request, error) {
	m := new(Request)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _Gripmock_Bidirectional_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(GripmockServer).Bidirectional(&gripmockBidirectionalServer{stream})
}

type Gripmock_BidirectionalServer interface {
	Send(*Reply) error
	Recv() (*Request, error)
	grpc.ServerStream
}

type gripmockBidirectionalServer struct {
	grpc.ServerStream
}

func (x *gripmockBidirectionalServer) Send(m *Reply) error {
	return x.ServerStream.SendMsg(m)
}

func (x *gripmockBidirectionalServer) Recv() (*Request, error) {
	m := new(Request)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _Gripmock_HealthCheck_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(empty.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GripmockServer).HealthCheck(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/main.Gripmock/HealthCheck",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GripmockServer).HealthCheck(ctx, req.(*empty.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

var _Gripmock_serviceDesc = grpc.ServiceDesc{
	ServiceName: "main.Gripmock",
	HandlerType: (*GripmockServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SayHello",
			Handler:    _Gripmock_SayHello_Handler,
		},
		{
			MethodName: "HealthCheck",
			Handler:    _Gripmock_HealthCheck_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "serverStream",
			Handler:       _Gripmock_ServerStream_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "clientStream",
			Handler:       _Gripmock_ClientStream_Handler,
			ClientStreams: true,
		},
		{
			StreamName:    "bidirectional",
			Handler:       _Gripmock_Bidirectional_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "pb/hello.proto",
}

func init() { proto.RegisterFile("pb/hello.proto", fileDescriptor_hello_64609aa425866106) }

var fileDescriptor_hello_64609aa425866106 = []byte{
	// 257 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x84, 0x90, 0x4f, 0x4b, 0xc3, 0x40,
	0x14, 0xc4, 0x59, 0xa9, 0xb6, 0xbe, 0x5a, 0x0f, 0x7b, 0x90, 0x12, 0xff, 0xa0, 0x3d, 0x48, 0x0f,
	0xb2, 0x29, 0xfa, 0x11, 0x8a, 0xd8, 0x73, 0x02, 0xde, 0x37, 0xf1, 0x99, 0x2c, 0xdd, 0x7f, 0xee,
	0x6e, 0x94, 0xdc, 0xfd, 0xe0, 0xb2, 0x89, 0xb9, 0x28, 0xa4, 0xb7, 0xf7, 0x66, 0x7e, 0x0c, 0xc3,
	0xc0, 0xb9, 0x2d, 0xd2, 0x1a, 0xa5, 0x34, 0xcc, 0x3a, 0x13, 0x0c, 0x9d, 0x28, 0x2e, 0x74, 0x72,
	0x59, 0x19, 0x53, 0x49, 0x4c, 0x3b, 0xad, 0x68, 0xde, 0x53, 0x54, 0x36, 0xb4, 0x3d, 0x92, 0xdc,
	0xfc, 0x35, 0xbf, 0x1c, 0xb7, 0x16, 0x9d, 0xef, 0xfd, 0xd5, 0x35, 0x4c, 0x33, 0xfc, 0x68, 0xd0,
	0x07, 0x4a, 0x61, 0xa2, 0xb9, 0xc2, 0x25, 0xb9, 0x25, 0xeb, 0xd3, 0xac, 0xbb, 0x57, 0x77, 0x70,
	0x9c, 0xa1, 0x95, 0x2d, 0x5d, 0xc2, 0x54, 0xa1, 0xf7, 0xbc, 0x1a, 0xfc, 0xe1, 0x7d, 0xfc, 0x3e,
	0x82, 0xd9, 0x8b, 0x13, 0x56, 0x99, 0x72, 0x4f, 0xef, 0x61, 0x96, 0xf3, 0x76, 0x17, 0x3b, 0xd2,
	0x05, 0x8b, 0xf5, 0xd8, 0x6f, 0x7c, 0x32, 0x1f, 0xde, 0x18, 0xf7, 0x00, 0x67, 0x1e, 0xdd, 0x27,
	0xba, 0x3c, 0x38, 0xe4, 0x6a, 0x8c, 0xdd, 0x90, 0x48, 0x97, 0x52, 0xa0, 0x0e, 0x87, 0xe9, 0x35,
	0xa1, 0x29, 0x2c, 0x0a, 0xf1, 0x26, 0x1c, 0x96, 0x41, 0x18, 0xcd, 0xe5, 0x38, 0xbe, 0x21, 0x74,
	0x0b, 0xf3, 0x1d, 0x72, 0x19, 0xea, 0x6d, 0x8d, 0xe5, 0x9e, 0x5e, 0xb0, 0x7e, 0x33, 0x36, 0x6c,
	0xc6, 0x9e, 0xe3, 0xa0, 0xc9, 0xd5, 0x3f, 0x3d, 0x0f, 0x4e, 0xe8, 0xea, 0x95, 0xcb, 0x06, 0x8b,
	0x93, 0x4e, 0x7d, 0xfa, 0x09, 0x00, 0x00, 0xff, 0xff, 0x37, 0x73, 0xb5, 0x17, 0xa4, 0x01, 0x00,
	0x00,
}