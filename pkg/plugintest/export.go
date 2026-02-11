package plugintest

import "github.com/bavix/gripmock/v3/pkg/plugins"

// Re-export public plugin API for tests so plugin tests don't have to import pkg/plugins.
type (
	Registry        = plugins.Registry
	Func            = plugins.Func
	FuncSpec        = plugins.FuncSpec
	SpecProvider    = plugins.SpecProvider
	SpecList        = plugins.SpecList
	PluginInfo      = plugins.PluginInfo
	PluginWithFuncs = plugins.PluginWithFuncs
	FunctionInfo    = plugins.FunctionInfo
	Plugin          = plugins.Plugin
	Author          = plugins.Author
)

// Specs forwards to plugins.Specs so tests can stay within plugintest imports.
//
//nolint:ireturn
func Specs(specs ...FuncSpec) SpecProvider {
	return plugins.Specs(specs...)
}

// NewPlugin forwards to plugins.NewPlugin for fixtures that prefer the full
// Plugin object instead of raw registry calls.
//
//nolint:ireturn
func NewPlugin(info PluginInfo, providers ...SpecProvider) Plugin {
	return plugins.NewPlugin(info, providers...)
}
