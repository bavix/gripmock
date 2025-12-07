package plugintest

import (
	"context"
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
	logCtx      context.Context
}

// NewRegistry creates an empty TestRegistry with no preloaded plugins.
func NewRegistry() *TestRegistry {
	return &TestRegistry{
		funcs:       make(map[string]plugins.Func),
		funcOwner:   make(map[string]string),
		pluginFuncs: make(map[string][]plugins.FunctionInfo),
		pluginDeps:  make(map[string][]string),
		logCtx:      context.Background(),
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

	if decorPlugin != "" {
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

func parseDecorates(spec plugins.FuncSpec) (targetName string, targetPlugin string) {
	raw := strings.TrimSpace(spec.Decorates)
	if raw == "" {
		return spec.Name, ""
	}

	if strings.HasPrefix(raw, "@") {
		raw = strings.TrimPrefix(raw, "@")
	}

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
	logger := zerolog.Ctx(r.logCtx)
	if logger == nil {
		return
	}

	logger.Warn().
		Str("plugin", plugin).
		Str("function", name).
		Str("owner", existing).
		Msg("function ignored (implicit override); use Decorates=@owner/function to decorate explicitly")
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
func (r *TestRegistry) Plugins() []plugins.PluginInfo {
	order, skipped := r.sortedPluginOrder()
	if len(skipped) > 0 {
		logger := zerolog.Ctx(r.logCtx)
		if logger != nil {
			logger.Warn().Strs("plugins", skipped).Msg("plugin dependency cycle detected; skipping")
		}
	}

	out := make([]plugins.PluginInfo, 0, len(order))
	for _, name := range order {
		p := r.lookupPlugin(name)
		if deps, ok := r.pluginDeps[p.Name]; ok {
			p.Depends = append([]string(nil), deps...)
		}
		out = append(out, p)
	}

	return out
}

// Groups returns plugin info alongside functions to support info-oriented tests.
func (r *TestRegistry) Groups() []plugins.PluginWithFuncs {
	order, skipped := r.sortedPluginOrder()
	if len(skipped) > 0 {
		logger := zerolog.Ctx(r.logCtx)
		if logger != nil {
			logger.Warn().Strs("plugins", skipped).Msg("plugin dependency cycle detected; skipping")
		}
	}

	res := make([]plugins.PluginWithFuncs, 0, len(order))
	for _, name := range order {
		info := r.lookupPlugin(name)
		if deps, ok := r.pluginDeps[info.Name]; ok {
			info.Depends = append([]string(nil), deps...)
		}

		res = append(res, plugins.PluginWithFuncs{
			Plugin: info,
			Funcs:  r.pluginFuncs[info.Name],
		})
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

func (r *TestRegistry) sortedPluginOrder() (ordered []string, skipped []string) {
	state := make(map[string]int, len(r.plugins))
	inCycle := make(map[string]struct{})
	registered := make(map[string]struct{}, len(r.plugins))
	for _, p := range r.plugins {
		registered[p.Name] = struct{}{}
	}

	var stack []string
	var dfs func(string)
	dfs = func(n string) {
		if state[n] == 1 {
			inCycle[n] = struct{}{}
			for i := len(stack) - 1; i >= 0; i-- {
				inCycle[stack[i]] = struct{}{}
				if stack[i] == n {
					break
				}
			}
			return
		}

		if state[n] == 2 {
			return
		}

		state[n] = 1
		stack = append(stack, n)

		for _, dep := range r.pluginDeps[n] {
			dfs(dep)
		}

		stack = stack[:len(stack)-1]
		state[n] = 2

		if _, cyc := inCycle[n]; cyc {
			return
		}

		if _, ok := registered[n]; ok {
			ordered = append(ordered, n)
		}
	}

	for _, p := range r.plugins {
		dfs(p.Name)
	}

	if len(inCycle) > 0 {
		skipped = make([]string, 0, len(inCycle))
		for name := range inCycle {
			if _, ok := registered[name]; ok {
				skipped = append(skipped, name)
			}
		}
		sort.Strings(skipped)
	}

	return ordered, skipped
}
