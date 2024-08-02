// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v4.24.4
// source: stream.proto

package stream

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	Gripmock_ServerStream_FullMethodName  = "/stream.Gripmock/serverStream"
	Gripmock_ClientStream_FullMethodName  = "/stream.Gripmock/clientStream"
	Gripmock_Bidirectional_FullMethodName = "/stream.Gripmock/bidirectional"
)

// GripmockClient is the client API for Gripmock service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// The Gripmock service definition.
type GripmockClient interface {
	// server to client sreaming
	ServerStream(ctx context.Context, in *Request, opts ...grpc.CallOption) (grpc.ServerStreamingClient[Reply], error)
	// client to server streaming
	ClientStream(ctx context.Context, opts ...grpc.CallOption) (grpc.ClientStreamingClient[Request, Reply], error)
	// bidirectional streaming
	Bidirectional(ctx context.Context, opts ...grpc.CallOption) (grpc.BidiStreamingClient[Request, Reply], error)
}

type gripmockClient struct {
	cc grpc.ClientConnInterface
}

func NewGripmockClient(cc grpc.ClientConnInterface) GripmockClient {
	return &gripmockClient{cc}
}

func (c *gripmockClient) ServerStream(ctx context.Context, in *Request, opts ...grpc.CallOption) (grpc.ServerStreamingClient[Reply], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &Gripmock_ServiceDesc.Streams[0], Gripmock_ServerStream_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[Request, Reply]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Gripmock_ServerStreamClient = grpc.ServerStreamingClient[Reply]

func (c *gripmockClient) ClientStream(ctx context.Context, opts ...grpc.CallOption) (grpc.ClientStreamingClient[Request, Reply], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &Gripmock_ServiceDesc.Streams[1], Gripmock_ClientStream_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[Request, Reply]{ClientStream: stream}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Gripmock_ClientStreamClient = grpc.ClientStreamingClient[Request, Reply]

func (c *gripmockClient) Bidirectional(ctx context.Context, opts ...grpc.CallOption) (grpc.BidiStreamingClient[Request, Reply], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &Gripmock_ServiceDesc.Streams[2], Gripmock_Bidirectional_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[Request, Reply]{ClientStream: stream}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Gripmock_BidirectionalClient = grpc.BidiStreamingClient[Request, Reply]

// GripmockServer is the server API for Gripmock service.
// All implementations must embed UnimplementedGripmockServer
// for forward compatibility.
//
// The Gripmock service definition.
type GripmockServer interface {
	// server to client sreaming
	ServerStream(*Request, grpc.ServerStreamingServer[Reply]) error
	// client to server streaming
	ClientStream(grpc.ClientStreamingServer[Request, Reply]) error
	// bidirectional streaming
	Bidirectional(grpc.BidiStreamingServer[Request, Reply]) error
	mustEmbedUnimplementedGripmockServer()
}

// UnimplementedGripmockServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedGripmockServer struct{}

func (UnimplementedGripmockServer) ServerStream(*Request, grpc.ServerStreamingServer[Reply]) error {
	return status.Errorf(codes.Unimplemented, "method ServerStream not implemented")
}
func (UnimplementedGripmockServer) ClientStream(grpc.ClientStreamingServer[Request, Reply]) error {
	return status.Errorf(codes.Unimplemented, "method ClientStream not implemented")
}
func (UnimplementedGripmockServer) Bidirectional(grpc.BidiStreamingServer[Request, Reply]) error {
	return status.Errorf(codes.Unimplemented, "method Bidirectional not implemented")
}
func (UnimplementedGripmockServer) mustEmbedUnimplementedGripmockServer() {}
func (UnimplementedGripmockServer) testEmbeddedByValue()                  {}

// UnsafeGripmockServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to GripmockServer will
// result in compilation errors.
type UnsafeGripmockServer interface {
	mustEmbedUnimplementedGripmockServer()
}

func RegisterGripmockServer(s grpc.ServiceRegistrar, srv GripmockServer) {
	// If the following call pancis, it indicates UnimplementedGripmockServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&Gripmock_ServiceDesc, srv)
}

func _Gripmock_ServerStream_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(Request)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(GripmockServer).ServerStream(m, &grpc.GenericServerStream[Request, Reply]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Gripmock_ServerStreamServer = grpc.ServerStreamingServer[Reply]

func _Gripmock_ClientStream_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(GripmockServer).ClientStream(&grpc.GenericServerStream[Request, Reply]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Gripmock_ClientStreamServer = grpc.ClientStreamingServer[Request, Reply]

func _Gripmock_Bidirectional_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(GripmockServer).Bidirectional(&grpc.GenericServerStream[Request, Reply]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Gripmock_BidirectionalServer = grpc.BidiStreamingServer[Request, Reply]

// Gripmock_ServiceDesc is the grpc.ServiceDesc for Gripmock service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Gripmock_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "stream.Gripmock",
	HandlerType: (*GripmockServer)(nil),
	Methods:     []grpc.MethodDesc{},
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
	Metadata: "stream.proto",
}
