package plugintest_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/pkg/plugintest"
)

func TestDecorate(t *testing.T) {
	t.Parallel()

	base := plugintest.Func(func(_ context.Context, args ...any) (any, error) {
		if len(args) == 0 {
			return "", nil
		}

		s, _ := args[0].(string)

		return s + "-base", nil
	})

	decorated := plugintest.Decorate(base, func(next plugintest.Func) plugintest.Func {
		return func(ctx context.Context, args ...any) (any, error) {
			res, err := next(ctx, args...)
			if err != nil {
				return nil, err
			}

			return res.(string) + "-decorated", nil //nolint:forcetypeassert
		}
	})

	res, err := decorated(t.Context(), "x")
	require.NoError(t, err)
	require.Equal(t, "x-base-decorated", res)
}
