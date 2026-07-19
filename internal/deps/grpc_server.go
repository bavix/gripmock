package deps

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog"

	"github.com/bavix/gripmock/v3/internal/app"
	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/domain/proto"
)

//nolint:funlen,cyclop
func (b *Builder) GRPCServe(ctx context.Context, param *proto.Arguments) error {
	StartSessionGC(ctx, b.config, b.Budgerigar(), b.HistoryStore(), b.ender)

	grpcTLS := b.grpcTLSConfig()
	grpcTLS.ClientAuth = b.config.GRPCTLS.ClientAuth

	var (
		tlsCfg *tls.Config
		err    error
	)

	if grpcTLS.IsEnabled() {
		tlsCfg, err = grpcTLS.BuildTLSConfig()
		if err != nil {
			return errors.Wrap(err, "failed to build TLS config")
		}
	}

	listener, err := (&net.ListenConfig{}).Listen(ctx, b.config.GRPCNetwork, b.config.GRPC.Addr)
	if err != nil {
		return errors.Wrap(err, "failed to listen")
	}

	logger := zerolog.Ctx(ctx)

	logger.Info().
		Str("addr", listener.Addr().String()).
		Str("network", listener.Addr().Network()).
		Bool("tls", grpcTLS.IsEnabled()).
		Msg("Serving gRPC")

	var recorder history.Recorder
	if store := b.HistoryStore(); store != nil {
		recorder = store
	}

	grpcServer := app.NewGRPCServer(
		b.config.GRPCNetwork,
		b.config.GRPC.Addr,
		param,
		b.Budgerigar(),
		b.Extender(ctx),
		recorder,
		b.DescriptorRegistry(),
		tlsCfg,
		b.RemoteClient(),
		b.config.OTel.Enabled,
		b.config.MaxNestingDepth,
		b.StubValidator(),
		b.ErrorFormatter(),
	)

	server, err := grpcServer.Build(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to build gRPC server")
	}

	// Share proxy routes with the gateway.
	// The gateway reads the atomic pointer directly, so it picks up
	// the routes as soon as they are stored here.
	if p := grpcServer.Proxies(); p != nil {
		b.SetProxyRoutes(p)
	}

	b.ender.Add(func(_ context.Context) error {
		server.GracefulStop()

		return nil
	})

	ch := make(chan error)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Fatal().
					Interface("panic", r).
					Msg("Fatal panic in gRPC server goroutine - terminating server")
			}
		}()
		defer close(ch)

		ch <- server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		if !errors.Is(ctx.Err(), context.Canceled) {
			return errors.Wrap(ctx.Err(), "failed to serve")
		}
	case err := <-ch:
		if !errors.Is(err, context.Canceled) {
			return errors.Wrap(err, "failed to serve")
		}
	}

	return nil
}
