package cmd

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestShutdownGracefullyStopsSignalHandlerBeforeShutdown(t *testing.T) {
	t.Parallel()

	parent, cancelParent := context.WithCancel(context.Background())
	t.Cleanup(cancelParent)

	var order []string

	stop := func() {
		order = append(order, "stop")

		cancelParent()
	}

	shutdown := func(ctx context.Context) {
		order = append(order, "shutdown")

		require.NoError(t, ctx.Err())

		deadline, ok := ctx.Deadline()
		require.True(t, ok, "shutdown must be bounded by a deadline")
		require.Positive(t, time.Until(deadline))
	}

	shutdownGracefully(parent, stop, shutdown, time.Minute)

	require.Equal(t, []string{"stop", "shutdown"}, order)
}

func TestShutdownGracefullyFallsBackToDefaultTimeout(t *testing.T) {
	t.Parallel()

	var remaining time.Duration

	shutdown := func(ctx context.Context) {
		deadline, ok := ctx.Deadline()
		require.True(t, ok, "shutdown must be bounded by a deadline")

		remaining = time.Until(deadline)
	}

	shutdownGracefully(context.Background(), func() {}, shutdown, 0)

	require.Positive(t, remaining)
	require.LessOrEqual(t, remaining, defaultShutdownTimeout)
}
