package deps

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func (b *Builder) dialOptions(_ bool) []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
}

func (b *Builder) grpcClientConn(tls bool, dsn string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(dsn, b.dialOptions(tls)...)
	if err != nil {
		return nil, err
	}

	b.ender.Add(func(_ context.Context) error {
		return conn.Close()
	})

	return conn, nil
}
