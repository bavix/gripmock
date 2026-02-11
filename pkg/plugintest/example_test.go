package plugintest_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/pkg/plugintest"
)

func TestHelloPlugin(t *testing.T) {
	t.Parallel()

	reg := plugintest.NewRegistryWith(
		plugintest.PluginInfo{Name: "demo"},
		plugintest.Specs(
			plugintest.FuncSpec{
				Name: "hello",
				Fn: func(_ context.Context, args ...any) (any, error) {
					if len(args) == 0 {
						return "hello", nil
					}
					s, ok := args[0].(string)
					if !ok {
						return nil, fmt.Errorf("expected string argument, got %T", args[0])
					}

					return "hello " + s, nil
				},
			},
		),
	)

	fn := plugintest.MustLookupFunc(t, reg, "hello")

	res := plugintest.MustCall(t, fn, "world")
	require.Equal(t, "hello world", res)

	resNoArgs := plugintest.MustCall(t, fn)
	require.Equal(t, "hello", resNoArgs)
}
