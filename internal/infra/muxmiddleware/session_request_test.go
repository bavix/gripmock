package muxmiddleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/muxmiddleware"
)

func TestFromRequestNilRequest(t *testing.T) {
	t.Parallel()

	// Act
	result := muxmiddleware.FromRequest(nil)

	// Assert
	require.Empty(t, result)
}

func TestFromRequestPrefersHeader(t *testing.T) {
	t.Parallel()

	// Arrange
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.Header.Set(muxmiddleware.HeaderName, "  A  ")

	// Act
	result := muxmiddleware.FromRequest(req)

	// Assert
	require.Equal(t, "A", result)
}

func TestFromRequestEmptyWhenNotProvided(t *testing.T) {
	t.Parallel()

	// Arrange
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

	// Act
	result := muxmiddleware.FromRequest(req)

	// Assert
	require.Empty(t, result)
}
