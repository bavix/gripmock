package main

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/pkg/plugintest"
)

func TestMathPlugin(t *testing.T) {
	t.Parallel()

	reg := plugintest.NewRegistry()

	reg.AddPlugin(plugintest.PluginInfo{Name: "gripmock"}, []plugintest.SpecProvider{
		plugintest.Specs(plugintest.FuncSpec{
			Name: "add",
			Fn:   baseAddRealistic,
		}),
	})

	Register(reg)

	ctx := context.Background()

	fnSqrt, ok := plugintest.LookupFunc(reg, "sqrt")
	require.True(t, ok, "sqrt not registered")

	outSqrt, err := plugintest.Call(ctx, fnSqrt, 9.0)
	require.NoError(t, err)
	require.InEpsilon(t, math.Sqrt(9.0), outSqrt, 1e-9)

	fnAdd, ok := plugintest.LookupFunc(reg, "add")
	require.True(t, ok)

	outAdd, err := plugintest.Call(ctx, fnAdd, 1.0, 2.0)
	require.NoError(t, err)
	require.InEpsilon(t, 4.0, outAdd, 1e-9) // 1+2 then decorator adds +1
}

func baseAddRealistic(_ context.Context, args ...any) (any, error) {
	sum := 0.0

	for _, raw := range args {
		switch v := raw.(type) {
		case float64:
			sum += v
		case float32:
			sum += float64(v)
		case int:
			sum += float64(v)
		case int64:
			sum += float64(v)
		default:
			return nil, errInvalidArgs
		}
	}

	return sum, nil
}
