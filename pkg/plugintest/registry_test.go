package plugintest_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/pkg/plugintest"
)

func TestRegistry_AddPluginAndLookup(t *testing.T) {
	t.Parallel()

	reg := plugintest.NewRegistry()

	reg.AddPlugin(plugintest.PluginInfo{Name: "p1"}, []plugintest.SpecProvider{
		plugintest.Specs(
			plugintest.FuncSpec{
				Name: "sum",
				Fn: func(args ...any) any {
					total := 0
					for _, a := range args {
						if n, ok := a.(int); ok {
							total += n
						}
					}
					return total
				},
			},
			plugintest.FuncSpec{
				Name: "echo",
				Fn: func(_ ...any) any {
					return "ok"
				},
			},
		),
	})

	fn, ok := plugintest.LookupFunc(reg, "sum")
	require.True(t, ok)
	out, err := plugintest.Call(context.Background(), fn, 1, 2, 3)
	require.NoError(t, err)
	require.Equal(t, 6, out)

	fnEcho, ok := plugintest.LookupFunc(reg, "echo")
	require.True(t, ok)
	outEcho, err := plugintest.Call(context.Background(), fnEcho)
	require.NoError(t, err)
	require.Equal(t, "ok", outEcho)

	pluginsMeta := reg.Plugins()
	require.Len(t, pluginsMeta, 1)
	require.Equal(t, "p1", pluginsMeta[0].Name)

	groups := reg.Groups()
	require.Len(t, groups, 1)
	require.Len(t, groups[0].Funcs, 2)
}
