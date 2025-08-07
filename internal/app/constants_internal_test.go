package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstants(t *testing.T) {
	// Test gRPC server constants
	assert.NotEmpty(t, ServiceReflection)
	assert.NotEmpty(t, ErrMsgFailedToSendResponse)
	assert.NotEmpty(t, ErrMsgFailedToReceiveMessage)

	// Test excluded headers
	assert.NotNil(t, ExcludedHeaders)
	assert.Contains(t, ExcludedHeaders, "content-type")
	assert.Contains(t, ExcludedHeaders, ":authority")

	// Test logging fields
	assert.NotEmpty(t, LogFieldService)
	assert.NotEmpty(t, LogFieldMethod)
	assert.NotEmpty(t, LogFieldPeerAddress)
	assert.NotEmpty(t, LogFieldProtocol)

	// Test error messages
	assert.NotEmpty(t, ErrMsgFailedToFindStub)
	assert.NotEmpty(t, ErrMsgFailedToProcessMessage)
	assert.NotEmpty(t, ErrMsgFailedToMarshalData)
}

func TestExcludedHeadersContent(t *testing.T) {
	// Verify all expected headers are excluded
	expectedHeaders := []string{
		":authority",
		"content-type",
		"grpc-accept-encoding",
		"user-agent",
		"accept-encoding",
	}

	for _, header := range expectedHeaders {
		assert.Contains(t, ExcludedHeaders, header, "Header %s should be excluded", header)
	}
}

func TestLoggingFieldsFormat(t *testing.T) {
	// Test that logging fields are properly formatted
	assert.Equal(t, "service", LogFieldService)
	assert.Equal(t, "method", LogFieldMethod)
	assert.Equal(t, "peer.address", LogFieldPeerAddress)
	assert.Equal(t, "protocol", LogFieldProtocol)
}
