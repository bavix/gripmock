package protoset

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestProxyHandlerCanHandle(t *testing.T) {
	t.Parallel()

	h := &ProxyHandler{}

	require.True(t, h.CanHandle("grpc+proxy://localhost:50051"))
	require.True(t, h.CanHandle("grpcs+proxy://api.local:443"))
	require.True(t, h.CanHandle("grpc+replay://localhost:50051"))
	require.True(t, h.CanHandle("grpcs+replay://api.local:443"))
	require.True(t, h.CanHandle("grpc+capture://localhost:50051"))
	require.True(t, h.CanHandle("grpcs+capture://api.local:443"))
	require.False(t, h.CanHandle("grpc://localhost:50051"))
	require.False(t, h.CanHandle("service.proto"))
}

func TestProxyHandlerParse(t *testing.T) {
	t.Parallel()

	h := &ProxyHandler{}

	src, err := h.Parse("grpcs+capture://api.company.local:443?serverName=api.company.local&bearer=token&timeout=7s&insecureSkipVerify=true")
	require.NoError(t, err)
	require.Equal(t, SourceReflect, src.Type)
	require.Equal(t, "capture", src.ProxyMode)
	require.Equal(t, "api.company.local:443", src.ReflectAddress)
	require.True(t, src.ReflectTLS)
	require.Equal(t, 7*time.Second, src.ReflectTimeout)
	require.Equal(t, "api.company.local", src.ReflectServerName)
	require.Equal(t, "token", src.ReflectBearer)
	require.True(t, src.ReflectInsecure)
}

func TestProxyHandlerParseErrors(t *testing.T) {
	t.Parallel()

	h := &ProxyHandler{}

	_, err := h.Parse("grpc+proxy://")
	require.ErrorContains(t, err, "host:port")

	_, err = h.Parse("grpc+proxy://localhost:50051/path")
	require.ErrorContains(t, err, "must not include path")

	_, err = h.Parse("grpcs+capture://localhost:50051?timeout=oops")
	require.ErrorContains(t, err, "invalid timeout")

	_, err = h.Parse("grpcs+proxy://localhost:50051?insecureSkipVerify=oops")
	require.ErrorContains(t, err, "insecureSkipVerify")
}
