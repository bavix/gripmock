package waiter

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"google.golang.org/grpc"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
)

var ErrServerNotReady = errors.New("server did not become ready")

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
) (ServingStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return s.Ping(ctx, service)
}

func (s *Service) Ping(ctx context.Context, service string) (ServingStatus, error) {
	check, err := s.client.Check(
		ctx,
		&healthv1.HealthCheckRequest{Service: service},
		grpc.WaitForReady(true),
	)
	if err != nil {
		return Unknown, err //nolint:wrapcheck
	}

	switch check.GetStatus() {
	case healthv1.HealthCheckResponse_SERVING:
		return Serving, nil
	case healthv1.HealthCheckResponse_NOT_SERVING:
		return NotServing, nil
	case healthv1.HealthCheckResponse_SERVICE_UNKNOWN:
		return ServiceUnknown, nil
	case healthv1.HealthCheckResponse_UNKNOWN:
		return Unknown, nil
	default:
		return Unknown, nil
	}
}

func (s *Service) WaitForReady(ctx context.Context, timeout, interval time.Duration, service string) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		code, err := s.Ping(ctx, service)
		if err == nil && code == Serving {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}

	return ErrServerNotReady
}
