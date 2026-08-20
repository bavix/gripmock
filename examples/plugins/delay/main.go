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

func fibonacci(attempt, step any) string {
	unit, ok := toDuration(step)
	if !ok {
		return "0s"
	}

	prev, curr := 0, 1
	for range max(0, toInt(attempt)-1) {
		prev, curr = curr, prev+curr
	}

	return (time.Duration(curr) * unit).String()
}

func toDuration(v any) (time.Duration, bool) {
	text, ok := v.(string)
	if !ok {
		return 0, false
	}

	parsed, err := time.ParseDuration(text)

	return parsed, err == nil
}

func toInt(v any) int {
	switch value := v.(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}
