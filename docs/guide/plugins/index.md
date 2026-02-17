---
title: Plugins
---

# Plugins <VersionTag version="v3.5.0" />

Extend template functions with Go plugins.

## Create

```go
package main

import "github.com/bavix/gripmock/v3/pkg/plugins"

func Register(reg plugins.Registry) {
	reg.AddPlugin(plugins.PluginInfo{
		Name:         "myplugin",
		Version:      "v1.0.0",
		Kind:         "external",
		Capabilities: []string{"template-funcs"},
	}, []plugins.SpecProvider{
		plugins.Specs(
			plugins.FuncSpec{
				Name:        "myfunction",
				Fn:          myFunction,
				Description: "Does something",
			},
		),
	})
}

func myFunction(s string) string {
	return "processed: " + s
}
```

## Build & Load

```bash
go build -buildmode=plugin -o myplugin.so ./path/to/plugin
gripmock --plugins=./myplugin.so service.proto
```

For production-like compatibility, build plugins with the matching `:<tag>-builder` image and run with `:<tag>`. See [Builder Image](./builder-image.md).

## Use

::: v-pre
```yaml
output:
  data:
    hash: "{{.Request.data | sha256}}"
```
:::

## Examples

`examples/plugins/`: hash, math

## Related

- [Advanced](./advanced.md) - Decorators
- [Testing](./testing.md) - Tests
- [Builder Image](./builder-image.md) - Compatibility model
