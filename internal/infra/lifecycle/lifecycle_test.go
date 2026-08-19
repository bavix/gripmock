package lifecycle_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/lifecycle"
)

var errBoom = errors.New("boom")

func TestManagerOrderAndClear(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	var calls []int

	m := lifecycle.New()
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

func TestManagerLogsErrors(t *testing.T) {
	t.Parallel()

	var sink bytes.Buffer

	logger := zerolog.New(&sink)
	ctx := logger.WithContext(t.Context())

	m := lifecycle.New()
	m.Add(func(context.Context) error {
		return errBoom
	})

	m.Do(ctx)

	require.Contains(t, sink.String(), "shutdown callback failed")
	require.Contains(t, sink.String(), errBoom.Error())
}
