package app

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protodesc"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

type mockableHealthServer struct {
	healthgrpc.UnimplementedHealthServer

	real        *health.Server
	queryFinder stuber.QueryFinder
	resolver    protodesc.Resolver
}

func newMockableHealthServer(
	healthServer *health.Server,
	queryFinder stuber.QueryFinder,
	resolver protodesc.Resolver,
) *mockableHealthServer {
	return &mockableHealthServer{
		real:        healthServer,
		queryFinder: queryFinder,
		resolver:    resolver,
	}
}

func (s *mockableHealthServer) Check(
	ctx context.Context,
	req *healthgrpc.HealthCheckRequest,
) (*healthgrpc.HealthCheckResponse, error) {
	if s.shouldBypassMocks(req.GetService()) {
		return s.real.Check(ctx, req)
	}

	stub, ok := s.findStub(ctx, "Check", req.GetService())

	if !ok {
		return s.real.Check(ctx, req)
	}

	if err := delayResponse(ctx, stub.Output.Delay); err != nil {
		return nil, err
	}

	st, err := statusFromHealthOutput(stub.Output, s.resolver)
	if err != nil {
		return nil, err
	}

	if st != nil {
		return nil, st.Err()
	}

	healthStatus, err := healthStatusFromMap(stub.Output.Data)
	if err != nil {
		return nil, err
	}

	return &healthgrpc.HealthCheckResponse{Status: healthStatus}, nil
}

func (s *mockableHealthServer) Watch(req *healthgrpc.HealthCheckRequest, stream healthgrpc.Health_WatchServer) error {
	if s.shouldBypassMocks(req.GetService()) {
		return s.real.Watch(req, stream)
	}

	stub, ok := s.findStub(stream.Context(), "Watch", req.GetService())

	if !ok {
		return s.real.Watch(req, stream)
	}

	st, err := statusFromHealthOutput(stub.Output, s.resolver)
	if err != nil {
		return err
	}

	if st != nil {
		return st.Err()
	}

	responses, err := healthResponsesFromOutput(stub.Output)
	if err != nil {
		return err
	}

	if len(responses) == 0 {
		return status.Error(codes.Internal, "health watch stub output is empty")
	}

	if err := delayResponse(stream.Context(), stub.Output.Delay); err != nil {
		return err
	}

	for _, response := range responses {
		if err := stream.Send(response); err != nil {
			return err
		}
	}

	// For mocked Watch streams, return after all configured responses are sent.
	// This prevents hanging when the client doesn't cancel the context promptly.
	return nil
}

func (s *mockableHealthServer) shouldBypassMocks(service string) bool {
	return service == "" || service == HealthServiceName
}
