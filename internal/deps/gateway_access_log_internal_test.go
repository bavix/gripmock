package deps

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGatewayAccessLog_DoesNotTruncateRequestBody(t *testing.T) {
	t.Parallel()

	const bodySize = 100_000

	var (
		seen    int
		readErr error
	)

	handler := gatewayAccessLogMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		readErr = err
		seen = len(body)
	}))

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/pkg.Service/Method",
		strings.NewReader(strings.Repeat("a", bodySize)),
	)
	req.Header.Set("Content-Type", "application/json")

	handler.ServeHTTP(httptest.NewRecorder(), req)

	require.NoError(t, readErr)
	require.Equal(t, bodySize, seen)
}

func TestGatewayAccessLog_BodyIsStreamedNotPreRead(t *testing.T) {
	t.Parallel()

	reader, writer := io.Pipe()
	entered := make(chan struct{})
	done := make(chan struct{})

	handler := gatewayAccessLogMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(entered)

		_, _ = io.Copy(io.Discard, r.Body)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/pkg.Service/Method", reader)
	req.Header.Set("Content-Type", "application/connect+proto")

	go func() {
		defer close(done)

		handler.ServeHTTP(httptest.NewRecorder(), req)
	}()

	<-entered

	_, _ = writer.Write([]byte("frame"))
	require.NoError(t, writer.Close())
	<-done
}

func TestCaptureResponseWriter_IsFlusher(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	writer := &captureResponseWriter{ResponseWriter: rec, statusCode: http.StatusOK}

	flusher, ok := any(writer).(http.Flusher)
	require.True(t, ok, "captureResponseWriter must implement http.Flusher")

	_, err := writer.Write([]byte("chunk"))
	require.NoError(t, err)
	flusher.Flush()

	require.True(t, rec.Flushed)
}

func TestCaptureReqBody_CapturesBoundedPrefix(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/pkg.Service/Method",
		strings.NewReader(strings.Repeat("b", maxBodyCapture*3)),
	)

	capture := captureReqBody(req)

	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	require.Len(t, body, maxBodyCapture*3)

	logged := capture.String()
	require.Equal(t, strings.Repeat("b", maxBodyCapture)+"...", logged)
}
