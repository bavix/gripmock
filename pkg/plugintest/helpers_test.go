package plugintest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMustHelpers(t *testing.T) {
	t.Parallel()

	const (
		a = 1
		b = 2
		c = 3
	)

	reg := NewRegistry()
	reg.AddPlugin(PluginInfo{Name: "p"}, []SpecProvider{
		Specs(FuncSpec{
			Name: "ok",
			Fn: func(a ...any) any {
				return len(a)
			},
		}),
	})

	fn := MustLookupFunc(t, reg, "ok")
	res := MustCall(t, fn, a, b, c)
	require.Equal(t, 3, res)
}
