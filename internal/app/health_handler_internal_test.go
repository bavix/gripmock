package app

import (
	"context"
	stderrors "errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/types"
)

var errNilHealthResponse = stderrors.New("nil health response")

func newHealthTestEnv(stubs ...*stuber.Stub) *mockableHealthServer {
	realServer := health.NewServer()
	realServer.SetServingStatus(HealthServiceName, healthgrpc.HealthCheckResponse_SERVING)

	budgerigar := stuber.NewBudgerigar()
	budgerigar.PutMany(stubs...)

	return newMockableHealthServer(realServer, budgerigar, nil, nil)
}

func TestMockableHealthServerCheckUsesStubForGripmockService(t *testing.T) {
	t.Parallel()

	// Arrange
	handler := newHealthTestEnv(&stuber.Stub{
		Service: HealthServiceFullName,
		Method:  "Check",
		Input:   stuber.InputData{Equals: map[string]any{"service": ""}},
		Output:  stuber.Output{Data: map[string]any{"status": "NOT_SERVING"}},
	})

	// Act
	gripmockResp, gripmockErr := handler.Check(t.Context(), &healthgrpc.HealthCheckRequest{Service: HealthServiceName})
	globalResp, globalErr := handler.Check(t.Context(), &healthgrpc.HealthCheckRequest{Service: ""})

	// Assert
	require.NoError(t, gripmockErr)
	require.Equal(t, healthgrpc.HealthCheckResponse_NOT_SERVING, gripmockResp.GetStatus())
	require.NoError(t, globalErr)
	require.Equal(t, healthgrpc.HealthCheckResponse_NOT_SERVING, globalResp.GetStatus())
}

func TestMockableHealthServerCheckReturnsMockedStatus(t *testing.T) {
	t.Parallel()

	// Arrange
	handler := newHealthTestEnv(&stuber.Stub{
		Service: HealthServiceFullName,
		Method:  "Check",
		Input:   stuber.InputData{Equals: map[string]any{"service": "orders.v1.OrderService"}},
		Output:  stuber.Output{Data: map[string]any{"status": "NOT_SERVING"}},
	})

	// Act
	resp, err := handler.Check(t.Context(), &healthgrpc.HealthCheckRequest{Service: "orders.v1.OrderService"})

	// Assert
	require.NoError(t, err)
	require.Equal(t, healthgrpc.HealthCheckResponse_NOT_SERVING, resp.GetStatus())
}

func TestMockableHealthServerCheckFallbackToRealHealthServer(t *testing.T) {
	t.Parallel()

	// Arrange
	handler := newHealthTestEnv()

	// Act
	resp, err := handler.Check(t.Context(), &healthgrpc.HealthCheckRequest{Service: "inventory.v1.InventoryService"})

	// Assert
	require.Nil(t, resp)
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestMockableHealthServerCheckRespectsSessionMetadata(t *testing.T) {
	t.Parallel()

	// Arrange
	handler := newHealthTestEnv(&stuber.Stub{
		Service: HealthServiceFullName,
		Method:  "Check",
		Session: "s-42",
		Input:   stuber.InputData{Equals: map[string]any{"service": "billing.v1.BillingService"}},
		Output:  stuber.Output{Data: map[string]any{"status": "NOT_SERVING"}},
	})

	ctx := metadata.NewIncomingContext(t.Context(), metadata.New(map[string]string{"x-gripmock-session": "s-42"}))

	// Act
	resp, err := handler.Check(ctx, &healthgrpc.HealthCheckRequest{Service: "billing.v1.BillingService"})

	// Assert
	require.NoError(t, err)
	require.Equal(t, healthgrpc.HealthCheckResponse_NOT_SERVING, resp.GetStatus())
}

func TestMockableHealthServerCheckReturnsOutputError(t *testing.T) {
	t.Parallel()

	// Arrange
	realServer := health.NewServer()
	budgerigar := stuber.NewBudgerigar()
	c := codes.Unavailable

	budgerigar.PutMany(&stuber.Stub{
		Service: HealthServiceFullName,
		Method:  "Check",
		Input:   stuber.InputData{Equals: map[string]any{"service": "search.v1.SearchService"}},
		Output:  stuber.Output{Code: &c, Error: "dependency unavailable"},
	})

	handler := newMockableHealthServer(realServer, budgerigar, nil, nil)

	// Act
	resp, err := handler.Check(t.Context(), &healthgrpc.HealthCheckRequest{Service: "search.v1.SearchService"})

	// Assert
	require.Nil(t, resp)
	require.Error(t, err)
	require.Equal(t, codes.Unavailable, status.Code(err))
	require.Contains(t, err.Error(), "dependency unavailable")
}

func TestMockableHealthServerCheckReturnsOutputErrorWithDetails(t *testing.T) {
	t.Parallel()

	// Arrange
	realServer := health.NewServer()
	budgerigar := stuber.NewBudgerigar()
	c := codes.InvalidArgument

	budgerigar.PutMany(&stuber.Stub{
		Service: HealthServiceFullName,
		Method:  "Check",
		Input:   stuber.InputData{Equals: map[string]any{"service": "profile.v1.ProfileService"}},
		Output: stuber.Output{
			Code:  &c,
			Error: "invalid profile request",
			Details: []map[string]any{
				{
					"type":   "type.googleapis.com/google.rpc.ErrorInfo",
					"reason": "INVALID_PROFILE",
					"domain": "gripmock",
				},
			},
		},
	})

	handler := newMockableHealthServer(realServer, budgerigar, nil, nil)

	// Act
	resp, err := handler.Check(t.Context(), &healthgrpc.HealthCheckRequest{Service: "profile.v1.ProfileService"})

	// Assert
	require.Nil(t, resp)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	details := status.Convert(err).Details()
	require.Len(t, details, 1)

	info, ok := details[0].(*errdetails.ErrorInfo)
	require.True(t, ok)
	require.Equal(t, "INVALID_PROFILE", info.GetReason())
	require.Equal(t, "gripmock", info.GetDomain())
}

func TestMockableHealthServerCheckReturnsErrorOnInvalidOutputDetails(t *testing.T) {
	t.Parallel()

	// Arrange
	realServer := health.NewServer()
	budgerigar := stuber.NewBudgerigar()
	c := codes.InvalidArgument

	budgerigar.PutMany(&stuber.Stub{
		Service: HealthServiceFullName,
		Method:  "Check",
		Input:   stuber.InputData{Equals: map[string]any{"service": "profile.v1.ProfileService"}},
		Output: stuber.Output{
			Code:  &c,
			Error: "invalid profile request",
			Details: []map[string]any{
				{"reason": "MISSING_TYPE"},
			},
		},
	})

	handler := newMockableHealthServer(realServer, budgerigar, nil, nil)

	// Act
	resp, err := handler.Check(t.Context(), &healthgrpc.HealthCheckRequest{Service: "profile.v1.ProfileService"})

	// Assert
	require.Nil(t, resp)
	require.Error(t, err)
	require.Contains(t, err.Error(), "output.details[0]")
}

func TestMockableHealthServerWatchStreamsMockedResponses(t *testing.T) {
	t.Parallel()

	// Arrange
	realServer := health.NewServer()
	budgerigar := stuber.NewBudgerigar()

	budgerigar.PutMany(&stuber.Stub{
		Service: HealthServiceFullName,
		Method:  "Watch",
		Input:   stuber.InputData{Equals: map[string]any{"service": "payments.v1.PaymentsService"}},
		Output: stuber.Output{Stream: []any{
			map[string]any{"status": "NOT_SERVING"},
			map[string]any{"status": "SERVING"},
		}},
	})

	handler := newMockableHealthServer(realServer, budgerigar, nil, nil)
	stream := newHealthWatchTestStream(t.Context(), 2)

	errCh := make(chan error, 1)

	// Act
	go func() {
		errCh <- handler.Watch(&healthgrpc.HealthCheckRequest{Service: "payments.v1.PaymentsService"}, stream)
	}()

	require.Eventually(t, func() bool {
		return stream.Count() == 2
	}, time.Second, 10*time.Millisecond)

	var err error
	select {
	case err = <-errCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Watch handler to complete")
	}

	// Assert
	require.NoError(t, err)
	require.Equal(t, []healthgrpc.HealthCheckResponse_ServingStatus{
		healthgrpc.HealthCheckResponse_NOT_SERVING,
		healthgrpc.HealthCheckResponse_SERVING,
	}, stream.Statuses())
}

func TestMockableHealthServerWatchUsesStubForGripmockService(t *testing.T) {
	t.Parallel()

	// Arrange
	realServer := health.NewServer()
	realServer.SetServingStatus(HealthServiceName, healthgrpc.HealthCheckResponse_SERVING)

	handler := newMockableHealthServer(realServer, stuber.NewBudgerigar(), nil, nil)
	stream := newHealthWatchTestStream(t.Context(), 1)

	errCh := make(chan error, 1)

	// Act
	go func() {
		errCh <- handler.Watch(&healthgrpc.HealthCheckRequest{Service: HealthServiceName}, stream)
	}()

	require.Eventually(t, func() bool {
		return stream.Count() == 1
	}, time.Second, 10*time.Millisecond)

	var err error
	select {
	case err = <-errCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Watch handler to complete")
	}

	// Assert
	require.NoError(t, err)
	require.Equal(t, []healthgrpc.HealthCheckResponse_ServingStatus{
		healthgrpc.HealthCheckResponse_NOT_SERVING,
	}, stream.Statuses())
}

func TestMockableHealthServerWatchSupportsDelay(t *testing.T) {
	t.Parallel()

	// Arrange
	realServer := health.NewServer()
	budgerigar := stuber.NewBudgerigar()

	budgerigar.PutMany(&stuber.Stub{
		Service: HealthServiceFullName,
		Method:  "Watch",
		Input:   stuber.InputData{Equals: map[string]any{"service": "gateway.v1.GatewayService"}},
		Output: stuber.Output{
			Delay:  types.Duration(25 * time.Millisecond),
			Stream: []any{map[string]any{"status": "SERVING"}},
		},
	})

	handler := newMockableHealthServer(realServer, budgerigar, nil, nil)
	stream := newHealthWatchTestStream(t.Context(), 1)

	start := time.Now()

	// Act
	err := handler.Watch(&healthgrpc.HealthCheckRequest{Service: "gateway.v1.GatewayService"}, stream)

	// Assert
	require.NoError(t, err)
	require.GreaterOrEqual(t, time.Since(start), 25*time.Millisecond)
}

func TestMockableHealthServerWatchAppliesDelayOnlyBeforeFirstMessage(t *testing.T) {
	t.Parallel()

	const delayMs = 80

	// Arrange
	realServer := health.NewServer()
	budgerigar := stuber.NewBudgerigar()

	budgerigar.PutMany(&stuber.Stub{
		Service: HealthServiceFullName,
		Method:  "Watch",
		Input:   stuber.InputData{Equals: map[string]any{"service": "gateway.v1.SequenceService"}},
		Output: stuber.Output{
			Delay: types.Duration(delayMs * time.Millisecond),
			Stream: []any{
				map[string]any{"status": "NOT_SERVING"},
				map[string]any{"status": "SERVING"},
			},
		},
	})

	handler := newMockableHealthServer(realServer, budgerigar, nil, nil)
	stream := newHealthWatchTestStream(t.Context(), 2)

	start := time.Now()

	// Act
	err := handler.Watch(&healthgrpc.HealthCheckRequest{Service: "gateway.v1.SequenceService"}, stream)
	duration := time.Since(start)

	// Assert
	require.NoError(t, err)
	require.Equal(t, []healthgrpc.HealthCheckResponse_ServingStatus{
		healthgrpc.HealthCheckResponse_NOT_SERVING,
		healthgrpc.HealthCheckResponse_SERVING,
	}, stream.Statuses())

	// Delay applies before first message, not between messages.
	// Total duration ~= delay + overhead for sending all messages.
	require.GreaterOrEqual(t, duration, delayMs*time.Millisecond,
		"total duration should include initial delay")
}

type healthWatchTestStream struct {
	contextProvider func() context.Context
	cancel          context.CancelFunc
	cancelAfter     int

	mu     sync.Mutex
	status []healthgrpc.HealthCheckResponse_ServingStatus
}

func newHealthWatchTestStream(parent context.Context, cancelAfter int) *healthWatchTestStream {
	ctx, cancel := context.WithCancel(parent)

	return &healthWatchTestStream{
		contextProvider: func() context.Context { return ctx },
		cancel:          cancel,
		cancelAfter:     cancelAfter,
	}
}

func (s *healthWatchTestStream) Context() context.Context {
	return s.contextProvider()
}

func (s *healthWatchTestStream) SetHeader(metadata.MD) error {
	return nil
}

func (s *healthWatchTestStream) SendHeader(metadata.MD) error {
	return nil
}

func (s *healthWatchTestStream) SetTrailer(metadata.MD) {}

func (s *healthWatchTestStream) SendMsg(any) error {
	return nil
}

func (s *healthWatchTestStream) RecvMsg(any) error {
	<-s.Context().Done()

	return s.Context().Err()
}

func (s *healthWatchTestStream) Send(resp *healthgrpc.HealthCheckResponse) error {
	if resp == nil {
		return errNilHealthResponse
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.status = append(s.status, resp.GetStatus())
	if s.cancelAfter > 0 && len(s.status) >= s.cancelAfter {
		s.cancel()
	}

	return nil
}

func (s *healthWatchTestStream) Statuses() []healthgrpc.HealthCheckResponse_ServingStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]healthgrpc.HealthCheckResponse_ServingStatus, len(s.status))
	copy(out, s.status)

	return out
}

func (s *healthWatchTestStream) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.status)
}
