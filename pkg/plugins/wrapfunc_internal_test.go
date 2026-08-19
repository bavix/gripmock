package plugins

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

var errPluginFailed = errors.New("plugin failed")

func TestWrapFuncKeepsCanonicalShapes(t *testing.T) {
	t.Parallel()

	canonical := Func(func(_ context.Context, args ...any) (any, error) {
		return args[0], nil
	})

	got, err := WrapFunc(canonical, failFallback(t))(t.Context(), "kept")

	require.NoError(t, err)
	require.Equal(t, "kept", got)
}

func TestWrapFuncAdaptsContextlessShapes(t *testing.T) {
	t.Parallel()

	t.Run("value only", func(t *testing.T) {
		t.Parallel()

		fn := func(args ...any) any { return args[0] }

		got, err := WrapFunc(fn, failFallback(t))(t.Context(), "value")

		require.NoError(t, err)
		require.Equal(t, "value", got)
	})

	t.Run("value and error", func(t *testing.T) {
		t.Parallel()

		fn := func(_ ...any) (any, error) { return nil, errPluginFailed }

		_, err := WrapFunc(fn, failFallback(t))(t.Context(), "ignored")

		require.ErrorIs(t, err, errPluginFailed,
			"an error from the plugin must reach the caller, not be swallowed by the adapter")
	})
}

func TestWrapFuncPassesEveryArgument(t *testing.T) {
	t.Parallel()

	fn := func(args ...any) any { return args }

	got, err := WrapFunc(fn, failFallback(t))(t.Context(), 1, "two", true)

	require.NoError(t, err)
	require.Equal(t, []any{1, "two", true}, got)
}

func TestWrapFuncSendsUnknownShapesToTheFallback(t *testing.T) {
	t.Parallel()

	original := func(a int) int { return a }

	var received any

	wrapped := WrapFunc(original, func(fn any) Func {
		received = fn

		return func(_ context.Context, _ ...any) (any, error) { return "from fallback", nil }
	})

	got, err := wrapped(t.Context())

	require.NoError(t, err)
	require.Equal(t, "from fallback", got)
	require.NotNil(t, received, "the fallback must receive the original function to reflect over")
}

func failFallback(t *testing.T) func(any) Func {
	t.Helper()

	return func(any) Func {
		t.Fatal("a shape WrapFunc understands must not reach the fallback")

		return nil
	}
}
