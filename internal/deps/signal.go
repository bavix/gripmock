package deps

import (
	"context"
	"os/signal"
	"syscall"
)

func (*Builder) SignalNotify(ctx context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
}
