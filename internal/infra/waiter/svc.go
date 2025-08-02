package waiter

import (
	"context"
	"time"

	"google.golang.org/grpc"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/bavix/gripmock/v3/internal/domain/waiter"
)

type Service struct {
	client healthv1.HealthClient
}

func NewService(client healthv1.HealthClient) *Service {
	return &Service{client: client}
}

func (s *Service) PingWithTimeout(
	ctx context.Context,
	timeout time.Duration,
	service string,
) (waiter.ServingStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return s.Ping(ctx, service)
}

func (s *Service) Ping(ctx context.Context, service string) (waiter.ServingStatus, error) {
	check, err := s.client.Check(
		ctx,
		&healthv1.HealthCheckRequest{Service: service},
		grpc.WaitForReady(true),
	)
	if err != nil {
		return waiter.Unknown, err //nolint:wrapcheck
	}

	switch check.GetStatus() {
	case healthv1.HealthCheckResponse_SERVING:
		return waiter.Serving, nil
	case healthv1.HealthCheckResponse_NOT_SERVING:
		return waiter.NotServing, nil
	case healthv1.HealthCheckResponse_SERVICE_UNKNOWN:
		return waiter.ServiceUnknown, nil
	case healthv1.HealthCheckResponse_UNKNOWN:
		return waiter.Unknown, nil
	default:
		return waiter.Unknown, nil
	}
}
