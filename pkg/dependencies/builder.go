//go:build wireinject

package dependencies

import (
	"context"

	"github.com/google/wire"
	"github.com/gripmock/environment"
	"github.com/gripmock/shutdown"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"

	"github.com/bavix/gripmock/pkg/grpcreflector"
)

type Builder struct {
	ender      *shutdown.Shutdown
	config     environment.Config
	tracer     *trace.TracerProvider
	logger     *zerolog.Logger
	reflector  *grpcreflector.GReflector
	grpcClient *grpc.ClientConn
}

func (b *Builder) Config() environment.Config {
	return b.config
}

func (b *Builder) Tracer() *trace.TracerProvider {
	return b.tracer
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

func New(ctx context.Context, appName string) (*Builder, error) {
	panic(wire.Build(
		environment.New,
		NewZerolog,
		newLog,
		wire.Bind(new(shutdown.Logger), new(*Logger)),
		shutdown.New,
		tracer,
		grpcClient,
		grpcreflector.New,
		wire.Struct(new(Builder), "*"),
	))
	return &Builder{}, nil
}
