// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v4.23.4
// source: file2.proto

package multi_files

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	Gripmock2_SayHello_FullMethodName = "/multifiles.Gripmock2/SayHello"
)

// Gripmock2Client is the client API for Gripmock2 service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type Gripmock2Client interface {
	// simple unary method
	SayHello(ctx context.Context, in *Request2, opts ...grpc.CallOption) (*Reply2, error)
}

type gripmock2Client struct {
	cc grpc.ClientConnInterface
}

func NewGripmock2Client(cc grpc.ClientConnInterface) Gripmock2Client {
	return &gripmock2Client{cc}
}

func (c *gripmock2Client) SayHello(ctx context.Context, in *Request2, opts ...grpc.CallOption) (*Reply2, error) {
	out := new(Reply2)
	err := c.cc.Invoke(ctx, Gripmock2_SayHello_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Gripmock2Server is the server API for Gripmock2 service.
// All implementations must embed UnimplementedGripmock2Server
// for forward compatibility
type Gripmock2Server interface {
	// simple unary method
	SayHello(context.Context, *Request2) (*Reply2, error)
	mustEmbedUnimplementedGripmock2Server()
}

// UnimplementedGripmock2Server must be embedded to have forward compatible implementations.
type UnimplementedGripmock2Server struct {
}

func (UnimplementedGripmock2Server) SayHello(context.Context, *Request2) (*Reply2, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SayHello not implemented")
}
func (UnimplementedGripmock2Server) mustEmbedUnimplementedGripmock2Server() {}

// UnsafeGripmock2Server may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to Gripmock2Server will
// result in compilation errors.
type UnsafeGripmock2Server interface {
	mustEmbedUnimplementedGripmock2Server()
}

func RegisterGripmock2Server(s grpc.ServiceRegistrar, srv Gripmock2Server) {
	s.RegisterService(&Gripmock2_ServiceDesc, srv)
}

func _Gripmock2_SayHello_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Request2)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(Gripmock2Server).SayHello(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Gripmock2_SayHello_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(Gripmock2Server).SayHello(ctx, req.(*Request2))
	}
	return interceptor(ctx, in, info, handler)
}

// Gripmock2_ServiceDesc is the grpc.ServiceDesc for Gripmock2 service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Gripmock2_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "multifiles.Gripmock2",
	HandlerType: (*Gripmock2Server)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SayHello",
			Handler:    _Gripmock2_SayHello_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "file2.proto",
}