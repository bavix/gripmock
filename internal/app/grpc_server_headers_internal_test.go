package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// mockServerStream mocks grpc.ServerStream for testing.
type mockServerStream struct {
	headers metadata.MD
}

func (m *mockServerStream) SetHeader(md metadata.MD) error {
	m.headers = md

	return nil
}

func (m *mockServerStream) SendHeader(md metadata.MD) error {
	m.headers = md

	return nil
}

func (m *mockServerStream) SetTrailer(md metadata.MD) {
	// Not used in current implementation
}

func (m *mockServerStream) Context() context.Context {
	return context.Background()
}

func (m *mockServerStream) SendMsg(any) error {
	return nil
}

func (m *mockServerStream) RecvMsg(any) error {
	return nil
}

func TestHandleOutputErrorWithHeaders(t *testing.T) {
	t.Parallel()

	// Test error with headers
	output := stuber.Output{
		Error: "Test error",
		Code:  &[]codes.Code{codes.Aborted}[0],
		Headers: map[string]string{
			"error-code": "TEST_ERROR",
			"message":    "Test error message",
		},
	}

	stream := &mockServerStream{}
	mocker := &grpcMocker{}

	// Test header setting
	err := mocker.setResponseHeadersAny(stream.Context(), stream, output.Headers)
	require.NoError(t, err)

	// Verify headers were set
	require.NotNil(t, stream.headers)
	require.Equal(t, "TEST_ERROR", stream.headers.Get("error-code")[0])
	require.Equal(t, "Test error message", stream.headers.Get("message")[0])

	// Test error handling
	err = mocker.handleOutputError(stream.Context(), stream, output)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Aborted, st.Code())
	require.Equal(t, "Test error", st.Message())
}

func TestHandleOutputErrorWithoutHeaders(t *testing.T) {
	t.Parallel()

	// Test error without headers
	output := stuber.Output{
		Error: "Simple error",
		Code:  &[]codes.Code{codes.InvalidArgument}[0],
	}

	stream := &mockServerStream{}
	mocker := &grpcMocker{}

	// Test error handling
	err := mocker.handleOutputError(stream.Context(), stream, output)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, st.Code())
	require.Equal(t, "Simple error", st.Message())

	// Verify no headers were set
	require.Nil(t, stream.headers)
}

func TestHandleOutputErrorSuccess(t *testing.T) {
	t.Parallel()

	// Test success case
	output := stuber.Output{
		Data: map[string]any{"result": "success"},
		Headers: map[string]string{
			"x-request-id": "req-123",
		},
	}

	stream := &mockServerStream{}
	mocker := &grpcMocker{}

	// Test header setting
	err := mocker.setResponseHeadersAny(stream.Context(), stream, output.Headers)
	require.NoError(t, err)

	// Verify headers were set
	require.NotNil(t, stream.headers)
	require.Equal(t, "req-123", stream.headers.Get("x-request-id")[0])

	// Test error handling (should not return error)
	err = mocker.handleOutputError(stream.Context(), stream, output)
	require.NoError(t, err)
}

func TestHandleOutputErrorNilCode(t *testing.T) {
	t.Parallel()

	// Test error with nil code
	output := stuber.Output{
		Error: "Error without code",
		Headers: map[string]string{
			"error-type": "validation",
		},
	}

	stream := &mockServerStream{}
	mocker := &grpcMocker{}

	// Test header setting
	err := mocker.setResponseHeadersAny(stream.Context(), stream, output.Headers)
	require.NoError(t, err)

	// Verify headers were set
	require.NotNil(t, stream.headers)
	require.Equal(t, "validation", stream.headers.Get("error-type")[0])

	// Test error handling
	err = mocker.handleOutputError(stream.Context(), stream, output)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Aborted, st.Code()) // Default code for nil
	require.Equal(t, "Error without code", st.Message())
}
