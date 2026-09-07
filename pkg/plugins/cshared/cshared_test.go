package cshared_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/pkg/plugins"
	"github.com/bavix/gripmock/v3/pkg/plugins/cshared"
)

var errBoom = errors.New("boom")

func TestMain(m *testing.M) {
	cshared.Use(register)
	os.Exit(m.Run())
}

func register(reg plugins.Registry) {
	reg.AddPlugin(plugins.PluginInfo{
		Name:    "sample",
		Version: "v1.0.0",
		Kind:    "external",
	}, []plugins.SpecProvider{
		plugins.Specs(
			plugins.FuncSpec{Name: "upper", Fn: strings.ToUpper, Description: "upper case", Group: "text"},
			plugins.FuncSpec{
				Name: "boom",
				Fn: func(_ context.Context, _ ...any) (any, error) {
					return nil, errBoom
				},
			},
			plugins.FuncSpec{
				Name:      "decorated",
				Decorates: "@gripmock/add",
				Fn: func(base plugins.Func) plugins.Func {
					return base
				},
			},
		),
	})
}

// A decorator targets a function living in the server, which the DLL cannot call
// back into, so it never reaches the manifest.
func TestDescribeListsFuncsWithoutDecorators(t *testing.T) {
	t.Parallel()

	var manifest cshared.Manifest

	require.NoError(t, json.Unmarshal([]byte(cshared.Describe()), &manifest))
	require.Len(t, manifest.Plugins, 1)
	require.Equal(t, "sample", manifest.Plugins[0].Plugin.Name)

	names := make([]string, 0, len(manifest.Plugins[0].Funcs))
	for _, fn := range manifest.Plugins[0].Funcs {
		names = append(names, fn.Name)
	}

	require.ElementsMatch(t, []string{"upper", "boom"}, names)
}

func TestCall(t *testing.T) {
	t.Parallel()

	for name, expect := range map[string]struct {
		call  string
		fails bool
	}{
		"result":           {cshared.Call("upper", `["abc"]`), false},
		"unknown function": {cshared.Call("nope", `[]`), true},
		"plugin error":     {cshared.Call("boom", `[]`), true},
		"broken args":      {cshared.Call("upper", `{`), true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var response cshared.Response

			require.NoError(t, json.Unmarshal([]byte(expect.call), &response))

			if expect.fails {
				require.NotEmpty(t, response.Error)
				require.Empty(t, response.Result)

				return
			}

			require.Empty(t, response.Error)
			require.JSONEq(t, `"ABC"`, string(response.Result))
		})
	}
}
