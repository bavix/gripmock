package protoset

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGRPCHandlerCanHandle(t *testing.T) {
	t.Parallel()

	h := &GRPCHandler{}

	require.True(t, h.CanHandle("grpc://localhost:50051"))
	require.True(t, h.CanHandle("grpcs://api.company.local:443"))
	require.False(t, h.CanHandle("buf.build/acme/payments"))
	require.False(t, h.CanHandle("service.proto"))
}

func TestGRPCHandlerParseDefaultOptions(t *testing.T) {
	t.Parallel()

	h := &GRPCHandler{}

	src, err := h.Parse("grpc://localhost:50051")
	require.NoError(t, err)
	require.Equal(t, SourceReflect, src.Type)
	require.Equal(t, "localhost:50051", src.ReflectAddress)
	require.False(t, src.ReflectTLS)
	require.Equal(t, defaultReflectTimeout, src.ReflectTimeout)
	require.Empty(t, src.ReflectServerName)
	require.Empty(t, src.ReflectBearer)
}

func TestGRPCHandlerParseWithQuery(t *testing.T) {
	t.Parallel()

	h := &GRPCHandler{}

	src, err := h.Parse("grpcs://api.company.local:443?serverName=api.company.local&bearer=token&timeout=7s")
	require.NoError(t, err)
	require.Equal(t, SourceReflect, src.Type)
	require.Equal(t, "api.company.local:443", src.ReflectAddress)
	require.True(t, src.ReflectTLS)
	require.Equal(t, 7*time.Second, src.ReflectTimeout)
	require.Equal(t, "api.company.local", src.ReflectServerName)
	require.Equal(t, "token", src.ReflectBearer)
}

func TestGRPCHandlerParseErrors(t *testing.T) {
	t.Parallel()

	h := &GRPCHandler{}

	_, err := h.Parse("grpc://")
	require.ErrorContains(t, err, "host:port")

	_, err = h.Parse("grpc://localhost:50051/path")
	require.ErrorContains(t, err, "must not include path")

	_, err = h.Parse("grpcs://localhost:50051?timeout=oops")
	require.ErrorContains(t, err, "invalid timeout")
}
