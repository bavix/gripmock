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
	plug := &dllPlugin{path: "sample.dll", invoke: func(name string, argsJSON string) (string, error) {
		return cshared.Call(name, argsJSON), nil
	}}

	ctx := context.Background()
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

	total, err := sum(ctx, 2, 3)
	require.NoError(t, err)
	require.InEpsilon(t, 5.0, total, 1e-9)
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

	require.Error(t, plug.register(context.Background(), NewRegistry(), "{"))

	_, err := plug.proxy("upper")(context.Background(), "abc")
	require.Error(t, err)
}
