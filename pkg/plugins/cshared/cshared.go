// Package cshared is the plugin side of the c-shared transport: on Windows a
// plugin is a DLL and answers over two C entry points instead of Go symbols.
package cshared

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/bavix/gripmock/v3/internal/infra/funcwrap"
	"github.com/bavix/gripmock/v3/pkg/plugins"
)

type Manifest struct {
	Plugins []plugins.PluginWithFuncs `json:"plugins"`
}

type Request struct {
	Args  []any    `json:"args"`
	Kinds []string `json:"kinds,omitempty"`
}

type Response struct {
	Result json.RawMessage `json:"result,omitempty"`
	Float  bool            `json:"float,omitempty"`
	Error  string          `json:"error,omitempty"`
}

//nolint:gochecknoglobals
var registry = &collector{funcs: map[string]plugins.Func{}}

// Use registers the plugin the entry points answer for. A DLL calls it once,
// from init, before the server can reach the entry points.
func Use(register func(plugins.Registry)) {
	reg := &collector{funcs: map[string]plugins.Func{}}
	register(reg)

	registry = reg
}

func Describe() string {
	return marshal(Manifest{Plugins: registry.entries})
}

func Call(ctx context.Context, name string, requestJSON string) string {
	fn, ok := registry.funcs[name]
	if !ok {
		return marshal(Response{Error: "unknown function: " + name})
	}

	var request Request

	decodeErr := json.Unmarshal([]byte(requestJSON), &request)
	if decodeErr != nil {
		return marshal(Response{Error: "decode args: " + decodeErr.Error()})
	}

	result, err := fn(ctx, restore(request)...)
	if err != nil {
		return marshal(Response{Error: err.Error()})
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		return marshal(Response{Error: "encode result: " + err.Error()})
	}

	return marshal(Response{Result: encoded, Float: isFloat(result)})
}

// restore puts back the integer kinds the JSON round trip flattened to float64,
// so a plugin sees the argument types the server passed.
func restore(request Request) []any {
	for i, kind := range request.Kinds {
		if i >= len(request.Args) || kind == "" {
			continue
		}

		value, ok := request.Args[i].(float64)
		if !ok {
			continue
		}

		request.Args[i] = number(value, kind)
	}

	return request.Args
}

func number(value float64, kind string) any {
	shapes := []any{int(0), int8(0), int16(0), int32(0), int64(0), uint(0), uint8(0), uint16(0), uint32(0), uint64(0), float32(0)}

	for _, shape := range shapes {
		typ := reflect.TypeOf(shape)
		if typ.Kind().String() == kind {
			return reflect.ValueOf(value).Convert(typ).Interface()
		}
	}

	return value
}

func isFloat(v any) bool {
	switch v.(type) {
	case float32, float64:
		return true
	default:
		return false
	}
}

func marshal(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return `{"error":"encode response"}`
	}

	return string(data)
}

type collector struct {
	entries []plugins.PluginWithFuncs
	funcs   map[string]plugins.Func
}

func (c *collector) AddPlugin(info plugins.PluginInfo, providers []plugins.SpecProvider) {
	entry := plugins.PluginWithFuncs{Plugin: info}

	for _, provider := range providers {
		if provider == nil {
			continue
		}

		for _, spec := range provider.Specs() {
			if spec.Name == "" || spec.Decorates != "" {
				continue
			}

			fn := plugins.WrapFunc(spec.Fn, func(fn any) plugins.Func { return funcwrap.WrapReflect(fn) })
			if fn == nil {
				continue
			}

			if _, exists := c.funcs[spec.Name]; exists {
				continue
			}

			c.funcs[spec.Name] = fn
			entry.Funcs = append(entry.Funcs, plugins.FunctionInfo{
				Name:        spec.Name,
				Description: spec.Description,
				Group:       spec.Group,
				Replacement: spec.Replacement,
			})
		}
	}

	c.entries = append(c.entries, entry)
}

func (c *collector) Funcs() map[string]any {
	out := make(map[string]any, len(c.funcs))
	for name, fn := range c.funcs {
		out[name] = fn
	}

	return out
}

func (c *collector) Plugins(context.Context) []plugins.PluginInfo {
	out := make([]plugins.PluginInfo, 0, len(c.entries))
	for _, entry := range c.entries {
		out = append(out, entry.Plugin)
	}

	return out
}

func (c *collector) Groups(context.Context) []plugins.PluginWithFuncs { return c.entries }

func (c *collector) Hooks(group string) []plugins.Func {
	var out []plugins.Func

	for _, entry := range c.entries {
		for _, info := range entry.Funcs {
			if group != "" && info.Group == group {
				out = append(out, c.funcs[info.Name])
			}
		}
	}

	return out
}
