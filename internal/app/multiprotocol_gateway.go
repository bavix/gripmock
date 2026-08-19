package app

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/go-playground/validator/v10"

	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/infra/proxyroutes"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// MultiProtocolGateway dispatches to the ConnectRPC or gRPC-Web handler
// based on the request Content-Type. This lets both protocols share a
// single HTTP port.
//
//   - application/grpc-web+proto, application/grpc-web+json → gRPC-Web
//   - everything else (application/json, application/proto,
//     application/connect+proto, application/connect+json, …) → ConnectRPC
//
// Both handlers read mux.Vars from the request context, which the
// gorilla/mux router populates before calling ServeHTTP.
type MultiProtocolGateway struct {
	connect *ConnectRPCGateway
	grpcweb *GRPCWebGateway
}

func NewMultiProtocolGateway(
	ctx context.Context,
	budgerigar *stuber.Budgerigar,
	descriptorRegistry *descriptors.Registry,
	recorder history.Recorder,
	proxyRoutesRef *atomic.Pointer[proxyroutes.Registry],
	validator *validator.Validate,
	errorFormatter *ErrorFormatter,
) *MultiProtocolGateway {
	return &MultiProtocolGateway{
		connect: NewConnectRPCGateway(ctx, budgerigar, descriptorRegistry, recorder, proxyRoutesRef, validator, errorFormatter),
		grpcweb: NewGRPCWebGateway(ctx, budgerigar, descriptorRegistry, recorder, proxyRoutesRef, validator, errorFormatter),
	}
}

// RequireProtocolVersion applies only to Connect: gRPC-Web has no such header.
func (g *MultiProtocolGateway) RequireProtocolVersion(require bool) {
	g.connect.RequireProtocolVersion(require)
}

func (g *MultiProtocolGateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	ct := r.Header.Get("Content-Type")

	switch {
	case strings.HasPrefix(normalizeContentType(ct), "application/grpc-web"):
		g.grpcweb.ServeHTTP(w, r)

		return
	default:
		g.connect.ServeHTTP(w, r)
	}
}
