package muxmiddleware

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := zerolog.Ctx(r.Context())
		ww := &responseWriter{w: w, status: http.StatusOK}
		ip, err := getIP(r)
		now := time.Now()

		next.ServeHTTP(ww, r)

		logger.
			Info().
			Err(err).
			IPAddr("ip", ip).
			Str("method", r.Method).
			Str("url", r.URL.RequestURI()).
			Dur("elapsed", time.Since(now)).
			Str("ua", r.UserAgent()).
			Int("bytes", ww.bytes).
			Int("code", ww.status).
			Send()
	})
}
