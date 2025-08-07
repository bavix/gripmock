package muxmiddleware

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/bavix/gripmock/v3/internal/infra/jsondecoder"
)

// RequestLogger logs the request and response.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := zerolog.Ctx(r.Context())
		ww := &responseWriter{w: w, status: http.StatusOK}
		ip, err := getIP(r)
		start := time.Now()

		bodyBytes, _ := io.ReadAll(r.Body)

		_ = r.Body.Close()

		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		next.ServeHTTP(ww, r)

		event := logger.Info().
			Err(err).
			IPAddr("ip", ip).
			Str("method", r.Method).
			Str("url", r.URL.RequestURI()).
			Dur("elapsed", time.Since(start)).
			Str("ua", r.UserAgent()).
			Int("bytes", ww.bytesWritten).
			Int("code", ww.status)

		var result []any

		err = jsondecoder.UnmarshalSlice(bodyBytes, &result)
		if err == nil {
			event.RawJSON("input", bodyBytes)
		}

		event.Send()
	})
}
