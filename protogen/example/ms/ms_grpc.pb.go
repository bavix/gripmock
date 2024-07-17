// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.4.0
// - protoc             v5.27.2
// source: ms.proto

package ms

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.62.0 or later.
const _ = grpc.SupportPackageIsVersion8

const (
	MicroService_SayHello_FullMethodName = "/ms.MicroService/SayHello"
)

// MicroServiceClient is the client API for MicroService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type MicroServiceClient interface {
	SayHello(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Reply, error)
}

type microServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewMicroServiceClient(cc grpc.ClientConnInterface) MicroServiceClient {
	return &microServiceClient{cc}
}

func (c *microServiceClient) SayHello(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Reply, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Reply)
	err := c.cc.Invoke(ctx, MicroService_SayHello_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MicroServiceServer is the server API for MicroService service.
// All implementations must embed UnimplementedMicroServiceServer
// for forward compatibility
type MicroServiceServer interface {
	SayHello(context.Context, *Request) (*Reply, error)
	mustEmbedUnimplementedMicroServiceServer()
}

// UnimplementedMicroServiceServer must be embedded to have forward compatible implementations.
type UnimplementedMicroServiceServer struct {
}

func (UnimplementedMicroServiceServer) SayHello(context.Context, *Request) (*Reply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SayHello not implemented")
}
func (UnimplementedMicroServiceServer) mustEmbedUnimplementedMicroServiceServer() {}

// UnsafeMicroServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to MicroServiceServer will
// result in compilation errors.
type UnsafeMicroServiceServer interface {
	mustEmbedUnimplementedMicroServiceServer()
}

func RegisterMicroServiceServer(s grpc.ServiceRegistrar, srv MicroServiceServer) {
	s.RegisterService(&MicroService_ServiceDesc, srv)
}

func _MicroService_SayHello_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MicroServiceServer).SayHello(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: MicroService_SayHello_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MicroServiceServer).SayHello(ctx, req.(*Request))
	}
	return interceptor(ctx, in, info, handler)
}

// MicroService_ServiceDesc is the grpc.ServiceDesc for MicroService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var MicroService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "ms.MicroService",
	HandlerType: (*MicroServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SayHello",
			Handler:    _MicroService_SayHello_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "ms.proto",
}
