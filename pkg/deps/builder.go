//go:build wireinject

package deps

import (
	"context"

	"github.com/google/wire"
	"github.com/gripmock/environment"
	"github.com/gripmock/shutdown"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"

	"github.com/bavix/gripmock/pkg/grpcreflector"
)

type Builder struct {
	ender      *shutdown.Shutdown
	config     environment.Config
	logger     *zerolog.Logger
	reflector  *grpcreflector.GReflector
	grpcClient *grpc.ClientConn
}

func (b *Builder) Config() environment.Config {
	return b.config
}

func (b *Builder) Logger() *zerolog.Logger {
	return b.logger
}

func (b *Builder) Reflector() *grpcreflector.GReflector {
	return b.reflector
}

func (b *Builder) GRPCClient() *grpc.ClientConn {
	return b.grpcClient
}

func New(ctx context.Context) (*Builder, error) {
	panic(wire.Build(
		environment.New,
		NewZerolog,
		newLog,
		wire.Bind(new(shutdown.Logger), new(*Logger)),
		shutdown.New,
		grpcClient,
		grpcreflector.New,
		wire.Struct(new(Builder), "*"),
	))
	return &Builder{}, nil
}
