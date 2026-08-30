package template

import (
	"testing"

	"github.com/stretchr/testify/require"

	internalplugins "github.com/bavix/gripmock/v3/internal/infra/plugins"
	pkgplugins "github.com/bavix/gripmock/v3/pkg/plugins"
)

func TestEngineWithoutRegistryAnswersWithLoadedPlugins(t *testing.T) {
	t.Parallel()

	internalplugins.Default().AddPlugin(
		pkgplugins.PluginInfo{Name: "probe", Source: "test", Kind: "external"},
		[]pkgplugins.SpecProvider{pkgplugins.Specs(pkgplugins.FuncSpec{
			Name: "probeShout",
			Fn:   func(s string) string { return s + "!" },
		})},
	)

	engine := New(t.Context(), nil)

	out, err := engine.Render(`{{ probeShout "hi" }}`, Data{})
	require.NoError(t, err)
	require.Equal(t, "hi!", out)
}
