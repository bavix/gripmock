package muxmiddleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransportSessionMovesHeaderToContextAndStripsHeader(t *testing.T) {
	t.Parallel()

	// Arrange
	var (
		gotSession string
		gotHeader  string
	)

	h := TransportSession(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotSession = FromRequest(r)
		gotHeader = r.Header.Get(HeaderName)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.Header.Set(HeaderName, "A")

	w := httptest.NewRecorder()

	// Act
	h.ServeHTTP(w, req)

	// Assert
	require.Equal(t, "A", gotSession)
	require.Empty(t, gotHeader)
}
