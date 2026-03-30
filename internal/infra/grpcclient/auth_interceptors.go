package grpcclient

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// UnaryBearerInterceptor injects Authorization bearer token for unary client calls.
func UnaryBearerInterceptor(token string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req,
		reply any,
		conn *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if token != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
		}

		return invoker(ctx, method, req, reply, conn, opts...)
	}
}

// StreamBearerInterceptor injects Authorization bearer token for stream client calls.
func StreamBearerInterceptor(token string) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		if token != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
		}

		return streamer(ctx, desc, cc, method, opts...)
	}
}
