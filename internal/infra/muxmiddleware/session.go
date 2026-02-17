package muxmiddleware

import (
	"net/http"

	"github.com/bavix/gripmock/v3/internal/pkg/session"
)

// TransportSession moves X-Gripmock-Session into internal context and strips the header.
func TransportSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, session.ConsumeRequest(r))
	})
}
