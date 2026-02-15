package muxmiddleware

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/bavix/gripmock/v3/internal/infra/httputil"
	"github.com/bavix/gripmock/v3/internal/infra/jsondecoder"
)

const logBodyLimit = 4 << 10 // 4KB max body for structured logging

// RequestLogger logs the request and response. Uses pooled buffer for body read.
// Place after MaxBodySize middleware. Body is read once and replayed to handlers.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := zerolog.Ctx(r.Context())
		ww := &responseWriter{w: w, status: http.StatusOK}
		ip, err := getIP(r)
		start := time.Now()

		bodyBytes, readErr := httputil.ReadBody(r)
		if readErr != nil {
			r.Body = io.NopCloser(bytes.NewReader(nil))
		} else {
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			r = r.WithContext(httputil.ContextWithBody(r.Context(), bodyBytes))
		}

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

		if len(bodyBytes) <= logBodyLimit {
			var result []any

			if jsondecoder.UnmarshalSlice(bodyBytes, &result) == nil {
				event.RawJSON("input", bodyBytes)
			}
		}

		event.Send()
	})
}
