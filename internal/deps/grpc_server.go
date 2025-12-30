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
	listener, err := (&net.ListenConfig{}).Listen(ctx, b.config.GRPCNetwork, b.config.GRPCAddr)
	if err != nil {
		return errors.Wrap(err, "failed to listen")
	}

	logger := zerolog.Ctx(ctx)

	logger.Info().
		Str("addr", listener.Addr().String()).
		Str("network", listener.Addr().Network()).
		Msg("Serving gRPC")

	grpcServer := app.NewGRPCServer(
		b.config.GRPCNetwork,
		b.config.GRPCAddr,
		param,
		b.Budgerigar(),
		b.Extender(ctx),
	)

	server, err := grpcServer.Build(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to build gRPC server")
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
