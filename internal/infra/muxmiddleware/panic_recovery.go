package muxmiddleware

import (
	"net/http"

	"github.com/rs/zerolog"
)

// PanicRecoveryMiddleware recovers from panics in HTTP handlers.
func PanicRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		defer func() {
			if panicValue := recover(); panicValue != nil {
				zerolog.Ctx(ctx).
					Error().
					Interface("panic", panicValue).
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Msg("Panic recovered in HTTP handler")

				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
