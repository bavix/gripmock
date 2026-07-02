package deps

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/config"
)

func startConnectServer(t *testing.T) (string, func()) {
	t.Helper()

	lc := net.ListenConfig{}

	listener, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	addr := listener.Addr().String()
	require.NoError(t, listener.Close())

	builder := NewBuilder(WithConfig(config.Config{
		ConnectAddr: addr,
	}))

	ctx, cancel := context.WithCancel(t.Context())

	errCh := make(chan error, 1)

	go func() {
		errCh <- builder.ConnectServe(ctx)
	}()

	deadline := time.Now().Add(2 * time.Second)

	dialer := net.Dialer{Timeout: 50 * time.Millisecond}

	for {
		conn, dialErr := dialer.DialContext(t.Context(), "tcp", addr)
		if dialErr == nil {
			_ = conn.Close()

			break
		}

		if time.Now().After(deadline) {
			cancel()
			t.Fatalf("ConnectRPC server did not start within deadline: %v", dialErr)
		}

		time.Sleep(20 * time.Millisecond)
	}

	teardown := func() {
		cancel()

		select {
		case <-errCh:
		case <-time.After(500 * time.Millisecond):
		}
	}

	return addr, teardown
}

func TestConnectServe_RejectsMethodNotAllowed(t *testing.T) {
	t.Parallel()

	addr, teardown := startConnectServer(t)
	defer teardown()

	resp, err := http.Get("http://" + addr + "/test.Service/TestMethod") //nolint:noctx
	require.NoError(t, err)

	defer resp.Body.Close() //nolint:errcheck

	require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

func TestConnectServe_AcceptsPostToUnknownRoute(t *testing.T) {
	t.Parallel()

	addr, teardown := startConnectServer(t)
	defer teardown()

	resp, err := http.Post( //nolint:noctx
		"http://"+addr+"/unknown.Service/UnknownMethod",
		"application/json",
		strings.NewReader("{}"),
	)
	require.NoError(t, err)

	defer resp.Body.Close() //nolint:errcheck

	require.NotEqual(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

// TestConnectServe_CompressionRequestAccepted verifies that gzip-encoded
// requests are accepted and decoded by the GzipRequestMiddleware.
func TestConnectServe_CompressionRequestAccepted(t *testing.T) {
	t.Parallel()

	addr, teardown := startConnectServer(t)
	defer teardown()

	body := gzipEncode(t, []byte("{}"))

	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"http://"+addr+"/unknown.Service/UnknownMethod",
		bytes.NewReader(body),
	)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close() //nolint:errcheck

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestConnectServe_CompressionResponse verifies that the client receives
// a gzipped response when Accept-Encoding: gzip is sent.
func TestConnectServe_CompressionResponse(t *testing.T) {
	t.Parallel()

	addr, teardown := startConnectServer(t)
	defer teardown()

	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"http://"+addr+"/unknown.Service/UnknownMethod",
		strings.NewReader("{}"),
	)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close() //nolint:errcheck

	require.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))

	decoded, err := gunzipResponse(resp)
	require.NoError(t, err)
	require.Contains(t, string(decoded), `"code":"not_found"`)
}

// TestConnectServe_RespectsContextCancellation verifies that the server
// is reachable while running and that context cancellation does not
// deadlock the calling goroutine.
func TestConnectServe_RespectsContextCancellation(t *testing.T) {
	t.Parallel()

	addr, teardown := startConnectServer(t)
	defer teardown()

	resp, err := http.Get("http://" + addr + "/") //nolint:noctx
	require.NoError(t, err)

	_ = resp.Body.Close()

	require.NotEqual(t, 0, resp.StatusCode)

	done := make(chan struct{})

	go func() {
		teardown()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("teardown did not return within 5s (likely deadlock)")
	}
}

func gzipEncode(t *testing.T, data []byte) []byte {
	t.Helper()

	var buf bytes.Buffer

	gw := gzip.NewWriter(&buf)
	_, err := gw.Write(data)
	require.NoError(t, err)
	require.NoError(t, gw.Close())

	return buf.Bytes()
}

func gunzipResponse(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	gr, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer gr.Close() //nolint:errcheck

	return io.ReadAll(gr)
}
