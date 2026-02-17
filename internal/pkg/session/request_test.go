package session_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/pkg/session"
)

func TestFromRequest_NilRequest(t *testing.T) {
	t.Parallel()

	// Act
	result := session.FromRequest(nil)

	// Assert
	require.Empty(t, result)
}

func TestFromRequest_PrefersHeader(t *testing.T) {
	t.Parallel()

	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(session.HeaderName, "  A  ")

	// Act
	result := session.FromRequest(req)

	// Assert
	require.Equal(t, "A", result)
}

func TestFromRequest_EmptyWhenNotProvided(t *testing.T) {
	t.Parallel()

	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	// Act
	result := session.FromRequest(req)

	// Assert
	require.Empty(t, result)
}
