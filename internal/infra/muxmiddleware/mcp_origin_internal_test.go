package muxmiddleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func serveWithGuard(t *testing.T, allowed []string, origin string) int {
	t.Helper()

	handler := MCPOriginGuard(allowed)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/mcp", nil)

	if origin != "" {
		request.Header.Set("Origin", origin)
	}

	handler.ServeHTTP(recorder, request)

	return recorder.Code
}

func TestMCPOriginGuardRejectsForeignOrigins(t *testing.T) {
	t.Parallel()

	for _, origin := range []string{
		"http://evil.example.com",
		"https://attacker.test",
		"http://192.168.1.10:4771",
	} {
		require.Equal(t, http.StatusForbidden, serveWithGuard(t, []string{"*"}, origin), origin)
	}
}

func TestMCPOriginGuardAllowsLocalAndHeaderless(t *testing.T) {
	t.Parallel()

	require.Equal(t, http.StatusOK, serveWithGuard(t, []string{"*"}, ""),
		"curl and SDK clients send no Origin")

	for _, origin := range []string{
		"http://localhost:3000",
		"http://127.0.0.1:4771",
		"https://localhost",
		"http://[::1]:8080",
	} {
		require.Equal(t, http.StatusOK, serveWithGuard(t, []string{"*"}, origin), origin)
	}
}

func TestMCPOriginGuardHonoursExplicitAllowList(t *testing.T) {
	t.Parallel()

	require.Equal(t, http.StatusOK,
		serveWithGuard(t, []string{"http://studio.example.com"}, "http://studio.example.com"))
	require.Equal(t, http.StatusForbidden,
		serveWithGuard(t, []string{"http://studio.example.com"}, "http://other.example.com"))
	require.Equal(t, http.StatusForbidden,
		serveWithGuard(t, []string{"*"}, "http://studio.example.com"),
		"a wildcard list names no origin")
}
