package app

import (
	"context"
	"net/http"
	"strings"

	"google.golang.org/grpc/metadata"
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
	result := make(map[string]any)
	count := 0

	forLowerConnectHeaders(hdr, func(lower string, values []string) {
		result[lower] = strings.Join(values, ";")
		count++
	})

	if count == 0 {
		return nil
	}

	return result
}

// httpHeadersToGRPCContext converts HTTP request headers into a gRPC
// incoming metadata and attaches it to ctx. The resulting context is then
// consumable by gRPC-style helpers (metadata.FromIncomingContext) used
// inside the mocker for stub matching (session, headers).
func httpHeadersToGRPCContext(ctx context.Context, hdr http.Header) context.Context {
	md := metadata.MD{}
	count := 0

	forLowerConnectHeaders(hdr, func(lower string, values []string) {
		md[lower] = append(md[lower], values...)
		count++
	})

	if count == 0 {
		return ctx
	}

	return metadata.NewIncomingContext(ctx, md)
}

// forLowerConnectHeaders iterates over hdr, lowercases each key, and
// calls fn for every header NOT in the connectExcludedHeaders set.
func forLowerConnectHeaders(hdr http.Header, fn func(lower string, values []string)) {
	for k, goval := range hdr {
		lower := strings.ToLower(k)
		if _, excluded := connectExcludedHeaders[lower]; excluded {
			continue
		}

		fn(lower, goval)
	}
}
