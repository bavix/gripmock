package muxmiddleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/muxmiddleware"
)

func TestConsumeRequestMovesHeaderToContext(t *testing.T) {
	t.Parallel()

	// Arrange
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.Header.Set(muxmiddleware.HeaderName, "  A  ")

	// Act
	consumed := muxmiddleware.ConsumeRequest(req)
	got := muxmiddleware.FromRequest(consumed)

	// Assert
	require.Equal(t, "A", got)
	require.Empty(t, consumed.Header.Get(muxmiddleware.HeaderName))
}

func TestConsumeRequestDefaultGlobalAsEmptySession(t *testing.T) {
	t.Parallel()

	// Arrange
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

	// Act
	consumed := muxmiddleware.ConsumeRequest(req)
	got := muxmiddleware.FromRequest(consumed)

	// Assert
	require.Empty(t, got)
}
