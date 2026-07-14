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

func startGatewayServer(t *testing.T) (string, func()) {
	t.Helper()

	lc := net.ListenConfig{}

	listener, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	addr := listener.Addr().String()
	require.NoError(t, listener.Close())

	cfg := config.Load()
	cfg.Gateway.Addr = addr

	builder := NewBuilder(WithConfig(cfg))

	ctx, cancel := context.WithCancel(t.Context())

	errCh := make(chan error, 1)

	go func() {
		errCh <- builder.GatewayServe(ctx)
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
			t.Fatalf("gateway server did not start within deadline: %v", dialErr)
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

func TestGatewayServe_RejectsMethodNotAllowed(t *testing.T) {
	t.Parallel()

	addr, teardown := startGatewayServer(t)
	defer teardown()

	resp, err := http.Get("http://" + addr + "/test.Service/TestMethod") //nolint:noctx
	require.NoError(t, err)

	defer resp.Body.Close() //nolint:errcheck

	require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

func TestGatewayServe_AcceptsPostToUnknownRoute(t *testing.T) {
	t.Parallel()

	addr, teardown := startGatewayServer(t)
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

// TestGatewayServe_ConnectRPCErrorFormat verifies that a request with
// Content-Type: application/json is routed to the ConnectRPC handler and
// returns a Connect-style JSON error (non-200 status, JSON body).
func TestGatewayServe_ConnectRPCErrorFormat(t *testing.T) {
	t.Parallel()

	addr, teardown := startGatewayServer(t)
	defer teardown()

	resp, err := http.Post( //nolint:noctx
		"http://"+addr+"/unknown.Service/UnknownMethod",
		"application/json",
		strings.NewReader("{}"),
	)
	require.NoError(t, err)

	defer resp.Body.Close() //nolint:errcheck

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	require.Equal(t, "application/connect+json", resp.Header.Get("Content-Type"))

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), `"code":"not_found"`)
}

// TestGatewayServe_GRPCWebRoutedByContentType verifies that a request with
// Content-Type: application/grpc-web+proto is routed to the GRPCWeb handler
// and returns a gRPC-web-style response (HTTP 200 with trailers).
func TestGatewayServe_GRPCWebRoutedByContentType(t *testing.T) {
	t.Parallel()

	addr, teardown := startGatewayServer(t)
	defer teardown()

	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"http://"+addr+"/unknown.Service/UnknownMethod",
		strings.NewReader("{}"),
	)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/grpc-web+proto")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close() //nolint:errcheck

	// gRPC-web always returns 200, status is in trailers
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/grpc-web+proto", resp.Header.Get("Content-Type"))

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Response should be a trailers frame (flag 0x80) with grpc-status
	require.GreaterOrEqual(t, len(body), 5, "expected at least a frame header")

	// First byte should be the trailers flag (0x80)
	require.Equal(t, byte(0x80), body[0], "expected gRPC-web trailers flag")
	require.Contains(t, string(body), "grpc-status")
}

// TestGatewayServe_CompressionRequestAccepted verifies that gzip-encoded
// requests are accepted and decoded by the GzipRequestMiddleware.
func TestGatewayServe_CompressionRequestAccepted(t *testing.T) {
	t.Parallel()

	addr, teardown := startGatewayServer(t)
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

// TestGatewayServe_CompressionResponse verifies that the client receives
// a gzipped response when Accept-Encoding: gzip is sent.
func TestGatewayServe_CompressionResponse(t *testing.T) {
	t.Parallel()

	addr, teardown := startGatewayServer(t)
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

// TestGatewayServe_RespectsContextCancellation verifies that the server
// is reachable while running and that context cancellation does not
// deadlock the calling goroutine.
func TestGatewayServe_RespectsContextCancellation(t *testing.T) {
	t.Parallel()

	addr, teardown := startGatewayServer(t)
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
