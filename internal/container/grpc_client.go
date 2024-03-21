package container

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func (c *Container) grpcClient(ctx context.Context, target string) (*grpc.ClientConn, error) {
	if c.grpcClientConn != nil {
		return c.grpcClientConn, nil
	}

	conn, err := grpc.DialContext(ctx, target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	c.shutdown.Add(func(_ context.Context) error {
		return conn.Close()
	})

	c.grpcClientConn = conn

	return c.grpcClientConn, nil
}
