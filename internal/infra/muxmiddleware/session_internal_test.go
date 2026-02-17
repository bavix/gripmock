package muxmiddleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/pkg/session"
)

func TestTransportSession_MovesHeaderToContextAndStripsHeader(t *testing.T) {
	t.Parallel()

	// Arrange
	var (
		gotSession string
		gotHeader  string
	)

	h := TransportSession(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotSession = session.FromRequest(r)
		gotHeader = r.Header.Get(session.HeaderName)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(session.HeaderName, "A")

	w := httptest.NewRecorder()

	// Act
	h.ServeHTTP(w, req)

	// Assert
	require.Equal(t, "A", gotSession)
	require.Empty(t, gotHeader)
}
