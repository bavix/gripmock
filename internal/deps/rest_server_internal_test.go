package deps_test

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/config"
	"github.com/bavix/gripmock/v3/internal/deps"
)

func TestRestServeAssignsActualPortForHTTPAddrZero(t *testing.T) {
	t.Parallel()

	cfg := config.Load()
	cfg.HTTPAddr = "127.0.0.1:0"

	builder := deps.NewBuilder(deps.WithConfig(cfg))
	srv, err := builder.RestServe(t.Context(), "")
	require.NoError(t, err)

	_, port, splitErr := net.SplitHostPort(srv.Addr())
	require.NoError(t, splitErr)
	require.NotEqual(t, "0", port)

	errCh := make(chan error, 1)

	go func() {
		errCh <- srv.ListenAndServe()
	}()

	require.Eventually(t, func() bool {
		dialCtx, dialCancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
		defer dialCancel()

		conn, dialErr := (&net.Dialer{Timeout: 200 * time.Millisecond}).DialContext(dialCtx, "tcp", srv.Addr())
		if dialErr != nil {
			return false
		}

		_ = conn.Close()

		return true
	}, time.Second, 50*time.Millisecond)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()

	require.NoError(t, srv.Shutdown(shutdownCtx))
	require.ErrorIs(t, <-errCh, http.ErrServerClosed)
}

func TestRestServeReturnsErrorWhenHTTPPortIsBusy(t *testing.T) {
	t.Parallel()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = listener.Close()
	})

	cfg := config.Load()
	cfg.HTTPAddr = listener.Addr().String()

	builder := deps.NewBuilder(deps.WithConfig(cfg))
	srv, serveErr := builder.RestServe(t.Context(), "")
	require.Error(t, serveErr)
	require.Nil(t, srv)
}
