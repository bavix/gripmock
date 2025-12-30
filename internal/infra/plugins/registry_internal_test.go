package plugins

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgplugins "github.com/bavix/gripmock/v3/pkg/plugins"
)

func TestNewRegistry(t *testing.T) {
	t.Parallel()

	// Arrange & Act
	reg := NewRegistry()

	// Assert
	require.NotNil(t, reg)
	assert.NotNil(t, reg.funcs)
	assert.NotNil(t, reg.plugins)
}

func TestNewRegistry_WithForceSource(t *testing.T) {
	t.Parallel()

	// Arrange
	forceSource := "test-source"

	// Act
	reg := NewRegistry(WithForceSource(forceSource))

	// Assert
	require.NotNil(t, reg)
	assert.Equal(t, forceSource, reg.forceSource)
}

func TestRegistry_AddPlugin(t *testing.T) {
	t.Parallel()

	// Arrange
	reg := NewRegistry()
	info := pkgplugins.PluginInfo{
		Name:   "test-plugin",
		Source: "test",
		Kind:   "external",
	}
	specs := []pkgplugins.SpecProvider{
		pkgplugins.Specs(
			pkgplugins.FuncSpec{
				Name:        "testFunc",
				Fn:          func() string { return "test" }, //nolint:goconst // test value
				Description: "test function",
			},
		),
	}

	// Act
	reg.AddPlugin(info, specs)

	// Assert
	assert.Contains(t, reg.plugins, "test-plugin")
	assert.Contains(t, reg.funcs, "testFunc")
}

func TestRegistry_Funcs(t *testing.T) {
	t.Parallel()

	// Arrange
	reg := NewRegistry()
	info := pkgplugins.PluginInfo{
		Name:   "test-plugin",
		Source: "test",
		Kind:   "external",
	}
	specs := []pkgplugins.SpecProvider{
		pkgplugins.Specs(
			pkgplugins.FuncSpec{
				Name: "testFunc",
				Fn:   func() string { return "test" },
			},
		),
	}
	reg.AddPlugin(info, specs)

	// Act
	funcs := reg.Funcs()

	// Assert
	assert.Contains(t, funcs, "testFunc")
	assert.Len(t, funcs, 1)
}

func TestRegistry_Plugins(t *testing.T) {
	t.Parallel()

	// Arrange
	reg := NewRegistry()
	info1 := pkgplugins.PluginInfo{
		Name:   "plugin1",
		Source: "test1",
		Kind:   "external",
	}
	info2 := pkgplugins.PluginInfo{
		Name:   "plugin2",
		Source: "test2",
		Kind:   "external",
	}

	reg.AddPlugin(info1, nil)
	reg.AddPlugin(info2, nil)

	// Act
	plugins := reg.Plugins(context.Background())

	// Assert
	assert.Len(t, plugins, 2)
}

func TestRegistry_Groups(t *testing.T) {
	t.Parallel()

	// Arrange
	reg := NewRegistry()
	info := pkgplugins.PluginInfo{
		Name:   "test-plugin",
		Source: "test",
		Kind:   "external",
	}
	specs := []pkgplugins.SpecProvider{
		pkgplugins.Specs(
			pkgplugins.FuncSpec{
				Name:  "testFunc",
				Fn:    func() string { return "test" },
				Group: "test-group",
			},
		),
	}
	reg.AddPlugin(info, specs)

	// Act
	groups := reg.Groups(context.Background())

	// Assert
	assert.Len(t, groups, 1)
	assert.Equal(t, "test-plugin", groups[0].Plugin.Name)
	assert.Len(t, groups[0].Funcs, 1)
}

func TestRegistry_Hooks(t *testing.T) {
	t.Parallel()

	// Arrange
	reg := NewRegistry()
	info := pkgplugins.PluginInfo{
		Name:   "test-plugin",
		Source: "test",
		Kind:   "external",
	}
	specs := []pkgplugins.SpecProvider{
		pkgplugins.Specs(
			pkgplugins.FuncSpec{
				Name:  "hookFunc",
				Fn:    func() string { return "hook" },
				Group: "hooks",
			},
		),
	}
	reg.AddPlugin(info, specs)

	// Act
	hooks := reg.Hooks("hooks")

	// Assert
	assert.Len(t, hooks, 1)
}

func TestRegistry_Hooks_EmptyGroup(t *testing.T) {
	t.Parallel()

	// Arrange
	reg := NewRegistry()

	// Act
	hooks := reg.Hooks("")

	// Assert
	assert.Nil(t, hooks)
}

func TestRegistry_NormalizeInfo_ForceSource(t *testing.T) {
	t.Parallel()

	// Arrange
	forceSource := "forced-source"
	reg := NewRegistry(WithForceSource(forceSource))
	info := pkgplugins.PluginInfo{
		Name:   "test-plugin",
		Source: "original-source",
		Kind:   "external",
	}

	// Act
	normalized := reg.normalizeInfo(info)

	// Assert
	assert.Equal(t, forceSource, normalized.Source)
}

func TestRegistry_NormalizeInfo_DefaultKind(t *testing.T) {
	t.Parallel()

	// Arrange
	reg := NewRegistry()
	info := pkgplugins.PluginInfo{
		Name:   "test-plugin",
		Source: "external",
	}

	// Act
	normalized := reg.normalizeInfo(info)

	// Assert
	assert.Equal(t, "external", normalized.Kind)
}

func TestRegistry_NormalizeInfo_DefaultCapabilities(t *testing.T) {
	t.Parallel()

	// Arrange
	reg := NewRegistry()
	info := pkgplugins.PluginInfo{
		Name:   "test-plugin",
		Source: "test",
		Kind:   "external",
	}

	// Act
	normalized := reg.normalizeInfo(info)

	// Assert
	assert.Equal(t, []string{"template-funcs"}, normalized.Capabilities)
}

func TestRegistry_AddDepend(t *testing.T) {
	t.Parallel()

	// Arrange
	reg := NewRegistry()

	// Act
	reg.addDepend("plugin1", "plugin2")
	reg.addDepend("plugin1", "plugin3")

	// Assert
	deps := reg.pluginDeps["plugin1"]
	assert.Contains(t, deps, "plugin2")
	assert.Contains(t, deps, "plugin3")
}

func TestRegistry_AddDepend_Duplicate(t *testing.T) {
	t.Parallel()

	// Arrange
	reg := NewRegistry()

	// Act
	reg.addDepend("plugin1", "plugin2")
	reg.addDepend("plugin1", "plugin2")

	// Assert
	deps := reg.pluginDeps["plugin1"]
	assert.Len(t, deps, 1)
}

func TestRegistry_ParseDecorates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		spec     pkgplugins.FuncSpec
		expected string
		plugin   string
	}{
		{
			name: "empty decorates",
			spec: pkgplugins.FuncSpec{
				Name:      "func1",
				Decorates: "",
			},
			expected: "func1",
			plugin:   "",
		},
		{
			name: "with @ prefix",
			spec: pkgplugins.FuncSpec{
				Name:      "func1",
				Decorates: "@plugin/func2",
			},
			expected: "func2",
			plugin:   "plugin",
		},
		{
			name: "without @ prefix",
			spec: pkgplugins.FuncSpec{
				Name:      "func1",
				Decorates: "plugin/func2",
			},
			expected: "func2",
			plugin:   "plugin",
		},
		{
			name: "single token",
			spec: pkgplugins.FuncSpec{
				Name:      "func1",
				Decorates: "func2",
			},
			expected: "func2",
			plugin:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange - test case data is already set up

			// Act
			target, plugin := parseDecorates(tt.spec)

			// Assert
			assert.Equal(t, tt.expected, target)
			assert.Equal(t, tt.plugin, plugin)
		})
	}
}

func TestRegistry_SortedPluginOrder(t *testing.T) {
	t.Parallel()

	// Arrange
	reg := NewRegistry()
	info1 := pkgplugins.PluginInfo{Name: "plugin1", Source: "test1", Kind: "external"}
	info2 := pkgplugins.PluginInfo{Name: "plugin2", Source: "test2", Kind: "external"}

	reg.AddPlugin(info1, nil)
	reg.AddPlugin(info2, nil)

	// Act
	order, skipped := reg.sortedPluginOrder()

	// Assert
	assert.Len(t, order, 2)
	assert.Nil(t, skipped)
}

func TestRegistry_SortedPluginOrder_WithDependencies(t *testing.T) {
	t.Parallel()

	// Arrange
	reg := NewRegistry()
	info1 := pkgplugins.PluginInfo{Name: "plugin1", Source: "test1", Kind: "external"}
	info2 := pkgplugins.PluginInfo{Name: "plugin2", Source: "test2", Kind: "external"}

	reg.AddPlugin(info1, nil)
	reg.AddPlugin(info2, nil)
	reg.addDepend("plugin2", "plugin1")

	// Act
	order, skipped := reg.sortedPluginOrder()

	// Assert
	assert.Len(t, order, 2)
	assert.Nil(t, skipped)
	assert.Equal(t, "plugin1", order[0])
	assert.Equal(t, "plugin2", order[1])
}

func TestRegistry_ZeroIndegreeQueue(t *testing.T) {
	t.Parallel()

	// Arrange
	reg := NewRegistry()
	indegree := map[string]int{
		"plugin1": 0,
		"plugin2": 1,
		"plugin3": 0,
	}
	reg.pluginOrder = []string{"plugin1", "plugin2", "plugin3"}

	// Act
	queue := reg.zeroIndegreeQueue(indegree)

	// Assert
	assert.Contains(t, queue, "plugin1")
	assert.Contains(t, queue, "plugin3")
	assert.NotContains(t, queue, "plugin2")
}

func TestRegistry_CollectCycles(t *testing.T) {
	t.Parallel()

	// Arrange
	reg := NewRegistry()
	registered := map[string]struct{}{
		"plugin1": {},
		"plugin2": {},
		"plugin3": {},
	}
	indegree := map[string]int{
		"plugin1": 0,
		"plugin2": 1,
		"plugin3": 1,
	}
	orderedCount := 1

	// Act
	skipped := reg.collectCycles(registered, indegree, orderedCount)

	// Assert
	assert.Len(t, skipped, 2)
	assert.Contains(t, skipped, "plugin2")
	assert.Contains(t, skipped, "plugin3")
}

func TestRegistry_CollectCycles_NoCycles(t *testing.T) {
	t.Parallel()

	// Arrange
	reg := NewRegistry()
	registered := map[string]struct{}{
		"plugin1": {},
		"plugin2": {},
	}
	indegree := map[string]int{
		"plugin1": 0,
		"plugin2": 0,
	}
	orderedCount := 2

	// Act
	skipped := reg.collectCycles(registered, indegree, orderedCount)

	// Assert
	assert.Nil(t, skipped)
}
