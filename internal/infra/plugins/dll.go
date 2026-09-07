package plugins

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	pkgplugins "github.com/bavix/gripmock/v3/pkg/plugins"
	"github.com/bavix/gripmock/v3/pkg/plugins/cshared"
)

var errPlugin = errors.New("plugin call failed")

// dllPlugin is the transport-neutral half of the c-shared plugin: the C entry
// points are reduced to invoke, everything else is JSON.
type dllPlugin struct {
	path   string
	invoke func(name string, argsJSON string) (string, error)
}

func (p *dllPlugin) register(ctx context.Context, reg pkgplugins.Registry, manifestJSON string) error {
	var manifest cshared.Manifest

	err := json.Unmarshal([]byte(manifestJSON), &manifest)
	if err != nil {
		return fmt.Errorf("decode manifest: %w", err)
	}

	for _, entry := range manifest.Plugins {
		info := entry.Plugin
		if info.Source == "" {
			info.Source = p.path
		}

		specs := make([]pkgplugins.FuncSpec, 0, len(entry.Funcs))

		for _, fn := range entry.Funcs {
			if fn.Name == "" || fn.Deactivated {
				continue
			}

			specs = append(specs, pkgplugins.FuncSpec{
				Name:        fn.Name,
				Description: fn.Description,
				Group:       fn.Group,
				Replacement: fn.Replacement,
				Fn:          p.proxy(fn.Name),
			})
		}

		if !existsPlugin(ctx, reg, info.Name) {
			reg.AddPlugin(info, []pkgplugins.SpecProvider{pkgplugins.Specs(specs...)})
		}
	}

	return nil
}

// proxy turns an exported name into the canonical Func. Arguments and results
// travel as JSON, so numbers reach the plugin as float64 and come back as one.
func (p *dllPlugin) proxy(name string) pkgplugins.Func {
	return func(_ context.Context, args ...any) (any, error) {
		if args == nil {
			args = []any{}
		}

		encoded, err := json.Marshal(args)
		if err != nil {
			return nil, fmt.Errorf("encode args: %w", err)
		}

		raw, err := p.invoke(name, string(encoded))
		if err != nil {
			return nil, err
		}

		var response cshared.Response

		err = json.Unmarshal([]byte(raw), &response)
		if err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}

		if response.Error != "" {
			return nil, fmt.Errorf("%w: %s", errPlugin, response.Error)
		}

		var result any

		err = json.Unmarshal(response.Result, &result)
		if err != nil {
			return nil, fmt.Errorf("decode result: %w", err)
		}

		return result, nil
	}
}
