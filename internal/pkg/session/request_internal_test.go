package session_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/pkg/session"
)

func TestConsumeRequest_MovesHeaderToContext(t *testing.T) {
	t.Parallel()

	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(session.HeaderName, "  A  ")

	// Act
	consumed := session.ConsumeRequest(req)
	got := session.FromRequest(consumed)

	// Assert
	require.Equal(t, "A", got)
	require.Empty(t, consumed.Header.Get(session.HeaderName))
}

func TestConsumeRequest_DefaultGlobalAsEmptySession(t *testing.T) {
	t.Parallel()

	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	// Act
	consumed := session.ConsumeRequest(req)
	got := session.FromRequest(consumed)

	// Assert
	require.Empty(t, got)
}
