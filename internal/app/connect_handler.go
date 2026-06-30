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

// httpHeadersToGRPCContext converts HTTP request headers into a gRPC
// incoming metadata and attaches it to ctx. The resulting context is then
// consumable by gRPC-style helpers (metadata.FromIncomingContext) used
// inside the mocker for stub matching (session, headers).
func httpHeadersToGRPCContext(ctx context.Context, hdr http.Header) context.Context {
	if len(hdr) == 0 {
		return ctx
	}

	md := metadata.MD{}

	for k, goval := range hdr {
		lower := strings.ToLower(k)
		if _, excluded := connectExcludedHeaders[lower]; excluded {
			continue
		}

		md[lower] = append(md[lower], goval...)
	}

	if len(md) == 0 {
		return ctx
	}

	return metadata.NewIncomingContext(ctx, md)
}
