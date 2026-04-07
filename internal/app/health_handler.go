package app

import (
	"context"
	"io"
	"time"

	"github.com/cockroachdb/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protodesc"

	"github.com/bavix/gripmock/v3/internal/infra/proxycapture"
	"github.com/bavix/gripmock/v3/internal/infra/proxyroutes"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/types"
)

const (
	healthMethodCheck = "Check"
	healthMethodWatch = "Watch"
)

type mockableHealthServer struct {
	healthgrpc.UnimplementedHealthServer

	real     *health.Server
	storage  *stuber.Budgerigar
	resolver protodesc.Resolver
	proxies  *proxyroutes.Registry
}

func newMockableHealthServer(
	healthServer *health.Server,
	storage *stuber.Budgerigar,
	resolver protodesc.Resolver,
	proxies *proxyroutes.Registry,
) *mockableHealthServer {
	return &mockableHealthServer{
		real:     healthServer,
		storage:  storage,
		resolver: resolver,
		proxies:  proxies,
	}
}

func (s *mockableHealthServer) Check(
	ctx context.Context,
	req *healthgrpc.HealthCheckRequest,
) (*healthgrpc.HealthCheckResponse, error) {
	stub, ok := s.findStub(ctx, healthMethodCheck, req.GetService())

	if !ok {
		if route := s.proxyRoute(); route != nil && route.Conn != nil {
			return s.proxyCheck(ctx, req, route)
		}

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
	if stub, ok := s.findStub(stream.Context(), healthMethodWatch, req.GetService()); ok {
		return s.watchFromStub(stream.Context(), stream, stub)
	}

	return s.watchFromProxyOrFallback(req, stream)
}

func (s *mockableHealthServer) watchFromProxyOrFallback(
	req *healthgrpc.HealthCheckRequest,
	stream healthgrpc.Health_WatchServer,
) error {
	if route := s.proxyRoute(); route != nil && route.Conn != nil {
		return s.proxyWatch(req, stream, route)
	}

	return s.watchFallback(req, stream)
}

func (s *mockableHealthServer) watchFromStub(
	ctx context.Context,
	stream healthgrpc.Health_WatchServer,
	stub *stuber.Stub,
) error {
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

	if err := delayResponse(ctx, stub.Output.Delay); err != nil {
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

func (s *mockableHealthServer) watchFallback(req *healthgrpc.HealthCheckRequest, stream healthgrpc.Health_WatchServer) error {
	// Delegate to the real health server's Watch, but with a bounded context
	// timeout so the stream doesn't hang indefinitely when no stubs are
	// configured. This allows capture mode to work correctly while preventing
	// the CI hang seen when grpctestify waits for the stream to complete.
	ctx, cancel := context.WithTimeout(stream.Context(), watchFallbackTimeout)
	defer cancel()

	return s.real.Watch(req, &watchStreamWithContext{Health_WatchServer: stream, ctx: ctx})
}

const watchFallbackTimeout = 5 * time.Second

// watchStreamWithContext wraps a Health_WatchServer to enforce a bounded context.
type watchStreamWithContext struct {
	healthgrpc.Health_WatchServer

	ctx context.Context //nolint:containedctx
}

func (w *watchStreamWithContext) Context() context.Context {
	return w.ctx
}

func (s *mockableHealthServer) proxyRoute() *proxyroutes.Route {
	if s.proxies == nil || len(s.proxies.Routes()) != 1 {
		return nil
	}

	return s.proxies.Routes()[0]
}

func (s *mockableHealthServer) proxyCheck(
	ctx context.Context,
	req *healthgrpc.HealthCheckRequest,
	route *proxyroutes.Route,
) (*healthgrpc.HealthCheckResponse, error) {
	proxyCtx, cancel := route.WithTimeout(proxyroutes.ForwardIncomingMetadata(ctx))
	defer cancel()

	var header, trailer metadata.MD

	startTime := time.Now()
	resp, err := healthgrpc.NewHealthClient(route.Conn).Check(proxyCtx, req, grpc.Header(&header), grpc.Trailer(&trailer))
	elapsed := time.Since(startTime)

	if len(header) > 0 {
		_ = grpc.SetHeader(ctx, header)
	}

	if len(trailer) > 0 {
		_ = grpc.SetTrailer(ctx, trailer)
	}

	respHeaders := responseHeadersFromMetadata(header, trailer)
	s.captureProxyHealthStub(
		ctx, req, healthMethodCheck, proxycapture.MessageToMap(resp),
		nil, err, respHeaders, route, elapsed,
	)

	return resp, err
}

func (s *mockableHealthServer) proxyWatch(
	req *healthgrpc.HealthCheckRequest,
	stream healthgrpc.Health_WatchServer,
	route *proxyroutes.Route,
) error {
	proxyCtx, cancel := route.WithTimeout(proxyroutes.ForwardIncomingMetadata(stream.Context()))
	defer cancel()

	startTime := time.Now()

	clientStream, err := healthgrpc.NewHealthClient(route.Conn).Watch(proxyCtx, req)
	if err != nil {
		s.captureProxyHealthStub(stream.Context(), req, healthMethodWatch, nil, nil, err, nil, route, time.Since(startTime))

		return err
	}

	if header, headerErr := clientStream.Header(); headerErr == nil && len(header) > 0 {
		if setErr := stream.SetHeader(header); setErr != nil {
			return setErr
		}
	}

	responses := make([]map[string]any, 0, proxyMessagesInitCap)

	for {
		resp, recvErr := clientStream.Recv()
		if recvErr != nil {
			if errors.Is(recvErr, io.EOF) {
				recvErr = nil
			}

			if trailer := clientStream.Trailer(); len(trailer) > 0 {
				stream.SetTrailer(trailer)
			}

			respHeaders := responseHeadersFromMetadata(nil, clientStream.Trailer())
			s.captureProxyHealthStub(
				stream.Context(), req, healthMethodWatch,
				nil, responses, recvErr, respHeaders, route, time.Since(startTime),
			)

			return recvErr
		}

		responses = append(responses, proxycapture.MessageToMap(resp))

		if sendErr := stream.Send(resp); sendErr != nil {
			return sendErr
		}
	}
}

func (s *mockableHealthServer) captureProxyHealthStub(
	ctx context.Context,
	req *healthgrpc.HealthCheckRequest,
	method string,
	response map[string]any,
	responses []map[string]any,
	callErr error,
	responseHeaders map[string]string,
	route *proxyroutes.Route,
	elapsed time.Duration,
) {
	if s.storage == nil || route == nil || route.Mode != proxyroutes.ModeCapture {
		return
	}

	md, _ := metadata.FromIncomingContext(ctx)

	stub := s.buildHealthStub(method, md, req, response, responses, responseHeaders, callErr)
	if stub == nil {
		return
	}

	if route.Source != nil && route.Source.RecordDelay && elapsed > 0 {
		stub.Output.Delay = types.Duration(elapsed)
	}

	s.storage.PutMany(stub)
}

func (s *mockableHealthServer) buildHealthStub(
	method string,
	md metadata.MD,
	req *healthgrpc.HealthCheckRequest,
	response map[string]any,
	responses []map[string]any,
	responseHeaders map[string]string,
	callErr error,
) *stuber.Stub {
	switch method {
	case healthMethodCheck:
		return proxycapture.BuildUnaryStub(
			HealthServiceFullName, method, sessionFromMetadata(md),
			map[string]any{"service": req.GetService()}, requestHeadersFromMetadata(md),
			response, responseHeaders, callErr,
		)
	case healthMethodWatch:
		return proxycapture.BuildServerStreamStub(
			HealthServiceFullName, method, sessionFromMetadata(md),
			map[string]any{"service": req.GetService()}, requestHeadersFromMetadata(md),
			responses, responseHeaders, callErr,
		)
	}

	return nil
}
