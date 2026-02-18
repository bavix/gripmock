package lifecycle_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/lifecycle"
)

type testLogger struct {
	errs []error
}

func (l *testLogger) Err(err error) {
	l.errs = append(l.errs, err)
}

var errBoom = errors.New("boom")

func TestManager_OrderAndClear(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	var calls []int

	m := lifecycle.New(nil)
	m.Add(
		func(context.Context) error {
			calls = append(calls, 1)

			return nil
		},
		nil, // should be ignored
		func(context.Context) error {
			calls = append(calls, 2)

			return nil
		},
	)

	m.Do(ctx)
	require.Equal(t, []int{2, 1}, calls)

	// Ensure second Do is a no-op.
	m.Do(ctx)
	require.Equal(t, []int{2, 1}, calls)
}

func TestManager_LogsErrors(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	logger := &testLogger{}
	m := lifecycle.New(logger)

	m.Add(func(context.Context) error {
		return errBoom
	})

	m.Do(ctx)

	require.Equal(t, []error{errBoom}, logger.errs)
}
