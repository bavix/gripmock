package app

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/infra/proxycapture"
	"github.com/bavix/gripmock/v3/internal/infra/proxyroutes"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func (m *grpcMocker) proxyUnary(
	ctx context.Context,
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
	resp := dynamicpb.NewMessage(m.outputDesc)
	err := route.Conn.Invoke(proxyCtx, m.fullMethod, req, resp, grpc.Header(&header), grpc.Trailer(&trailer))
	elapsed := time.Since(startTime)

	if len(header) > 0 {
		_ = grpc.SetHeader(ctx, header)
	}

	if len(trailer) > 0 {
		_ = grpc.SetTrailer(ctx, trailer)
	}

	if capture {
		m.recordUnaryStub(ctx, req, resp, route, header, trailer, err, elapsed)
	}

	return resp, err
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
	requestData := convertToMap(req)
	responseHeaders := responseHeadersFromMetadata(header, trailer)

	var responseData map[string]any
	if resp != nil {
		responseData = messageToMap(resp)
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
