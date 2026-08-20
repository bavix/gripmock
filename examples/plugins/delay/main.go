package main

import (
	"time"

	"github.com/bavix/gripmock/v3/pkg/plugins"
)

func Register(reg plugins.Registry) {
	reg.AddPlugin(plugins.PluginInfo{
		Name:         "delay",
		Source:       "examples/plugins/delay",
		Version:      "v1.0.0",
		Kind:         "external",
		Capabilities: []string{"template-funcs"},
	}, []plugins.SpecProvider{
		plugins.Specs(
			plugins.FuncSpec{Name: "fibonacci", Fn: fibonacci, Description: "fibonacci delay curve"},
		),
	})
}

func fibonacci(attempt int, step string) string {
	unit, err := time.ParseDuration(step)
	if err != nil {
		return "0s"
	}

	prev, curr := 0, 1
	for range max(0, attempt-1) {
		prev, curr = curr, prev+curr
	}

	return (time.Duration(curr) * unit).String()
}
