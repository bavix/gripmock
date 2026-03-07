package plugintest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecorate(t *testing.T) {
	t.Parallel()

	base := Func(func(_ context.Context, args ...any) (any, error) {
		if len(args) == 0 {
			return "", nil
		}

		s, _ := args[0].(string)

		return s + "-base", nil
	})

	decorated := Decorate(base, func(next Func) Func {
		return func(ctx context.Context, args ...any) (any, error) {
			res, err := next(ctx, args...)
			if err != nil {
				return nil, err
			}

			return res.(string) + "-decorated", nil
		}
	})

	res, err := decorated(t.Context(), "x")
	require.NoError(t, err)
	require.Equal(t, "x-base-decorated", res)
}
