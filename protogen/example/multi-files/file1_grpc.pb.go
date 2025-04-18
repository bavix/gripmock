// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v6.30.2
// source: file1.proto

package multi_files

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
	Gripmock1_SayHello_FullMethodName = "/multifiles.Gripmock1/SayHello"
)

// Gripmock1Client is the client API for Gripmock1 service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// The Gripmock service definition.
type Gripmock1Client interface {
	// simple unary method
	SayHello(ctx context.Context, in *Request1, opts ...grpc.CallOption) (*Reply1, error)
}

type gripmock1Client struct {
	cc grpc.ClientConnInterface
}

func NewGripmock1Client(cc grpc.ClientConnInterface) Gripmock1Client {
	return &gripmock1Client{cc}
}

func (c *gripmock1Client) SayHello(ctx context.Context, in *Request1, opts ...grpc.CallOption) (*Reply1, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Reply1)
	err := c.cc.Invoke(ctx, Gripmock1_SayHello_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Gripmock1Server is the server API for Gripmock1 service.
// All implementations must embed UnimplementedGripmock1Server
// for forward compatibility.
//
// The Gripmock service definition.
type Gripmock1Server interface {
	// simple unary method
	SayHello(context.Context, *Request1) (*Reply1, error)
	mustEmbedUnimplementedGripmock1Server()
}

// UnimplementedGripmock1Server must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedGripmock1Server struct{}

func (UnimplementedGripmock1Server) SayHello(context.Context, *Request1) (*Reply1, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SayHello not implemented")
}
func (UnimplementedGripmock1Server) mustEmbedUnimplementedGripmock1Server() {}
func (UnimplementedGripmock1Server) testEmbeddedByValue()                   {}

// UnsafeGripmock1Server may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to Gripmock1Server will
// result in compilation errors.
type UnsafeGripmock1Server interface {
	mustEmbedUnimplementedGripmock1Server()
}

func RegisterGripmock1Server(s grpc.ServiceRegistrar, srv Gripmock1Server) {
	// If the following call pancis, it indicates UnimplementedGripmock1Server was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&Gripmock1_ServiceDesc, srv)
}

func _Gripmock1_SayHello_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Request1)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(Gripmock1Server).SayHello(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Gripmock1_SayHello_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(Gripmock1Server).SayHello(ctx, req.(*Request1))
	}
	return interceptor(ctx, in, info, handler)
}

// Gripmock1_ServiceDesc is the grpc.ServiceDesc for Gripmock1 service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Gripmock1_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "multifiles.Gripmock1",
	HandlerType: (*Gripmock1Server)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SayHello",
			Handler:    _Gripmock1_SayHello_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "file1.proto",
}
