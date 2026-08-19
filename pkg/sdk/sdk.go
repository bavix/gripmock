// Package sdk provides an embedded gRPC mock server for tests.
package sdk

import "context"

// UnaryHandler processes a unary gRPC request and returns a response.
type UnaryHandler func(ctx context.Context, in any) (any, error)

// ServerStreamHandler processes a server-stream request.
type ServerStreamHandler func(ctx context.Context, in any, stream any) error

type ClientStreamHandler func(ctx context.Context, messages []any) (any, error)

// BidirectionalHandler processes a bidirectional stream.
type BidirectionalHandler func(ctx context.Context, stream any) error
