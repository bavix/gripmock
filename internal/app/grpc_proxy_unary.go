package app

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/infra/proxyroutes"
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

	resp := dynamicpb.NewMessage(m.outputDesc)
	err := route.Conn.Invoke(proxyCtx, m.fullMethod, req, resp, grpc.Header(&header), grpc.Trailer(&trailer))

	if len(header) > 0 {
		_ = grpc.SetHeader(ctx, header)
	}

	if len(trailer) > 0 {
		_ = grpc.SetTrailer(ctx, trailer)
	}

	requestData := convertToMap(req)
	sessionID := m.sessionFromContext(ctx)

	if err != nil {
		if capture {
			m.recordCapturedUnaryStub(requestData, nil, err, sessionID)
		}

		return nil, err
	}

	responseData := convertToMap(resp)
	if capture {
		m.recordCapturedUnaryStub(requestData, responseData, nil, sessionID)
	}

	return resp, nil
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
