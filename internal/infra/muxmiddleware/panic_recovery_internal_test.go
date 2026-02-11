package muxmiddleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestPanicRecoveryMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("handler completes normally", func(t *testing.T) {
		t.Parallel()

		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})

		mw := PanicRecoveryMiddleware(next)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(zerolog.New(nil).WithContext(req.Context()))
		rec := httptest.NewRecorder()

		mw.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "ok", rec.Body.String())
	})

	t.Run("handler panics returns 500", func(t *testing.T) {
		t.Parallel()

		next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			panic("test panic")
		})

		mw := PanicRecoveryMiddleware(next)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(zerolog.New(nil).WithContext(req.Context()))
		rec := httptest.NewRecorder()

		mw.ServeHTTP(rec, req)

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		require.Contains(t, rec.Body.String(), "Internal Server Error")
	})
}
