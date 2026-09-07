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

## End-to-end

Unit tests cover `Register`. They do not prove the built artifact loads: that
depends on the transport, the toolchain and, for `.so`, on matching build paths.
Build it and let the server answer:

```bash
gripmock plugin build ./path/to/plugin --out ./plugins/myplugin.so
gripmock info --plugins=./plugins
```

`info` prints every loaded plugin with its functions. A plugin that failed to
load is absent from that list and the reason is in the log.

For behaviour, run the server against a stub that calls the function and drive it
with [grpctestify](https://github.com/gripmock/grpctestify):

```bash
gripmock --plugins=./plugins --stub=./testdata ./testdata/service.proto &
gripmock check --timeout=60s --silent
grpctestify ./testdata/
```

This repository runs exactly that for `examples/plugins/*` on Linux, macOS and
Windows; the fixtures live in `third_party/plugins`.
