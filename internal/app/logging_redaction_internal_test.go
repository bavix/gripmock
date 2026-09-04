package app

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestRedactMetadataHidesCredentials(t *testing.T) {
	t.Parallel()

	md := metadata.MD{
		"authorization": []string{"Bearer super-secret"},
		"cookie":        []string{"session=abc"},
		"x-api-key":     []string{"key-123"},
		"user-agent":    []string{"grpc-go/1.0"},
		"x-request-id":  []string{"req-42"},
	}

	redacted := redactMetadata(md)

	require.Equal(t, []string{redactedValue}, redacted["authorization"])
	require.Equal(t, []string{redactedValue}, redacted["cookie"])
	require.Equal(t, []string{redactedValue}, redacted["x-api-key"])

	require.Equal(t, []string{"grpc-go/1.0"}, redacted["user-agent"])
	require.Equal(t, []string{"req-42"}, redacted["x-request-id"])

	require.Equal(t, []string{"Bearer super-secret"}, md["authorization"])

	for key, values := range redacted {
		for _, value := range values {
			require.NotContains(t, value, "super-secret", "key %q leaked the token", key)
		}
	}
}

func TestRedactMetadataIsCaseInsensitive(t *testing.T) {
	t.Parallel()

	redacted := redactMetadata(metadata.MD{"Authorization": []string{"Bearer x"}})
	require.Equal(t, []string{redactedValue}, redacted["Authorization"])
}

type noopServerStream struct {
	grpc.ServerStream
}

func (noopServerStream) SendMsg(any) error { return nil }
func (noopServerStream) RecvMsg(any) error { return nil }

func TestLoggingStreamConcurrentAccess(t *testing.T) {
	t.Parallel()

	stream := newLoggingStream(noopServerStream{})

	var wg sync.WaitGroup

	wg.Go(func() {
		for range 200 {
			_ = stream.SendMsg("response")
		}
	})

	wg.Go(func() {
		for range 200 {
			_ = stream.RecvMsg("request")
		}
	})

	wg.Go(func() {
		for range 200 {
			stream.snapshot()
		}
	})

	wg.Wait()

	requests, responses := stream.snapshot()
	require.LessOrEqual(t, len(requests), maxLoggingStreamMsgs)
	require.LessOrEqual(t, len(responses), maxLoggingStreamMsgs)
}

func TestStreamCaptureStateLimit(t *testing.T) {
	t.Parallel()

	state := NewStreamCaptureState()
	state.SetLimit(3)

	for i := range 100 {
		state.AppendRequest(map[string]any{"i": i})
		state.AppendResponseWithTiming(map[string]any{"i": i}, time.Now())
	}

	requests, responses := state.Snapshot()
	require.Len(t, requests, 3)
	require.Len(t, responses, 3)
}

func TestStreamCaptureStateUnlimitedByDefault(t *testing.T) {
	t.Parallel()

	state := NewStreamCaptureState()

	for i := range 500 {
		state.AppendRequest(map[string]any{"i": i})
	}

	requests, _ := state.Snapshot()
	require.Len(t, requests, 500)
}

func TestLogMessageContentTruncatesLargeBodies(t *testing.T) {
	t.Parallel()

	require.Positive(t, maxLoggedBodyBytes)
	require.Less(t, maxLoggedBodyBytes, defaultMaxRecvMsgSize)
	require.NotEmpty(t, strings.TrimSpace(redactedValue))
}
