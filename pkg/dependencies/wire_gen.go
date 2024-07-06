// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package dependencies

import (
	"context"
	"github.com/bavix/gripmock/pkg/grpcreflector"
	"github.com/gripmock/environment"
	"github.com/gripmock/shutdown"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
)

// Injectors from builder.go:

func New(ctx context.Context, appName string) (*Builder, error) {
	config, err := environment.New()
	if err != nil {
		return nil, err
	}
	logger, err := NewZerolog(config)
	if err != nil {
		return nil, err
	}
	dependenciesLogger := newLog(logger)
	shutdownShutdown := shutdown.New(dependenciesLogger)
	tracerProvider, err := tracer(ctx, config, shutdownShutdown, appName)
	if err != nil {
		return nil, err
	}
	clientConn, err := grpcClient(config, shutdownShutdown)
	if err != nil {
		return nil, err
	}
	gReflector := grpcreflector.New(clientConn)
	builder := &Builder{
		ender:      shutdownShutdown,
		config:     config,
		tracer:     tracerProvider,
		logger:     logger,
		reflector:  gReflector,
		grpcClient: clientConn,
	}
	return builder, nil
}

// builder.go:

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
