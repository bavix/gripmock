package deps

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fullDuplexProbeHandler mimics a bidi stream: read one byte, flush a
// reply, read the next byte. On HTTP/1.x the second read only works when
// full duplex has been enabled before the first flush.
func fullDuplexProbeHandler(secondRead chan<- error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1)

		_, err := io.ReadFull(r.Body, buf)
		if err != nil {
			secondRead <- err

			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("a"))

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		_, err = io.ReadFull(r.Body, buf)
		secondRead <- err
	})
}

// runFullDuplexProbe sends a two-byte chunked body with a pause between
// the bytes and returns the error of the handler's second read.
func runFullDuplexProbe(t *testing.T, wrap func(http.Handler) http.Handler, contentType string) error {
	t.Helper()

	secondRead := make(chan error, 1)

	handler := fullDuplexProbeHandler(secondRead)
	if wrap != nil {
		handler = wrap(handler)
	}

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	pr, pw := io.Pipe()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL+"/svc/Method", pr)
	require.NoError(t, err)

	req.Header.Set("Content-Type", contentType)

	go func() {
		_, _ = pw.Write([]byte("1"))

		time.Sleep(100 * time.Millisecond)

		_, _ = pw.Write([]byte("2"))
		_ = pw.Close()
	}()

	resp, err := srv.Client().Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	_, _ = io.Copy(io.Discard, resp.Body)

	select {
	case err := <-secondRead:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not finish second read")

		return nil
	}
}

func TestGatewayFullDuplexMiddleware_KeepsBodyReadableAfterFlush(t *testing.T) {
	t.Parallel()

	for _, ct := range []string{
		"application/grpc-web+json",
		"application/grpc-web+proto",
		"application/grpc-web-text",
		"Application/Connect+JSON",
	} {
		require.NoError(t, runFullDuplexProbe(t, gatewayFullDuplexMiddleware, ct), ct)
	}
}

func TestGatewayFullDuplexMiddleware_WithoutItBodyClosesOnFlush(t *testing.T) {
	t.Parallel()

	err := runFullDuplexProbe(t, nil, "application/grpc-web+json")
	require.ErrorIs(t, err, http.ErrBodyReadAfterClose)
}

func TestGatewayFullDuplexMiddleware_SkipsNonStreamingContentType(t *testing.T) {
	t.Parallel()

	err := runFullDuplexProbe(t, gatewayFullDuplexMiddleware, "application/json")
	require.ErrorIs(t, err, http.ErrBodyReadAfterClose)
}

func TestIsStreamingGatewayContentType(t *testing.T) {
	t.Parallel()

	require.True(t, isStreamingGatewayContentType("application/grpc-web"))
	require.True(t, isStreamingGatewayContentType(" application/connect+proto "))
	require.False(t, isStreamingGatewayContentType("application/json"))
	require.False(t, isStreamingGatewayContentType(""))
}
