package deps

import (
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/bavix/gripmock/v3/internal/infra/waiter"
)

func (b *Builder) PingService() (*waiter.Service, error) {
	grpcConn, err := b.grpcClientConn(false, b.config.GRPCAddr)
	if err != nil {
		return nil, err
	}

	return waiter.NewService(healthv1.NewHealthClient(grpcConn)), nil
}
