package shutdown_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/shutdown"
)

var (
	ErrSame  = errors.New("some error")
	ErrSame1 = errors.New("some error 1")
	ErrSame2 = errors.New("some error 2")
	ErrSame3 = errors.New("some error 3")
)

// logger type removed as it's not used in simplified version

func TestShutdown_LoggerNil(t *testing.T) {
	t.Parallel()

	var val atomic.Bool

	s := shutdown.New(nil)
	s.Add(func(_ context.Context) error {
		val.Store(true)

		return ErrSame
	})

	s.Do(t.Context())

	require.True(t, val.Load())
}

func TestShutdown_Stack_ErrorAll(t *testing.T) {
	t.Parallel()
	// Simplified version doesn't have logger, so we just test that it doesn't panic
	s := shutdown.New(nil)
	s.Add(
		func(_ context.Context) error { return ErrSame1 },
		func(_ context.Context) error { return ErrSame2 },
		func(_ context.Context) error { return ErrSame3 },
	)

	// Should not panic
	s.Do(t.Context())
}

func TestShutdown_Stack_Error(t *testing.T) {
	t.Parallel()
	// Simplified version doesn't have logger, so we just test that it doesn't panic
	s := shutdown.New(nil)
	s.Add(
		func(_ context.Context) error { return ErrSame1 },
		func(_ context.Context) error { return nil },
		func(_ context.Context) error { return ErrSame3 },
	)

	// Should not panic
	s.Do(t.Context())
}
