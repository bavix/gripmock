package deps

import (
	"context"

	"github.com/bavix/gripmock/internal/app"
	"github.com/bavix/gripmock/internal/domain/proto"
)

func (b *Builder) GRPCServe(ctx context.Context, param *proto.ProtocParam) error {
	return app.NewGRPCServer(param).Serve(ctx)
}
