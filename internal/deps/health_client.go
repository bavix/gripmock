package deps

import (
	"net"

	healthv1 "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/bavix/gripmock/v3/internal/infra/waiter"
)

func (b *Builder) PingService() (*waiter.Service, error) {
	addr := normalizePingAddress(b.config.GRPCAddr)

	grpcConn, err := b.grpcClientConn(b.grpcTLSEnabled(), addr)
	if err != nil {
		return nil, err
	}

	return waiter.NewService(healthv1.NewHealthClient(grpcConn)), nil
}

func normalizePingAddress(address string) string {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return address
	}

	switch host {
	case "0.0.0.0":
		return net.JoinHostPort("127.0.0.1", port)
	case "::", "[::]":
		return net.JoinHostPort("::1", port)
	default:
		return address
	}
}
