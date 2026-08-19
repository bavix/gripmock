package waiter

import (
	"context"
	"fmt"
	"strconv"
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

	// time.NewTicker panics on a non-positive interval; a caller-supplied 0/negative
	// ping interval falls back to a tight-but-safe poll instead of crashing.
	if interval <= 0 {
		interval = time.Millisecond
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	started := time.Now()
	lastCause := "no ping response yet"

	for time.Now().Before(deadline) {
		code, err := s.Ping(ctx, service)

		switch {
		case err != nil:
			lastCause = err.Error()
		case code == Serving:
			return nil
		default:
			lastCause = "status " + strconv.Itoa(int(code))
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}

	return fmt.Errorf("%w: %s within %s (waited %s, last check: %s)",
		ErrServerNotReady, service, timeout, time.Since(started).Round(time.Millisecond), lastCause)
}
