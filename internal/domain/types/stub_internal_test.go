package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMethodRegistry_RegisterAndRetrieve(t *testing.T) {
	t.Parallel()

	registry := NewMethodRegistry()

	// Test unary method
	registry.RegisterMethod("test.Service", "UnaryMethod", false, false)

	info, exists := registry.GetMethodInfo("test.Service", "UnaryMethod")
	require.True(t, exists)
	assert.True(t, info.IsUnary)
	assert.False(t, info.IsClientStream)
	assert.False(t, info.IsServerStream)
	assert.False(t, info.IsBidiStream)

	// Test client streaming method
	registry.RegisterMethod("test.Service", "ClientStreamMethod", true, false)

	info, exists = registry.GetMethodInfo("test.Service", "ClientStreamMethod")
	require.True(t, exists)
	assert.False(t, info.IsUnary)
	assert.True(t, info.IsClientStream)
	assert.False(t, info.IsServerStream)
	assert.False(t, info.IsBidiStream)

	// Test server streaming method
	registry.RegisterMethod("test.Service", "ServerStreamMethod", false, true)

	info, exists = registry.GetMethodInfo("test.Service", "ServerStreamMethod")
	require.True(t, exists)
	assert.False(t, info.IsUnary)
	assert.False(t, info.IsClientStream)
	assert.True(t, info.IsServerStream)
	assert.False(t, info.IsBidiStream)

	// Test bidirectional streaming method
	registry.RegisterMethod("test.Service", "BidiStreamMethod", true, true)

	info, exists = registry.GetMethodInfo("test.Service", "BidiStreamMethod")
	require.True(t, exists)
	assert.False(t, info.IsUnary)
	assert.True(t, info.IsClientStream)
	assert.True(t, info.IsServerStream)
	assert.True(t, info.IsBidiStream)
}

func TestMethodRegistry_HelperMethods(t *testing.T) {
	t.Parallel()

	registry := NewMethodRegistry()

	// Register test methods
	registry.RegisterMethod("test.Service", "UnaryMethod", false, false)
	registry.RegisterMethod("test.Service", "ClientStreamMethod", true, false)
	registry.RegisterMethod("test.Service", "ServerStreamMethod", false, true)
	registry.RegisterMethod("test.Service", "BidiStreamMethod", true, true)

	// Test IsUnary
	assert.True(t, registry.IsUnary("test.Service", "UnaryMethod"))
	assert.False(t, registry.IsUnary("test.Service", "ClientStreamMethod"))
	assert.False(t, registry.IsUnary("test.Service", "ServerStreamMethod"))
	assert.False(t, registry.IsUnary("test.Service", "BidiStreamMethod"))

	// Test IsClientStream
	assert.False(t, registry.IsClientStream("test.Service", "UnaryMethod"))
	assert.True(t, registry.IsClientStream("test.Service", "ClientStreamMethod"))
	assert.False(t, registry.IsClientStream("test.Service", "ServerStreamMethod"))
	assert.True(t, registry.IsClientStream("test.Service", "BidiStreamMethod"))

	// Test IsServerStream
	assert.False(t, registry.IsServerStream("test.Service", "UnaryMethod"))
	assert.False(t, registry.IsServerStream("test.Service", "ClientStreamMethod"))
	assert.True(t, registry.IsServerStream("test.Service", "ServerStreamMethod"))
	assert.True(t, registry.IsServerStream("test.Service", "BidiStreamMethod"))

	// Test IsBidiStream
	assert.False(t, registry.IsBidiStream("test.Service", "UnaryMethod"))
	assert.False(t, registry.IsBidiStream("test.Service", "ClientStreamMethod"))
	assert.False(t, registry.IsBidiStream("test.Service", "ServerStreamMethod"))
	assert.True(t, registry.IsBidiStream("test.Service", "BidiStreamMethod"))
}

func TestMethodRegistry_NonExistentMethod(t *testing.T) {
	t.Parallel()

	registry := NewMethodRegistry()

	// Test non-existent method
	info, exists := registry.GetMethodInfo("test.Service", "NonExistentMethod")
	assert.False(t, exists)
	assert.Equal(t, MethodInfo{}, info)

	// Test helper methods with non-existent method
	assert.False(t, registry.IsUnary("test.Service", "NonExistentMethod"))
	assert.False(t, registry.IsClientStream("test.Service", "NonExistentMethod"))
	assert.False(t, registry.IsServerStream("test.Service", "NonExistentMethod"))
	assert.False(t, registry.IsBidiStream("test.Service", "NonExistentMethod"))
}

func TestMethodRegistry_OverwriteMethod(t *testing.T) {
	t.Parallel()

	registry := NewMethodRegistry()

	// Register method as unary first
	registry.RegisterMethod("test.Service", "Method", false, false)
	assert.True(t, registry.IsUnary("test.Service", "Method"))

	// Overwrite as bidirectional
	registry.RegisterMethod("test.Service", "Method", true, true)
	assert.True(t, registry.IsBidiStream("test.Service", "Method"))
	assert.False(t, registry.IsUnary("test.Service", "Method"))
}

// Note: Stub type determination tests have been moved to infrastructure layer
// See internal/infra/stuber/stub_type_service_test.go

// Note: Stub type determination tests have been moved to infrastructure layer
// See internal/infra/stuber/stub_type_service_test.go
