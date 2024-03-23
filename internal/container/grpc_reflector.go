package container

import (
	"context"

	"github.com/bavix/gripmock/internal/pkg/grpcreflector"
)

func (c *Container) GReflector(ctx context.Context) (*grpcreflector.GReflector, error) {
	if c.gRef != nil {
		return c.gRef, nil
	}

	client, err := c.grpcClient(ctx, c.conf.GRPCAddr())
	if err != nil {
		return nil, err
	}

	c.gRef = grpcreflector.New(client)

	return c.gRef, nil
}
