package deps

import (
	"context"
	"net"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog"

	"github.com/bavix/gripmock/v3/internal/app"
	"github.com/bavix/gripmock/v3/internal/domain/proto"
)

func (b *Builder) GRPCServe(ctx context.Context, param *proto.Arguments) error {
	network := b.config.GRPCNetwork
	addr := b.config.GRPCAddr

	listener, err := listen(ctx, network, addr)
	if err != nil {
		return err
	}

	grpcServer := b.buildGRPC(ctx, network, addr, param)

	return serveGRPC(ctx, grpcServer, listener, b)
}

func listen(ctx context.Context, network, addr string) (net.Listener, error) {
	listener, err := (&net.ListenConfig{}).Listen(ctx, network, addr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to listen")
	}

	return listener, nil
}

func (b *Builder) buildGRPC(ctx context.Context, network, addr string, param *proto.Arguments) *app.GRPCServer {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("addr", addr).
		Str("network", network).
		Msg("Serving gRPC")

	return app.NewGRPCServer(
		network,
		addr,
		param,
		b.Budgerigar(),
		b.ServiceManager(),
		b.Extender(ctx),
		b.ErrorFormatter(),
		b.TemplateRegistry(ctx),
	)
}

func serveGRPC(ctx context.Context, grpcServer *app.GRPCServer, listener net.Listener, b *Builder) error {
	server, err := grpcServer.Build(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to build gRPC server")
	}

	b.ender.Add(func(_ context.Context) error {
		server.GracefulStop()

		return nil
	})

	ch := make(chan error)
	logger := zerolog.Ctx(ctx)

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
