package grpccontext //nolint:testpackage

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestUnaryInterceptor(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	interceptor := UnaryInterceptor(&logger)

	// Test successful request
	req := "test request"
	resp := "test response"

	handler := func(ctx context.Context, req any) (any, error) {
		// Verify logger is in context
		ctxLogger := zerolog.Ctx(ctx)
		assert.NotNil(t, ctxLogger)

		return resp, nil
	}

	result, err := interceptor(context.Background(), req, nil, handler)
	require.NoError(t, err)
	assert.Equal(t, resp, result)
}

func TestStreamInterceptor(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	interceptor := StreamInterceptor(&logger)

	// Create a mock server stream
	mockStream := &mockServerStream{
		ctx: context.Background(),
	}

	// Test successful stream
	handler := func(srv any, stream grpc.ServerStream) error {
		// Verify logger is in context
		ctxLogger := zerolog.Ctx(stream.Context())
		assert.NotNil(t, ctxLogger)

		return nil
	}

	err := interceptor(nil, mockStream, nil, handler)
	require.NoError(t, err)
}

func TestServerStreamWrapper(t *testing.T) {
	originalCtx := context.Background()
	mockStream := &mockServerStream{ctx: originalCtx}

	wrapper := serverStreamWrapper{
		ss:  mockStream,
		ctx: originalCtx,
	}

	// Test Context method
	ctx := wrapper.Context()
	assert.Equal(t, originalCtx, ctx)

	// Test RecvMsg method
	msg := "test message"
	err := wrapper.RecvMsg(&msg)
	require.NoError(t, err)

	// Test SendMsg method
	err = wrapper.SendMsg(msg)
	require.NoError(t, err)

	// Test SendHeader method
	md := metadata.New(map[string]string{"key": "value"})
	err = wrapper.SendHeader(md)
	require.NoError(t, err)

	// Test SetHeader method
	err = wrapper.SetHeader(md)
	require.NoError(t, err)

	// Test SetTrailer method
	wrapper.SetTrailer(md)
	// No assertion needed as SetTrailer doesn't return error
}

// Mock server stream for testing.
type mockServerStream struct {
	grpc.ServerStream

	ctx context.Context //nolint:containedctx
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}

func (m *mockServerStream) RecvMsg(msg any) error {
	return nil
}

func (m *mockServerStream) SendMsg(msg any) error {
	return nil
}

func (m *mockServerStream) SendHeader(md metadata.MD) error {
	return nil
}

func (m *mockServerStream) SetHeader(md metadata.MD) error {
	return nil
}

func (m *mockServerStream) SetTrailer(md metadata.MD) {
	// Mock implementation
}
