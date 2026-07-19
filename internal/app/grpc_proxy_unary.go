package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/infra/proxycapture"
	"github.com/bavix/gripmock/v3/internal/infra/proxyroutes"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

const healthCheckFullMethod = "/grpc.health.v1.Health/Check"

// grpcWebTrailerAdder allows the unary proxy to attach upstream gRPC trailers
// to the gRPC-Web trailers frame so they reach @trailer assertions.
type grpcWebTrailerAdder interface {
	setTrailerExtra(lines ...string)
}

func (m *grpcMocker) proxyUnary(
	ctx context.Context,
	stream grpc.ServerStream,
	req *dynamicpb.Message,
	route *proxyroutes.Route,
	capture bool,
) (*dynamicpb.Message, error) {
	proxyCtx, cancel := route.WithTimeout(proxyroutes.ForwardIncomingMetadata(ctx))
	defer cancel()

	var (
		header  metadata.MD
		trailer metadata.MD
	)

	startTime := time.Now()

	var (
		resp *dynamicpb.Message
		err  error
	)

	if m.fullMethod == healthCheckFullMethod {
		resp, err = m.proxyHealthCheck(proxyCtx, route, req, &header, &trailer)
	} else {
		resp = dynamicpb.NewMessage(m.outputDesc)
		err = route.Conn.Invoke(proxyCtx, m.fullMethod, req, resp, grpc.Header(&header), grpc.Trailer(&trailer))
	}

	elapsed := time.Since(startTime)

	setStreamMetadata(ctx, stream, header, trailer)

	if ta, ok := stream.(grpcWebTrailerAdder); ok {
		for _, md := range []metadata.MD{header, trailer} {
			for k, v := range md {
				switch strings.ToLower(k) {
				case "content-type", "content-encoding", "content-length",
					"grpc-status", "grpc-message", "grpc-status-details-bin",
					":authority", "user-agent", "accept-encoding",
					"grpc-accept-encoding":
					continue
				}

				ta.setTrailerExtra(k + ": " + strings.Join(v, ","))
			}
		}
	}

	if capture {
		m.recordUnaryStub(ctx, req, resp, route, header, trailer, err, elapsed)
	}

	return resp, err
}

func (m *grpcMocker) proxyHealthCheck(
	ctx context.Context,
	route *proxyroutes.Route,
	req *dynamicpb.Message,
	header, trailer *metadata.MD,
) (*dynamicpb.Message, error) {
	raw, marshalErr := proto.Marshal(req)
	if marshalErr != nil {
		return nil, marshalErr
	}

	var healthReq healthgrpc.HealthCheckRequest
	if unmarshalErr := proto.Unmarshal(raw, &healthReq); unmarshalErr != nil {
		return nil, unmarshalErr
	}

	healthResp, err := healthgrpc.NewHealthClient(route.Conn).Check(ctx, &healthReq,
		grpc.Header(header), grpc.Trailer(trailer))
	if err != nil {
		return nil, err
	}

	resp := dynamicpb.NewMessage(m.outputDesc)

	respData, marshalErr := proto.Marshal(healthResp)
	if marshalErr != nil {
		return nil, fmt.Errorf("marshal health response: %w", marshalErr)
	}

	if unmarshalErr := proto.Unmarshal(respData, resp); unmarshalErr != nil {
		return nil, fmt.Errorf("unmarshal health response: %w", unmarshalErr)
	}

	return resp, nil
}

func (m *grpcMocker) recordUnaryStub(
	ctx context.Context,
	req *dynamicpb.Message,
	resp *dynamicpb.Message,
	route *proxyroutes.Route,
	header metadata.MD,
	trailer metadata.MD,
	callErr error,
	elapsed time.Duration,
) {
	captureCtx := m.newCaptureRequestContext(ctx)
	requestData := m.convertToMap(req)
	responseHeaders := responseHeadersFromMetadata(header, trailer)

	var responseData any
	if resp != nil {
		responseData = messageToAny(resp)
	}

	m.recordCapturedStub(
		func() *stuber.Stub {
			return proxycapture.BuildUnaryStub(
				m.fullServiceName, m.methodName, captureCtx.sessionID,
				requestData, captureCtx.headers, responseData, responseHeaders, callErr,
			)
		},
		route.Source.RecordDelay, elapsed,
	)
}

func (m *grpcMocker) proxyStream(stream grpc.ServerStream, route *proxyroutes.Route, capture bool) error {
	switch {
	case m.serverStream && !m.clientStream:
		return m.proxyServerStream(stream, route, capture)
	case !m.serverStream && m.clientStream:
		return m.proxyClientStream(stream, route, capture)
	case m.serverStream && m.clientStream:
		return m.proxyBidiStream(stream, route, capture)
	default:
		return status.Error(codes.Unimplemented, "unknown stream type")
	}
}
