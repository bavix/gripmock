package waiter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
)

var errPingRefused = errors.New("connection refused")

type stubHealthClient struct {
	status healthv1.HealthCheckResponse_ServingStatus
	err    error
	calls  int
}

func (c *stubHealthClient) Check(
	_ context.Context,
	_ *healthv1.HealthCheckRequest,
	_ ...grpc.CallOption,
) (*healthv1.HealthCheckResponse, error) {
	c.calls++

	if c.err != nil {
		return nil, c.err
	}

	return &healthv1.HealthCheckResponse{Status: c.status}, nil
}

func (c *stubHealthClient) List(
	_ context.Context,
	_ *healthv1.HealthListRequest,
	_ ...grpc.CallOption,
) (*healthv1.HealthListResponse, error) {
	return &healthv1.HealthListResponse{}, nil
}

func (c *stubHealthClient) Watch(
	_ context.Context,
	_ *healthv1.HealthCheckRequest,
	_ ...grpc.CallOption,
) (grpc.ServerStreamingClient[healthv1.HealthCheckResponse], error) {
	return nil, errPingRefused
}

func TestWaitForReadyReturnsOnFirstServingCheck(t *testing.T) {
	t.Parallel()

	client := &stubHealthClient{status: healthv1.HealthCheckResponse_SERVING}

	require.NoError(t, NewService(client).WaitForReady(t.Context(), time.Second, time.Hour, "svc"))
	require.Equal(t, 1, client.calls)
}

func TestWaitForReadyReportsWhyItGaveUp(t *testing.T) {
	t.Parallel()

	client := &stubHealthClient{err: errPingRefused}

	err := NewService(client).WaitForReady(t.Context(), 150*time.Millisecond, 10*time.Millisecond, "svc")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrServerNotReady)
	require.Contains(t, err.Error(), "svc")
	require.Contains(t, err.Error(), "connection refused")
	require.Contains(t, err.Error(), "waited")
}

func TestWaitForReadyReportsNotServing(t *testing.T) {
	t.Parallel()

	client := &stubHealthClient{status: healthv1.HealthCheckResponse_NOT_SERVING}

	err := NewService(client).WaitForReady(t.Context(), 100*time.Millisecond, 10*time.Millisecond, "svc")
	require.ErrorIs(t, err, ErrServerNotReady)
	require.Contains(t, err.Error(), "status")
}
