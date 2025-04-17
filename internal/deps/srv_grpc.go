package deps

import (
	"context"

	"github.com/bavix/gripmock/internal/app"
	"github.com/bavix/gripmock/internal/domain/proto"
)

func (b *Builder) GRPCServe(ctx context.Context, param *proto.Arguments) error {
	return app.NewGRPCServer(
		b.config.GRPCNetwork,
		b.config.GRPCAddr,
		param,
		b.Budgerigar(),
		b.Extender(),
	).Serve(ctx)
}
