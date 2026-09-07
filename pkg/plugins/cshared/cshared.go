// Package cshared is the plugin side of the c-shared transport: on Windows a
// plugin is a DLL and answers over two C entry points instead of Go symbols.
package cshared

import (
	"context"
	"encoding/json"

	"github.com/bavix/gripmock/v3/pkg/plugins"
	"github.com/bavix/gripmock/v3/pkg/plugintest"
)

type Manifest struct {
	Plugins []plugins.PluginWithFuncs `json:"plugins"`
}

type Response struct {
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

//nolint:gochecknoglobals
var registry = plugintest.NewRegistry()

// Use registers the plugin the entry points answer for. A DLL calls it once,
// from init, before the server can reach the entry points.
func Use(register func(plugins.Registry)) {
	reg := plugintest.NewRegistry()
	register(reg)

	registry = reg
}

func Describe() string {
	return marshal(Manifest{Plugins: registry.Groups(context.Background())})
}

func Call(name string, argsJSON string) string {
	fn, ok := registry.Funcs()[name].(plugins.Func)
	if !ok {
		return marshal(Response{Error: "unknown function: " + name})
	}

	var args []any

	decodeErr := json.Unmarshal([]byte(argsJSON), &args)
	if decodeErr != nil {
		return marshal(Response{Error: "decode args: " + decodeErr.Error()})
	}

	result, err := fn(context.Background(), args...)
	if err != nil {
		return marshal(Response{Error: err.Error()})
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		return marshal(Response{Error: "encode result: " + err.Error()})
	}

	return marshal(Response{Result: encoded})
}

func marshal(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return `{"error":"encode response"}`
	}

	return string(data)
}
