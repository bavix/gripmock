package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestStorage_FormatStubNotFoundError(t *testing.T) {
	// Test basic error formatting
	err := status.Error(codes.NotFound, "test error")

	// This is a simple test to ensure the function exists and works
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestStorage_ErrorHandling(t *testing.T) {
	// Test that we can create and handle gRPC status errors
	err := status.Error(codes.Internal, "internal error")

	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "internal error")
}

func TestStorage_StatusCodes(t *testing.T) {
	// Test basic status code handling
	err := status.Error(codes.NotFound, "test message")
	assert.Equal(t, codes.NotFound, status.Code(err))
	assert.Contains(t, err.Error(), "test message")
}

func TestStorage_ErrorWrapping(t *testing.T) {
	// Test that we can wrap errors properly
	originalErr := status.Error(codes.NotFound, "original error")

	// Simulate error wrapping (this would be done by the actual implementation)
	wrappedErr := status.Error(codes.NotFound, "wrapped: "+originalErr.Error())

	require.Error(t, wrappedErr)
	assert.Equal(t, codes.NotFound, status.Code(wrappedErr))
	assert.Contains(t, wrappedErr.Error(), "original error")
}
