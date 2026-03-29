package app

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/bavix/gripmock/v3/internal/infra/proxyroutes"
)

const (
	proxyMessagesInitCap = 8
	proxyErrChanCap      = 2
)

type captureRequestContext struct {
	headers   map[string]any
	sessionID string
}

func (m *grpcMocker) proxyRoute() *proxyroutes.Route {
	if m.proxies == nil {
		return nil
	}

	return m.proxies.RouteByMethod(m.fullMethod)
}

func (m *grpcMocker) sessionFromContext(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		return sessionFromMetadata(md)
	}

	return ""
}

func (m *grpcMocker) newCaptureRequestContext(ctx context.Context) captureRequestContext {
	md, _ := metadata.FromIncomingContext(ctx)

	return captureRequestContext{
		headers:   requestHeadersFromMetadata(md),
		sessionID: m.sessionFromContext(ctx),
	}
}

func (m *grpcMocker) hasCaptureRequestHeaders(ctx context.Context) bool {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return false
	}

	return len(requestHeadersFromMetadata(md)) > 0
}

func responseHeadersFromClientStream(clientStream grpc.ClientStream) map[string]string {
	if clientStream == nil {
		return nil
	}

	return responseHeadersFromMetadata(nil, clientStream.Trailer())
}
