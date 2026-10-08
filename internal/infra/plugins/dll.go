package plugins

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"

	pkgplugins "github.com/bavix/gripmock/v3/pkg/plugins"
	"github.com/bavix/gripmock/v3/pkg/plugins/cshared"
)

var errPlugin = errors.New("plugin call failed")

// pluginError carries the plugin message unchanged, so a failure reads the same
// through both transports, and still matches errPlugin.
type pluginError struct {
	message string
}

func (e pluginError) Error() string { return e.message }

func (e pluginError) Is(target error) bool { return target == errPlugin }

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
			if fn.Name == "" {
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

// kinds records the numeric type of every argument, because JSON keeps only the
// value and the plugin needs the type the template passed.
func kinds(args []any) []string {
	out := make([]string, len(args))

	for i, arg := range args {
		if arg == nil {
			continue
		}

		kind := reflect.TypeOf(arg).Kind()
		if kind >= reflect.Int && kind <= reflect.Float32 {
			out[i] = kind.String()
		}
	}

	return out
}

// whole restores the integers the JSON round trip turned into floats, so a
// number renders the same through both transports. Only the result itself
// carries its Go kind, so values inside containers fall back to the shape of the
// number: integral means integer.
func whole(value any, isFloat bool) any {
	switch typed := value.(type) {
	case float64:
		if isFloat {
			return typed
		}

		return integral(typed)
	case []any:
		for i, item := range typed {
			typed[i] = whole(item, false)
		}

		return typed
	case map[string]any:
		for key, item := range typed {
			typed[key] = whole(item, false)
		}

		return typed
	default:
		return value
	}
}

func integral(value float64) any {
	if value != math.Trunc(value) || math.Abs(value) >= 1<<53 {
		return value
	}

	return int64(value)
}

// proxy turns an exported name into the canonical Func. Arguments and results
// travel as JSON, so numbers reach the plugin as float64 and come back as one.
func (p *dllPlugin) proxy(name string) pkgplugins.Func {
	return func(ctx context.Context, args ...any) (any, error) {
		err := ctx.Err()
		if err != nil {
			return nil, err
		}

		if args == nil {
			args = []any{}
		}

		encoded, err := json.Marshal(cshared.Request{Args: args, Kinds: kinds(args)})
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
			return nil, pluginError{message: response.Error}
		}

		var result any

		err = json.Unmarshal(response.Result, &result)
		if err != nil {
			return nil, fmt.Errorf("decode result: %w", err)
		}

		return whole(result, response.Float), nil
	}
}
