package app

import (
	"context"

	"google.golang.org/grpc/metadata"

	"github.com/bavix/gripmock/v3/internal/infra/proxyroutes"
)

const (
	proxyMessagesInitCap = 8
	proxyErrChanCap      = 2
)

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
