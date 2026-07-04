package plugintest

import (
	"context"
	"slices"
	"sort"
	"strings"

	"github.com/rs/zerolog"

	"github.com/bavix/gripmock/v3/pkg/plugins"
)

// TestRegistry is a lightweight, in-memory Registry implementation dedicated to
// tests. It mirrors the public contract yet stays free of production-only
// behavior (sources, trust), letting tests register and look up plugin
// functions deterministically.
type TestRegistry struct {
	funcs       map[string]plugins.Func
	funcOwner   map[string]string
	plugins     []plugins.PluginInfo
	pluginFuncs map[string][]plugins.FunctionInfo
	pluginDeps  map[string][]string
}

// NewRegistry creates an empty TestRegistry with no preloaded plugins.
func NewRegistry() *TestRegistry {
	return &TestRegistry{
		funcs:       make(map[string]plugins.Func),
		funcOwner:   make(map[string]string),
		pluginFuncs: make(map[string][]plugins.FunctionInfo),
		pluginDeps:  make(map[string][]string),
	}
}

// NewRegistryWith creates a registry and immediately registers provided plugins,
// a convenience to keep tests concise.
func NewRegistryWith(info plugins.PluginInfo, providers ...plugins.SpecProvider) *TestRegistry {
	reg := NewRegistry()
	reg.AddPlugin(info, providers)

	return reg
}

// AddPlugin registers plugin specs without applying production priorities. The
// latest registration for a name wins unless a decorator chains onto an existing
// function.
func (r *TestRegistry) AddPlugin(info plugins.PluginInfo, providers []plugins.SpecProvider) {
	r.plugins = append(r.plugins, info)

	for _, provider := range providers {
		if provider == nil {
			continue
		}

		r.addProvider(info, provider)
	}
}

func (r *TestRegistry) addProvider(info plugins.PluginInfo, provider plugins.SpecProvider) {
	for _, spec := range provider.Specs() {
		r.addSpec(info, spec)
	}
}

//nolint:cyclop,funlen
func (r *TestRegistry) addSpec(info plugins.PluginInfo, spec plugins.FuncSpec) {
	if spec.Name == "" {
		return
	}

	targetName, decorPlugin := parseDecorates(spec)
	hasDecor := strings.TrimSpace(spec.Decorates) != ""

	entry := plugins.FunctionInfo{
		Name:        spec.Name,
		Description: spec.Description,
		Group:       spec.Group,
		Replacement: spec.Replacement,
	}

	if prev, ok := r.funcOwner[spec.Name]; ok && !hasDecor {
		r.warnDuplicate(spec.Name, info.Name, prev)
		entry.Deactivated = true
		entry.Decorates = ""
		entry.DecoratesPlugin = ""
		r.pluginFuncs[info.Name] = append(r.pluginFuncs[info.Name], entry)

		return
	}

	var fn plugins.Func
	if hasDecor {
		dec := wrapDecorator(spec.Fn)
		if dec == nil {
			return
		}

		existing, ok := r.funcs[targetName]
		if !ok {
			return
		}

		base := Wrap(existing)
		if base == nil {
			return
		}

		fn = dec(base)
	} else {
		fn = Wrap(spec.Fn)
		if fn == nil {
			return
		}
	}

	r.funcs[spec.Name] = fn

	if decorPluginPresent(decorPlugin) {
		entry.Decorates = targetName
		entry.DecoratesPlugin = decorPlugin
		r.addDepend(info.Name, decorPlugin)
	} else if prev, ok := r.funcOwner[spec.Name]; ok {
		entry.Decorates = spec.Name
		entry.DecoratesPlugin = prev
	}

	r.funcOwner[spec.Name] = info.Name
	r.pluginFuncs[info.Name] = append(r.pluginFuncs[info.Name], entry)
}

func decorPluginPresent(p string) bool { return strings.TrimSpace(p) != "" }

func wrapDecorator(fn any) func(plugins.Func) plugins.Func {
	switch f := fn.(type) {
	case func(plugins.Func) plugins.Func:
		return f
	case func(func(context.Context, ...any) (any, error)) func(context.Context, ...any) (any, error):
		return func(base plugins.Func) plugins.Func {
			return f(base)
		}
	default:
		return nil
	}
}

func parseDecorates(spec plugins.FuncSpec) (string, string) {
	raw := strings.TrimSpace(spec.Decorates)
	if raw == "" {
		return spec.Name, ""
	}

	raw, _ = strings.CutPrefix(raw, "@")

	if strings.Contains(raw, "/") {
		parts := strings.SplitN(raw, "/", 2)
		if len(parts) == 2 {
			return parts[1], parts[0]
		}
	}

	return raw, ""
}

func (r *TestRegistry) addDepend(pluginName, depend string) {
	if depend == "" {
		return
	}

	deps := r.pluginDeps[pluginName]
	seen := make(map[string]struct{}, len(deps)+1)
	for _, d := range deps {
		seen[d] = struct{}{}
	}

	if _, exists := seen[depend]; exists {
		return
	}

	r.pluginDeps[pluginName] = append(deps, depend)
}

func (r *TestRegistry) warnDuplicate(name, plugin, existing string) {
	// Test registry doesn't log warnings
}

// Funcs implements Registry.Funcs and returns a copy safe for mutation in tests.
func (r *TestRegistry) Funcs() map[string]any {
	out := make(map[string]any, len(r.funcs))
	for k, v := range r.funcs {
		out[k] = v
	}

	return out
}

// Plugins returns shallow-copied plugin metadata for assertions.
func (r *TestRegistry) Plugins(ctx context.Context) []plugins.PluginInfo {
	order, skipped := r.sortedPluginOrder()
	if len(skipped) > 0 {
		logger := zerolog.Ctx(ctx)
		if logger != nil {
			logger.Warn().Strs("plugins", skipped).Msg("plugin dependency cycle detected; skipping")
		}
	}

	out := make([]plugins.PluginInfo, 0, len(order))
	for _, name := range order {
		p := r.lookupPlugin(name)
		if deps, ok := r.pluginDeps[p.Name]; ok {
			p.Depends = slices.Clone(deps)
		}
		out = append(out, p)
	}

	return out
}

// Groups returns plugin info alongside functions to support info-oriented tests.
func (r *TestRegistry) Groups(ctx context.Context) []plugins.PluginWithFuncs {
	order, skipped := r.sortedPluginOrder()
	if len(skipped) > 0 {
		logger := zerolog.Ctx(ctx)
		if logger != nil {
			logger.Warn().Strs("plugins", skipped).Msg("plugin dependency cycle detected; skipping")
		}
	}

	res := make([]plugins.PluginWithFuncs, 0, len(order))
	for _, name := range order {
		info := r.lookupPlugin(name)
		if deps, ok := r.pluginDeps[info.Name]; ok {
			info.Depends = slices.Clone(deps)
		}

		res = append(res, plugins.PluginWithFuncs{
			Plugin: info,
			Funcs:  r.pluginFuncs[info.Name],
		})
	}

	return res
}

// Hooks returns functions filtered by Group.
func (r *TestRegistry) Hooks(group string) []plugins.Func {
	if strings.TrimSpace(group) == "" {
		return nil
	}

	res := make([]plugins.Func, 0)
	for _, funcs := range r.pluginFuncs {
		for _, f := range funcs {
			if f.Group != group {
				continue
			}

			if fn, ok := r.funcs[f.Name]; ok {
				res = append(res, fn)
			}
		}
	}

	return res
}

func (r *TestRegistry) lookupPlugin(name string) plugins.PluginInfo {
	for _, p := range r.plugins {
		if p.Name == name {
			return p
		}
	}

	return plugins.PluginInfo{Name: name}
}

func (r *TestRegistry) sortedPluginOrder() ([]string, []string) {
	var ordered, skipped []string

	visited := make(map[string]int)
	cycle := make(map[string]bool)
	registered := make(map[string]bool, len(r.plugins))
	for _, p := range r.plugins {
		registered[p.Name] = true
	}

	var visit func(string) bool
	visit = func(name string) bool {
		if visited[name] == 2 {
			return cycle[name]
		}
		if visited[name] == 1 {
			return true
		}

		visited[name] = 1
		inCycle := false
		for _, dep := range r.pluginDeps[name] {
			if visit(dep) {
				inCycle = true
			}
		}
		visited[name] = 2

		if inCycle {
			cycle[name] = true
		} else if registered[name] {
			ordered = append(ordered, name)
		}

		return inCycle
	}

	for _, p := range r.plugins {
		visit(p.Name)
		if cycle[p.Name] {
			skipped = append(skipped, p.Name)
		}
	}

	sort.Strings(skipped)

	return ordered, skipped
}
