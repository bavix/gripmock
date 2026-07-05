package plugintest_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/pkg/plugintest"
)

func TestMustHelpers(t *testing.T) {
	t.Parallel()

	const (
		a = 1
		b = 2
		c = 3
	)

	reg := plugintest.NewRegistry()
	reg.AddPlugin(plugintest.PluginInfo{Name: "p"}, []plugintest.SpecProvider{
		plugintest.Specs(plugintest.FuncSpec{
			Name: "ok",
			Fn: func(a ...any) any {
				return len(a)
			},
		}),
	})

	fn := plugintest.MustLookupFunc(t, reg, "ok")
	res := plugintest.MustCall(t, fn, a, b, c)
	require.Equal(t, 3, res)
}
