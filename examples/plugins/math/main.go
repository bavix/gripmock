package main

import (
	"context"
	"errors"
	"math"

	"github.com/rs/zerolog"

	"github.com/bavix/gripmock/v3/pkg/plugins"
)

func Register(reg plugins.Registry) {
	reg.AddPlugin(plugins.PluginInfo{
		Name:         "math",
		Source:       "examples/plugins/math",
		Version:      "v0.9.1",
		Kind:         "external",
		Capabilities: []string{"template-funcs"},
		Authors: []plugins.Author{
			{Name: "Bob Lee", Contact: "math-team@example.com"},
			{Name: "Eve Doe"},
		},
	}, []plugins.SpecProvider{
		plugins.Specs(
			plugins.FuncSpec{Name: "pow", Fn: math.Pow, Description: "math power"},
			plugins.FuncSpec{Name: "abs", Fn: math.Abs, Description: "math absolute"},
			plugins.FuncSpec{Name: "sqrt", Fn: math.Sqrt, Description: "square root"},
			plugins.FuncSpec{
				Name:        "round",
				Fn:          roundDeactivated,
				Description: "round to nearest int (deactivated)",
			},
			plugins.FuncSpec{
				Name:      "add",
				Decorates: "@gripmock/add",
				Fn: func(base func(context.Context, ...any) (any, error)) func(context.Context, ...any) (any, error) {
					return func(ctx context.Context, args ...any) (any, error) {
						val, err := base(ctx, args...)
						if err != nil {
							return nil, err
						}

						switch v := val.(type) {
						case float64:
							return v + 1, nil
						case int:
							return v + 1, nil
						case int64:
							return v + 1, nil
						default:
							return val, nil
						}
					}
				},
				Description: "decorated add (plus one)",
			},
		),
	})
}

var (
	errInvalidArgs = errors.New("invalid args")
	errDeactivated = errors.New("function deactivated")
)

func roundDeactivated(ctx context.Context, _ ...any) (any, error) {
	logger := zerolog.Ctx(ctx)
	if logger != nil {
		logger.Error().Err(errDeactivated).Msg("math.round is deactivated")
	}

	return nil, errDeactivated
}
