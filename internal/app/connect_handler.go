package app

import (
	"net/http"
	"strings"
)

//nolint:gochecknoglobals
var connectExcludedHeaders = map[string]struct{}{
	"accept":                   {},
	"accept-encoding":          {},
	"content-encoding":         {},
	"content-length":           {},
	"content-type":             {},
	"connect-protocol-version": {},
	"connect-timeout-ms":       {},
	"user-agent":               {},
}

type connectMethodNotFoundError struct {
	service string
	method  string
}

func (e *connectMethodNotFoundError) Error() string {
	return "unknown service/method: " + e.service + "/" + e.method
}

func extractConnectHeaders(hdr http.Header) map[string]any {
	if len(hdr) == 0 {
		return nil
	}

	result := make(map[string]any, len(hdr))

	for k, goval := range hdr {
		lower := strings.ToLower(k)
		if _, excluded := connectExcludedHeaders[lower]; excluded {
			continue
		}

		result[lower] = strings.Join(goval, ";")
	}

	if len(result) == 0 {
		return nil
	}

	return result
}
