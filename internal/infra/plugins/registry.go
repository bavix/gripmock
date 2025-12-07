package plugins

import (
	"context"
	"maps"
	"sort"
	"strings"
	"sync"

	"github.com/rs/zerolog"

	pkgplugins "github.com/bavix/gripmock/v3/pkg/plugins"
)

type Registry struct {
	mu sync.RWMutex

	funcs     map[string]any
	funcOwner map[string]string

	plugins     map[string]pkgplugins.PluginInfo
	pluginOrder []string
	pluginFuncs map[string][]pkgplugins.FunctionInfo
	pluginDeps  map[string][]string

	forceSource string
	logCtx      context.Context
}

type Option func(*Registry)

func WithForceSource(src string) Option {
	return func(r *Registry) {
		r.forceSource = src
	}
}

func WithContext(ctx context.Context) Option {
	return func(r *Registry) {
		r.logCtx = ctx
	}
}

func NewRegistry(opts ...Option) *Registry {
	r := &Registry{
		funcs:       make(map[string]any),
		funcOwner:   make(map[string]string),
		plugins:     make(map[string]pkgplugins.PluginInfo),
		pluginFuncs: make(map[string][]pkgplugins.FunctionInfo),
		pluginDeps:  make(map[string][]string),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

func (r *Registry) AddPlugin(info pkgplugins.PluginInfo, providers []pkgplugins.SpecProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()

	info = r.normalizeInfo(info)
	r.ensurePlugin(info)

	for _, provider := range providers {
		if provider == nil {
			continue
		}

		r.addProvider(info, provider)
	}
}

func (r *Registry) Funcs() map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	copyMap := make(map[string]any, len(r.funcs))
	maps.Copy(copyMap, r.funcs)

	return copyMap
}

func (r *Registry) Plugins() []pkgplugins.PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	order, skipped := r.sortedPluginOrder()
	if len(skipped) > 0 {
		if logger := zerolog.Ctx(r.logCtx); logger != nil {
			logger.Warn().Strs("plugins", skipped).Msg("plugin dependency cycle detected; skipping")
		}
	}

	result := make([]pkgplugins.PluginInfo, 0, len(order))
	for _, name := range order {
		info := r.plugins[name]
		if deps, ok := r.pluginDeps[name]; ok {
			info.Depends = append([]string(nil), deps...)
		}

		result = append(result, info)
	}

	return result
}

func (r *Registry) Groups() []pkgplugins.PluginWithFuncs {
	r.mu.RLock()
	defer r.mu.RUnlock()

	order, skipped := r.sortedPluginOrder()
	if len(skipped) > 0 {
		if logger := zerolog.Ctx(r.logCtx); logger != nil {
			logger.Warn().Strs("plugins", skipped).Msg("plugin dependency cycle detected; skipping")
		}
	}

	result := make([]pkgplugins.PluginWithFuncs, 0, len(order))
	for _, name := range order {
		info := r.plugins[name]
		if deps, ok := r.pluginDeps[name]; ok {
			info.Depends = append([]string(nil), deps...)
		}

		result = append(result, pkgplugins.PluginWithFuncs{
			Plugin: info,
			Funcs:  r.pluginFuncs[name],
		})
	}

	return result
}

// Hooks returns functions filtered by Group.
func (r *Registry) Hooks(group string) []pkgplugins.Func {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if group == "" {
		return nil
	}

	funcs := make([]pkgplugins.Func, 0)
	for _, infoList := range r.pluginFuncs {
		for _, f := range infoList {
			if f.Group != group {
				continue
			}

			if fn, ok := r.funcs[f.Name]; ok {
				if wrapped := wrapFunc(fn); wrapped != nil {
					funcs = append(funcs, wrapped)
				}
			}
		}
	}

	return funcs
}

func (r *Registry) normalizeInfo(info pkgplugins.PluginInfo) pkgplugins.PluginInfo {
	if r.forceSource != "" {
		info.Source = r.forceSource
	}

	if info.Kind == "" && info.Source == "external" {
		info.Kind = "external"
	}

	if len(info.Capabilities) == 0 {
		info.Capabilities = []string{"template-funcs"}
	}

	return info
}

func (r *Registry) ensurePlugin(info pkgplugins.PluginInfo) {
	if _, ok := r.plugins[info.Name]; ok {
		return
	}

	r.plugins[info.Name] = info
	r.pluginOrder = append(r.pluginOrder, info.Name)
}

func (r *Registry) addProvider(info pkgplugins.PluginInfo, provider pkgplugins.SpecProvider) {
	for _, spec := range provider.Specs() {
		r.addSpec(info, spec)
	}
}

func (r *Registry) addSpec(info pkgplugins.PluginInfo, spec pkgplugins.FuncSpec) {
	if spec.Name == "" {
		return
	}

	name := spec.Name
	targetName, decorPlugin := parseDecorates(spec)

	infoEntry := pkgplugins.FunctionInfo{
		Name:        name,
		Description: spec.Description,
		Group:       spec.Group,
		Replacement: spec.Replacement,
	}

	if prev, ok := r.funcOwner[name]; ok && spec.Decorates == "" {
		r.warnDuplicate(name, info.Name, prev)

		infoEntry.Deactivated = true
		infoEntry.Decorates = ""
		infoEntry.DecoratesPlugin = ""
		r.pluginFuncs[info.Name] = append(r.pluginFuncs[info.Name], infoEntry)

		return
	}

	fn := r.applyDecorator(targetName, spec)
	if fn == nil {
		return
	}

	r.funcs[name] = fn

	if decorPlugin != "" {
		infoEntry.Decorates = targetName
		infoEntry.DecoratesPlugin = decorPlugin
		r.addDepend(info.Name, decorPlugin)
	} else if prev, ok := r.funcOwner[name]; ok {
		infoEntry.Decorates = name
		infoEntry.DecoratesPlugin = prev
	}

	r.pluginFuncs[info.Name] = append(r.pluginFuncs[info.Name], infoEntry)
	r.funcOwner[name] = info.Name
}

// Groups method defined above

func (r *Registry) applyDecorator(target string, spec pkgplugins.FuncSpec) pkgplugins.Func {
	// When Decorates is set, Fn is treated as a decorator.
	if spec.Decorates != "" {
		dec := wrapDecorator(spec.Fn)
		if dec == nil {
			return nil
		}

		if existing, ok := r.funcs[target]; ok {
			if wrapped := wrapFunc(existing); wrapped != nil {
				return dec(wrapped)
			}
		}

		return nil
	}

	return wrapFunc(spec.Fn)
}

func parseDecorates(spec pkgplugins.FuncSpec) (string, string) {
	raw := strings.TrimSpace(spec.Decorates)
	if raw == "" {
		return spec.Name, ""
	}

	if after, ok := strings.CutPrefix(raw, "@"); ok {
		raw = after
	}

	const splitLimit = 2
	if strings.Contains(raw, "/") {
		parts := strings.SplitN(raw, "/", splitLimit)
		if len(parts) == splitLimit {
			return parts[1], parts[0]
		}
	}

	// fallback: single token => same plugin
	return raw, ""
}

func (r *Registry) addDepend(pluginName, depend string) {
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

func (r *Registry) sortedPluginOrder() ([]string, []string) {
	registered, graph, indegree := r.buildGraph()
	queue := r.zeroIndegreeQueue(indegree)

	ordered := make([]string, 0, len(registered))
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]

		ordered = append(ordered, n)

		for _, next := range graph[n] {
			indegree[next]--
			if indegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	skipped := r.collectCycles(registered, indegree, len(ordered))

	return ordered, skipped
}

func (r *Registry) buildGraph() (map[string]struct{}, map[string][]string, map[string]int) {
	registered := make(map[string]struct{}, len(r.pluginOrder))
	for _, name := range r.pluginOrder {
		registered[name] = struct{}{}
	}

	indegree := make(map[string]int, len(r.pluginOrder))
	graph := make(map[string][]string, len(r.pluginDeps))

	for _, name := range r.pluginOrder {
		for _, dep := range r.pluginDeps[name] {
			if _, ok := registered[dep]; !ok {
				continue
			}

			graph[dep] = append(graph[dep], name)
			indegree[name]++
		}
	}

	return registered, graph, indegree
}

func (r *Registry) zeroIndegreeQueue(indegree map[string]int) []string {
	queue := make([]string, 0, len(r.pluginOrder))
	for _, name := range r.pluginOrder {
		if indegree[name] == 0 {
			queue = append(queue, name)
		}
	}

	return queue
}

func (r *Registry) collectCycles(registered map[string]struct{}, indegree map[string]int, orderedCount int) []string {
	if orderedCount == len(registered) {
		return nil
	}

	skipped := make([]string, 0, len(registered)-orderedCount)
	for name := range registered {
		if indegree[name] > 0 {
			skipped = append(skipped, name)
		}
	}

	sort.Strings(skipped)

	return skipped
}

func (r *Registry) warnDuplicate(name, plugin, existing string) {
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
