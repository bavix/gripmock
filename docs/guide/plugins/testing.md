---
title: Testing Plugins
---

# Testing Plugins <VersionTag version="v3.5.0" />

## Basic

```go
package main

import (
	"testing"
	"github.com/stretchr/testify/require"
	"github.com/bavix/gripmock/v3/pkg/plugintest"
)

func TestMyPlugin(t *testing.T) {
	reg := plugintest.NewRegistry()
	Register(reg)
	
	fn := plugintest.MustLookupFunc(t, reg, "myfunction")
	result := plugintest.MustCall(t, fn, "test")
	require.Equal(t, "processed: test", result)
}
```

## Floating Point

```go
fn := plugintest.MustLookupFunc(t, reg, "sqrt")
result := plugintest.MustCall(t, fn, 9.0)
require.InEpsilon(t, 3.0, result, 1e-9)
```

## Decorators

```go
reg.AddPlugin(plugintest.PluginInfo{Name: "gripmock"}, []plugintest.SpecProvider{
	plugintest.Specs(plugintest.FuncSpec{
		Name: "add",
		Fn:   baseAddFunction,
	}),
})
Register(reg)

fn := plugintest.MustLookupFunc(t, reg, "add")
result := plugintest.MustCall(t, fn, 1.0, 2.0)
require.InEpsilon(t, 4.0, result, 1e-9)
```

## Errors

```go
fn := plugintest.MustLookupFunc(t, reg, "divide")
_, err := plugintest.Call(t.Context(), fn, 10.0, 0.0)
require.Error(t, err)
```
