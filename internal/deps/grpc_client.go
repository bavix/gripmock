package deps

import (
	"context"

	"github.com/cockroachdb/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func (b *Builder) grpcClientConn(_ bool, dsn string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(
		dsn,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create gRPC client connection")
	}

	b.ender.Add(func(_ context.Context) error {
		return conn.Close()
	})

	return conn, nil
}
