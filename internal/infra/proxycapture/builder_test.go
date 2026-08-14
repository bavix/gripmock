package proxycapture_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/bavix/gripmock/v3/internal/infra/proxycapture"
)

func TestBuildServerStreamStub(t *testing.T) {
	t.Parallel()

	stub := proxycapture.BuildServerStreamStub(
		"test.Service",
		"ServerStreamMethod",
		"session-srv",
		map[string]any{"request": "data"},
		map[string]any{"req-header": "value"},
		[]any{map[string]any{"stream": 1}, map[string]any{"stream": 2}, map[string]any{"stream": 3}},
		proxycapture.ResponseMetadata{Trailers: map[string]string{"trailer": "value"}},
		nil,
	)

	require.Equal(t, "test.Service", stub.Service)
	require.Equal(t, "ServerStreamMethod", stub.Method)
	require.Len(t, stub.Output.Stream, 3)
}

// A replayed stub must put each captured value back where it arrived: a trailer
// stays a trailer, and a key present in both is not spliced into one string.
func TestCaptureMetadataKeepsHeadersAndTrailersApart(t *testing.T) {
	t.Parallel()

	meta := proxycapture.CaptureMetadata(
		metadata.Pairs("x-channel", "header", "content-type", "application/grpc"),
		metadata.Pairs("x-channel", "trailer", "x-audit", "done", "grpc-status", "0"),
	)

	require.Equal(t, "header", meta.Headers["x-channel"])
	require.Equal(t, "trailer", meta.Trailers["x-channel"])
	require.Equal(t, "done", meta.Trailers["x-audit"])
	require.NotContains(t, meta.Trailers, "grpc-status",
		"gRPC generates the status for every call; replaying it would fight the stub")
}

func TestCaptureMetadataEmpty(t *testing.T) {
	t.Parallel()

	meta := proxycapture.CaptureMetadata(nil, nil)

	require.Nil(t, meta.Headers)
	require.Nil(t, meta.Trailers)
}
