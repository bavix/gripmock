package plugintest_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/pkg/plugintest"
)

func TestSpyFunc_NilBase(t *testing.T) {
	t.Parallel()

	spy := plugintest.NewSpy(nil)
	require.Nil(t, spy.Func())
}

func TestSpyRecordsCalls(t *testing.T) {
	t.Parallel()

	base := func(ctx context.Context, args ...any) (any, error) {
		return len(args), nil
	}

	spy := plugintest.NewSpy(base)
	fn := spy.Func()

	out1, err := fn(context.Background(), 1, 2)
	require.NoError(t, err)
	require.Equal(t, 2, out1)

	out2, err := fn(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, out2)

	require.Len(t, spy.Calls, 2)
	require.Equal(t, []any{1, 2}, spy.Calls[0].Args)
	require.Equal(t, []any{}, spy.Calls[1].Args)
}

func TestSpyDecorator(t *testing.T) {
	t.Parallel()

	base := func(ctx context.Context, args ...any) (any, error) {
		return "from-base", nil
	}

	spy := plugintest.NewSpy(nil)
	decorate := spy.Decorator()
	wrapped := decorate(base)

	res, err := wrapped(context.Background(), "arg1")
	require.NoError(t, err)
	require.Equal(t, "from-base", res)
	require.Len(t, spy.Calls, 1)
	require.Equal(t, []any{"arg1"}, spy.Calls[0].Args)
}

func TestSpyReset(t *testing.T) {
	t.Parallel()

	base := func(ctx context.Context, args ...any) (any, error) { return nil, nil }
	spy := plugintest.NewSpy(base)
	fn := spy.Func()

	_, _ = fn(context.Background())
	require.Len(t, spy.Calls, 1)

	spy.Reset()
	require.Empty(t, spy.Calls)
}
