package deps

import "github.com/bavix/gripmock/pkg/grpcreflector"

func (b *Builder) reflector() (*grpcreflector.GReflector, error) {
	conn, err := b.grpcClientConn(false, b.config.GRPCAddr)
	if err != nil {
		return nil, err
	}

	return grpcreflector.New(conn), nil
}
