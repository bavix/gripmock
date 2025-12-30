package plugins

import "context"

type Registry interface {
	AddPlugin(info PluginInfo, providers []SpecProvider)
	Funcs() map[string]any
	Plugins(ctx context.Context) []PluginInfo
	Groups(ctx context.Context) []PluginWithFuncs
	// Hooks returns functions whose Group equals the provided name.
	Hooks(group string) []Func
}

type FuncSpec struct {
	Name        string
	Fn          any
	Description string
	Group       string
	Decorates   string
	Replacement string
}

// Func is the canonical function signature for plugins.
type Func func(context.Context, ...any) (any, error)

type SpecProvider interface {
	Specs() []FuncSpec
}

type SpecList []FuncSpec

func (s SpecList) Specs() []FuncSpec { return s }

func Specs(specs ...FuncSpec) SpecProvider {
	return SpecList(specs)
}

type Plugin interface {
	Info() PluginInfo
	Providers() []SpecProvider
}

type pluginDef struct {
	info      PluginInfo
	providers []SpecProvider
}

func (p pluginDef) Info() PluginInfo          { return p.info }
func (p pluginDef) Providers() []SpecProvider { return p.providers }

func NewPlugin(info PluginInfo, providers ...SpecProvider) Plugin {
	return pluginDef{info: info, providers: providers}
}

type PluginInfo struct {
	Name         string   `json:"name"`
	Version      string   `json:"version,omitempty"`
	Source       string   `json:"source,omitempty"`
	Authors      []Author `json:"authors,omitempty"`
	Description  string   `json:"description,omitempty"`
	Depends      []string `json:"depends,omitempty"`
	Kind         string   `json:"kind,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

type Author struct {
	Name    string `json:"name"`
	Contact string `json:"contact,omitempty"`
}

type FunctionInfo struct {
	Name            string `json:"name"`
	Description     string `json:"description,omitempty"`
	Group           string `json:"group,omitempty"`
	Decorates       string `json:"decorates,omitempty"`
	DecoratesPlugin string `json:"decoratesPlugin,omitempty"`
	Replacement     string `json:"replacement,omitempty"`
	Deactivated     bool   `json:"deactivated,omitempty"`
}

type PluginWithFuncs struct {
	Plugin PluginInfo     `json:"plugin"`
	Funcs  []FunctionInfo `json:"funcs"`
}
