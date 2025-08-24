package types

import "context"

// Executor provides high-performance execution of stubs.
type Executor interface {
	Execute(ctx context.Context, stub StubStrict, headers map[string]any, requests []map[string]any, w Writer) (bool, error)
}

// Writer interface for writing responses.
type Writer interface {
	Send(data map[string]any) error
	SetHeaders(headers map[string]string) error
	SetTrailers(trailers map[string]string) error
	End(status *GrpcStatus) error
}
