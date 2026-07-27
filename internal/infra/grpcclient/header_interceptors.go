package grpcclient

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// unaryHeaderInterceptor appends the given header to outgoing unary calls when
// value is non-empty.
func unaryHeaderInterceptor(key, value string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req,
		reply any,
		conn *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if value != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, key, value)
		}

		return invoker(ctx, method, req, reply, conn, opts...)
	}
}

// streamHeaderInterceptor appends the given header to outgoing stream calls when
// value is non-empty.
func streamHeaderInterceptor(key, value string) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		if value != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, key, value)
		}

		return streamer(ctx, desc, cc, method, opts...)
	}
}
