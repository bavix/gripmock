package app

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/bavix/features"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

const (
	mcpLeakEventuallyTimeout  = 2 * time.Second
	mcpLeakEventuallyInterval = 20 * time.Millisecond
)

func TestMcpMessage_NoGoroutineLeak_OnNotifications(t *testing.T) {
	t.Parallel()

	budgerigar := stuber.NewBudgerigar(features.New())
	server, err := NewRestServer(t.Context(), budgerigar, &mockExtender{}, nil, nil, nil)
	require.NoError(t, err)

	baseline := runtime.NumGoroutine()

	payload := map[string]any{"jsonrpc": "2.0", "method": "ping"}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	const iterations = 2000
	for range iterations {
		req := httptest.NewRequest(http.MethodPost, "/api/mcp", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		server.McpMessage(w, req)
		require.Equal(t, http.StatusNoContent, w.Code)
	}

	// Allow small jitter from runtime/test infrastructure.
	require.Eventually(t, func() bool {
		after := runtime.NumGoroutine()

		return after <= baseline+5
	}, mcpLeakEventuallyTimeout, mcpLeakEventuallyInterval)
}
