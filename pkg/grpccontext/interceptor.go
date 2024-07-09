package grpccontext

import (
	"context"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// UnaryInterceptor is a gRPC interceptor that adds a logger to the context.
// The logger can be used to log messages related to the gRPC request.
//
// It takes a logger as a parameter and returns a grpc.UnaryServerInterceptor.
// The returned interceptor is used to intercept the gRPC unary requests.
func UnaryInterceptor(logger *zerolog.Logger) grpc.UnaryServerInterceptor {
	// The interceptor function is called for each gRPC unary request.
	// It takes the inner context, the request, the server info, and the handler.
	// It returns the response and an error.
	return func(
		innerCtx context.Context, // The context of the gRPC request.
		req interface{}, // The request object.
		_ *grpc.UnaryServerInfo, // The server info.
		handler grpc.UnaryHandler, // The handler function for the request.
	) (interface{}, error) {
		// Add the logger to the context and call the handler.
		// The logger can be accessed using grpc.GetLogger(ctx).
		return handler(logger.WithContext(innerCtx), req)
	}
}

type serverStreamWrapper struct {
	ss  grpc.ServerStream
	ctx context.Context //nolint:containedctx
}

func (w serverStreamWrapper) Context() context.Context        { return w.ctx }
func (w serverStreamWrapper) RecvMsg(msg interface{}) error   { return w.ss.RecvMsg(msg) }
func (w serverStreamWrapper) SendMsg(msg interface{}) error   { return w.ss.SendMsg(msg) }
func (w serverStreamWrapper) SendHeader(md metadata.MD) error { return w.ss.SendHeader(md) }
func (w serverStreamWrapper) SetHeader(md metadata.MD) error  { return w.ss.SetHeader(md) }
func (w serverStreamWrapper) SetTrailer(md metadata.MD)       { w.ss.SetTrailer(md) }

// StreamInterceptor is a gRPC interceptor that adds a logger to the context.
// The logger can be used to log messages related to the gRPC stream.
//
// It takes a logger as a parameter and returns a grpc.StreamServerInterceptor.
// The returned interceptor is used to intercept the gRPC stream requests.
func StreamInterceptor(logger *zerolog.Logger) grpc.StreamServerInterceptor {
	// The interceptor function is called for each gRPC stream request.
	// It takes the server, the stream, the server info, and the handler.
	// It returns an error.
	return func(
		srv interface{}, // The server object.
		ss grpc.ServerStream, // The stream object.
		_ *grpc.StreamServerInfo, // The server info.
		handler grpc.StreamHandler, // The handler function for the stream.
	) error {
		// Create a serverStreamWrapper object with the stream and context.
		// The context is created with the logger.
		return handler(srv, serverStreamWrapper{
			ss:  ss,
			ctx: logger.WithContext(ss.Context()),
		})
	}
}
