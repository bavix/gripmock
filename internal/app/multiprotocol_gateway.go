package app

import (
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"google.golang.org/grpc/codes"

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
	budgerigar *stuber.Budgerigar,
	descriptorRegistry *descriptors.Registry,
	recorder history.Recorder,
	proxies *proxyroutes.Registry,
	validator *validator.Validate,
	errorFormatter *ErrorFormatter,
) *MultiProtocolGateway {
	return &MultiProtocolGateway{
		connect: NewConnectRPCGateway(budgerigar, descriptorRegistry, recorder, proxies, validator, errorFormatter),
		grpcweb: NewGRPCWebGateway(budgerigar, descriptorRegistry, recorder, proxies, validator, errorFormatter),
	}
}

func (g *MultiProtocolGateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	ct := r.Header.Get("Content-Type")

	switch {
	case strings.HasPrefix(ct, "application/grpc-web-text"):
		writeGRPCWebError(w, codes.Unimplemented,
			"grpc-web-text (base64) encoding is not supported; use application/grpc-web+proto or application/grpc-web+json")

		return
	case strings.HasPrefix(ct, "application/grpc-web"):
		g.grpcweb.ServeHTTP(w, r)

		return
	default:
		g.connect.ServeHTTP(w, r)
	}
}

// SetProxies updates the proxy routes on both sub-gateways at runtime.
// This lets the gRPC server share its proxy routes with the gateway after
// they are built (the gateway is created before proxy routes are available).
func (g *MultiProtocolGateway) SetProxies(r *proxyroutes.Registry) {
	g.grpcweb.SetProxies(r)
	g.connect.SetProxies(r)
}
