package plugins

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpecList_Specs(t *testing.T) {
	t.Parallel()

	specs := []FuncSpec{
		{Name: "fn1", Description: "first"},
		{Name: "fn2", Group: "hooks"},
	}
	list := SpecList(specs)

	got := list.Specs()
	assert.Equal(t, specs, got)
}

func TestSpecList_Specs_Empty(t *testing.T) {
	t.Parallel()

	var list SpecList
	got := list.Specs()
	assert.Nil(t, got)
}

func TestSpecs(t *testing.T) {
	t.Parallel()

	specs := []FuncSpec{
		{Name: "a"},
		{Name: "b", Group: "g"},
	}
	provider := Specs(specs...)
	require.NotNil(t, provider)

	got := provider.Specs()
	assert.Equal(t, specs, got)
}

func TestSpecs_Variadic(t *testing.T) {
	t.Parallel()

	provider := Specs(
		FuncSpec{Name: "x"},
		FuncSpec{Name: "y", Description: "desc"},
	)
	got := provider.Specs()
	require.Len(t, got, 2)
	assert.Equal(t, "x", got[0].Name)
	assert.Equal(t, "y", got[1].Name)
	assert.Equal(t, "desc", got[1].Description)
}

func TestNewPlugin(t *testing.T) {
	t.Parallel()

	info := PluginInfo{
		Name:        "test-plugin",
		Version:     "1.0",
		Description: "test desc",
	}
	spec := FuncSpec{Name: "myFunc"}
	provider := Specs(spec)

	plugin := NewPlugin(info, provider)
	require.NotNil(t, plugin)

	assert.Equal(t, info, plugin.Info())
	providers := plugin.Providers()
	require.Len(t, providers, 1)
	assert.Equal(t, []FuncSpec{spec}, providers[0].Specs())
}

func TestNewPlugin_MultipleProviders(t *testing.T) {
	t.Parallel()

	info := PluginInfo{Name: "multi"}
	p1 := Specs(FuncSpec{Name: "a"})
	p2 := Specs(FuncSpec{Name: "b"}, FuncSpec{Name: "c"})

	plugin := NewPlugin(info, p1, p2)
	providers := plugin.Providers()
	require.Len(t, providers, 2)
	assert.Len(t, providers[0].Specs(), 1)
	assert.Len(t, providers[1].Specs(), 2)
}

func TestNewPlugin_NoProviders(t *testing.T) {
	t.Parallel()

	info := PluginInfo{Name: "empty"}
	plugin := NewPlugin(info)
	assert.Empty(t, plugin.Providers())
	assert.Equal(t, info, plugin.Info())
}

func TestPlugin_WorksWithRegistry(t *testing.T) {
	t.Parallel()

	rec := &recordRegistry{}
	info := PluginInfo{Name: "p1", Version: "0.1"}
	specs := []FuncSpec{
		{Name: "upper", Fn: func(s string) string { return s }},
	}
	plugin := NewPlugin(info, Specs(specs...))

	rec.AddPlugin(plugin.Info(), plugin.Providers())

	require.Len(t, rec.added, 1)
	assert.Equal(t, info, rec.added[0].info)
	assert.Len(t, rec.added[0].providers, 1)
	assert.Equal(t, "upper", rec.added[0].providers[0].Specs()[0].Name)
}

func TestFuncSpec_ZeroValue(t *testing.T) {
	t.Parallel()

	var spec FuncSpec
	assert.Empty(t, spec.Name)
	assert.Empty(t, spec.Group)
	assert.Nil(t, spec.Fn)
}

func TestPluginInfo_ZeroValue(t *testing.T) {
	t.Parallel()

	var info PluginInfo
	assert.Empty(t, info.Name)
	assert.Nil(t, info.Authors)
	assert.Nil(t, info.Depends)
}

func TestPluginWithFuncs_ZeroValue(t *testing.T) {
	t.Parallel()

	var pwf PluginWithFuncs
	assert.Empty(t, pwf.Plugin.Name)
	assert.Nil(t, pwf.Funcs)
}

// recordRegistry is a minimal Registry impl that records AddPlugin calls for testing.
type recordRegistry struct {
	added []struct {
		info      PluginInfo
		providers []SpecProvider
	}
}

func (r *recordRegistry) AddPlugin(info PluginInfo, providers []SpecProvider) {
	r.added = append(r.added, struct {
		info      PluginInfo
		providers []SpecProvider
	}{info, providers})
}

func (r *recordRegistry) Funcs() map[string]any { return nil }

func (r *recordRegistry) Plugins(context.Context) []PluginInfo { return nil }

func (r *recordRegistry) Groups(context.Context) []PluginWithFuncs { return nil }

func (r *recordRegistry) Hooks(string) []Func { return nil }
