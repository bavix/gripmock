package deps

import (
	"context"

	"github.com/gripmock/environment"
	"github.com/gripmock/shutdown"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func grpcClient(
	config environment.Config,
	shutdown *shutdown.Shutdown,
) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.NewClient(config.GRPCAddr, opts...)
	if err != nil {
		return nil, err
	}

	shutdown.Add(func(_ context.Context) error {
		return conn.Close()
	})

	return conn, nil
}
