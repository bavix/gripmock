// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v6.30.0
// source: wkt.proto

package well_known_types

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	apipb "google.golang.org/protobuf/types/known/apipb"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	Gripmock_ApiInfo_FullMethodName   = "/well_known_types.Gripmock/ApiInfo"
	Gripmock_ApiInfoV2_FullMethodName = "/well_known_types.Gripmock/ApiInfoV2"
)

// GripmockClient is the client API for Gripmock service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type GripmockClient interface {
	// this shows us example on using WKT as dependency
	// api.proto in particular has go_package alias with semicolon
	// "google.golang.org/genproto/protobuf/api;api"
	ApiInfo(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*apipb.Api, error)
	ApiInfoV2(ctx context.Context, in *ApiInfoV2Request, opts ...grpc.CallOption) (*ApiInfoV2Response, error)
}

type gripmockClient struct {
	cc grpc.ClientConnInterface
}

func NewGripmockClient(cc grpc.ClientConnInterface) GripmockClient {
	return &gripmockClient{cc}
}

func (c *gripmockClient) ApiInfo(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*apipb.Api, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(apipb.Api)
	err := c.cc.Invoke(ctx, Gripmock_ApiInfo_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gripmockClient) ApiInfoV2(ctx context.Context, in *ApiInfoV2Request, opts ...grpc.CallOption) (*ApiInfoV2Response, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ApiInfoV2Response)
	err := c.cc.Invoke(ctx, Gripmock_ApiInfoV2_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GripmockServer is the server API for Gripmock service.
// All implementations must embed UnimplementedGripmockServer
// for forward compatibility.
type GripmockServer interface {
	// this shows us example on using WKT as dependency
	// api.proto in particular has go_package alias with semicolon
	// "google.golang.org/genproto/protobuf/api;api"
	ApiInfo(context.Context, *emptypb.Empty) (*apipb.Api, error)
	ApiInfoV2(context.Context, *ApiInfoV2Request) (*ApiInfoV2Response, error)
	mustEmbedUnimplementedGripmockServer()
}

// UnimplementedGripmockServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedGripmockServer struct{}

func (UnimplementedGripmockServer) ApiInfo(context.Context, *emptypb.Empty) (*apipb.Api, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ApiInfo not implemented")
}
func (UnimplementedGripmockServer) ApiInfoV2(context.Context, *ApiInfoV2Request) (*ApiInfoV2Response, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ApiInfoV2 not implemented")
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

func _Gripmock_ApiInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GripmockServer).ApiInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Gripmock_ApiInfo_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GripmockServer).ApiInfo(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _Gripmock_ApiInfoV2_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ApiInfoV2Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GripmockServer).ApiInfoV2(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Gripmock_ApiInfoV2_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GripmockServer).ApiInfoV2(ctx, req.(*ApiInfoV2Request))
	}
	return interceptor(ctx, in, info, handler)
}

// Gripmock_ServiceDesc is the grpc.ServiceDesc for Gripmock service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Gripmock_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "well_known_types.Gripmock",
	HandlerType: (*GripmockServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ApiInfo",
			Handler:    _Gripmock_ApiInfo_Handler,
		},
		{
			MethodName: "ApiInfoV2",
			Handler:    _Gripmock_ApiInfoV2_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "wkt.proto",
}
