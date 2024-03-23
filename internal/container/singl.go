package container

import (
	"google.golang.org/grpc"

	"github.com/bavix/gripmock/internal/app"
	"github.com/bavix/gripmock/internal/config"
	"github.com/bavix/gripmock/internal/pkg/grpcreflector"
	"github.com/bavix/gripmock/internal/pkg/shutdown"
)

type Container struct {
	shutdown *shutdown.Shutdown
	conf     *config.Config

	grpcClientConn *grpc.ClientConn
	gRef           *grpcreflector.GReflector

	rest *app.RestServer
}

func New(conf *config.Config) *Container {
	return &Container{
		shutdown: shutdown.New(),
		conf:     conf,
	}
}
