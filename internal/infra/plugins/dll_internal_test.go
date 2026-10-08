package plugins

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	pkgplugins "github.com/bavix/gripmock/v3/pkg/plugins"
	"github.com/bavix/gripmock/v3/pkg/plugins/cshared"
)

var errBroken = errors.New("broken")

func TestMain(m *testing.M) {
	cshared.Use(samplePlugin)
	os.Exit(m.Run())
}

func samplePlugin(reg pkgplugins.Registry) {
	reg.AddPlugin(pkgplugins.PluginInfo{Name: "sample", Version: "v1.0.0", Kind: "external"},
		[]pkgplugins.SpecProvider{
			pkgplugins.Specs(
				pkgplugins.FuncSpec{Name: "upper", Fn: strings.ToUpper, Description: "upper case"},
				pkgplugins.FuncSpec{Name: "sum", Fn: func(a, b float64) float64 { return a + b }},
				pkgplugins.FuncSpec{Name: "count", Fn: func(s string) int { return len(s) }},
				pkgplugins.FuncSpec{Name: "half", Fn: func(v float64) float64 { return v / 2 }},
				pkgplugins.FuncSpec{Name: "twice", Fn: func(n int) int { return n * 2 }},
				pkgplugins.FuncSpec{Name: "nested", Fn: func() any {
					return map[string]any{"id": 891568578, "ratio": 2.5, "tags": []any{1, "a"}}
				}},
				pkgplugins.FuncSpec{
					Name: "boom",
					Fn: func(_ context.Context, _ ...any) (any, error) {
						return nil, errBroken
					},
				},
			),
		})
}

// linked wires the server half straight onto the plugin half, so the whole
// contract runs in one process and only the C boundary is left out.
func linked(t *testing.T) (*Registry, context.Context) {
	t.Helper()

	reg := NewRegistry()
	ctx := t.Context()
	plug := &dllPlugin{path: "sample.dll", invoke: func(name string, argsJSON string) (string, error) {
		return cshared.Call(ctx, name, argsJSON), nil
	}}

	require.NoError(t, plug.register(ctx, reg, cshared.Describe()))

	return reg, ctx
}

func TestDLLRegistersManifest(t *testing.T) {
	t.Parallel()

	reg, ctx := linked(t)

	infos := reg.Plugins(ctx)
	require.Len(t, infos, 1)
	require.Equal(t, "sample", infos[0].Name)
	require.Equal(t, "sample.dll", infos[0].Source)

	funcs := reg.Funcs()
	for _, name := range []string{"upper", "sum", "boom"} {
		require.Contains(t, funcs, name)
	}
}

func TestDLLCallsCrossTheContract(t *testing.T) {
	t.Parallel()

	reg, ctx := linked(t)

	upper, ok := reg.Funcs()["upper"].(pkgplugins.Func)
	require.True(t, ok)

	out, err := upper(ctx, "abc")
	require.NoError(t, err)
	require.Equal(t, "ABC", out)

	sum, ok := reg.Funcs()["sum"].(pkgplugins.Func)
	require.True(t, ok)

	total, err := sum(ctx, 2.0, 3.0)
	require.NoError(t, err)
	require.InEpsilon(t, 5.0, total, 1e-9)
}

func TestDLLKeepsIntegersWhole(t *testing.T) {
	t.Parallel()

	reg, ctx := linked(t)

	count, ok := reg.Funcs()["count"].(pkgplugins.Func)
	require.True(t, ok)

	out, err := count(ctx, "abc")
	require.NoError(t, err)
	require.Equal(t, int64(3), out)

	half, ok := reg.Funcs()["half"].(pkgplugins.Func)
	require.True(t, ok)

	ratio, err := half(ctx, 9.0)
	require.NoError(t, err)
	require.InEpsilon(t, 4.5, ratio, 1e-9)

	whole, err := half(ctx, 8.0)
	require.NoError(t, err)
	require.InEpsilon(t, 4.0, whole, 1e-9)
	require.IsType(t, float64(0), whole)
}

func TestDLLKeepsArgumentKinds(t *testing.T) {
	t.Parallel()

	reg, ctx := linked(t)

	twice, ok := reg.Funcs()["twice"].(pkgplugins.Func)
	require.True(t, ok)

	out, err := twice(ctx, 21)
	require.NoError(t, err)
	require.Equal(t, int64(42), out)

	sum, ok := reg.Funcs()["sum"].(pkgplugins.Func)
	require.True(t, ok)

	_, err = sum(ctx, 2, 3)
	require.ErrorContains(t, err, "have int want float64")
}

func TestDLLKeepsIntegersInsideContainers(t *testing.T) {
	t.Parallel()

	reg, ctx := linked(t)

	nested, ok := reg.Funcs()["nested"].(pkgplugins.Func)
	require.True(t, ok)

	out, err := nested(ctx)
	require.NoError(t, err)

	value, ok := out.(map[string]any)
	require.True(t, ok)
	require.Equal(t, int64(891568578), value["id"])
	require.InEpsilon(t, 2.5, value["ratio"], 1e-9)
	require.Equal(t, []any{int64(1), "a"}, value["tags"])
}

func TestDLLPropagatesPluginError(t *testing.T) {
	t.Parallel()

	reg, ctx := linked(t)

	boom, ok := reg.Funcs()["boom"].(pkgplugins.Func)
	require.True(t, ok)

	_, err := boom(ctx)
	require.ErrorIs(t, err, errPlugin)
	require.ErrorContains(t, err, "broken")
}

func TestDLLRejectsBrokenPayloads(t *testing.T) {
	t.Parallel()

	plug := &dllPlugin{path: "sample.dll", invoke: func(string, string) (string, error) {
		return "{", nil
	}}

	require.Error(t, plug.register(t.Context(), NewRegistry(), "{"))

	_, err := plug.proxy("upper")(t.Context(), "abc")
	require.Error(t, err)
}
