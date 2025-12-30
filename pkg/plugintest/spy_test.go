package plugintest_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/pkg/plugintest"
)

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
