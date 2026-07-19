package app

import (
	"context"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/bavix/gripmock/v3/internal/infra/proxyroutes"
)

func ssmFilterMD(md metadata.MD) metadata.MD {
	if len(md) == 0 {
		return nil
	}

	excluded := map[string]struct{}{
		"content-type":            {},
		"content-encoding":        {},
		"content-length":          {},
		"grpc-status":             {},
		"grpc-message":            {},
		"grpc-status-details-bin": {},
		":authority":              {},
		"user-agent":              {},
		"accept-encoding":         {},
		"grpc-accept-encoding":    {},
	}

	filtered := make(metadata.MD, len(md))
	for k, v := range md {
		if _, exclude := excluded[strings.ToLower(k)]; exclude {
			continue
		}

		filtered[k] = v
	}

	return filtered
}

func setStreamMetadata(ctx context.Context, stream grpc.ServerStream, header, trailer metadata.MD) {
	if stream == nil {
		setContextMetadata(ctx, header, trailer)

		return
	}

	// Forward filtered upstream metadata as HTTP response headers
	// for ConnectRPC (httpStreamAdapter).  Skip for gRPC-Web
	// (grpcwebAdapter) — its framed format does not use HTTP headers.
	if _, ok := stream.(*grpcwebAdapter); ok {
		return
	}

	if h := ssmFilterMD(header); len(h) > 0 {
		if err := stream.SetHeader(h); err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Msg("failed to set stream header metadata")
		}
	}

	if t := ssmFilterMD(trailer); len(t) > 0 {
		stream.SetTrailer(t)
	}
}

func setContextMetadata(ctx context.Context, header, trailer metadata.MD) {
	if len(header) > 0 {
		if err := grpc.SetHeader(ctx, header); err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Msg("failed to set header metadata")
		}
	}

	if len(trailer) > 0 {
		if err := grpc.SetTrailer(ctx, trailer); err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Msg("failed to set trailer metadata")
		}
	}
}

const (
	proxyMessagesInitCap     = 8
	proxyErrChanCap          = 2
	proxyBidiTimeoutFallback = 5 * time.Second
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
