package deps

import (
	"context"

	"github.com/cockroachdb/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	infraTLS "github.com/bavix/gripmock/v3/internal/infra/tls"
)

func (b *Builder) grpcClientConn(useTLS bool, dsn string) (*grpc.ClientConn, error) {
	transportCreds := insecure.NewCredentials()

	if useTLS {
		clientCfg, err := b.grpcTLSConfig().BuildClientTLSConfig(dsn)
		if err != nil {
			return nil, err
		}

		transportCreds = credentials.NewTLS(clientCfg)
	}

	conn, err := grpc.NewClient(
		dsn,
		grpc.WithTransportCredentials(transportCreds),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create gRPC client connection")
	}

	b.ender.Add(func(_ context.Context) error {
		return conn.Close()
	})

	return conn, nil
}

func (b *Builder) grpcTLSEnabled() bool {
	return b.grpcTLSConfig().IsClientEnabled()
}

func (b *Builder) grpcTLSConfig() infraTLS.TLSConfig {
	return infraTLS.TLSConfig{
		CertFile:   b.config.GRPCTLSCertFile,
		KeyFile:    b.config.GRPCTLSKeyFile,
		CAFile:     b.config.GRPCTLSCAFile,
		MinVersion: b.config.GRPCTLSMinVersion,
	}
}
