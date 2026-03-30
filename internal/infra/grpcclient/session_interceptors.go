package grpcclient

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const sessionHeader = "x-gripmock-session"

// UnarySessionInterceptor injects gripmock session header into unary calls.
func UnarySessionInterceptor(session string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req,
		reply any,
		conn *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if session != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, sessionHeader, session)
		}

		return invoker(ctx, method, req, reply, conn, opts...)
	}
}

// StreamSessionInterceptor injects gripmock session header into stream calls.
func StreamSessionInterceptor(session string) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		if session != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, sessionHeader, session)
		}

		return streamer(ctx, desc, cc, method, opts...)
	}
}
