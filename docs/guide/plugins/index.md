---
title: Plugins
---

# Plugins <VersionTag version="v3.5.0" />

Extend template functions with Go plugins.

::: info Built-in Faker <VersionTag version="v3.10.0" />
You do not need an external plugin for common fake data generation.
GripMock includes a built-in `faker` object (see [Dynamic Templates](/guide/stubs/dynamic-templates)).
Use custom plugins only for domain-specific logic that is not covered by built-ins.
:::

::: warning Requires a cgo build
`plugin.Open` only works in builds made with `CGO_ENABLED=1`:

| build | plugins |
| --- | --- |
| `bavix/gripmock:<tag>` | yes |
| `brew install --cask gripmock` | yes |
| release archives, `setup.sh` (glibc) | yes |
| `bavix/gripmock:<tag>-slim`, `gripmock-slim` cask, `gripmock-slim_*` archives | no |
| any windows build | no (Go has no plugin support there) |

On musl (Alpine) `setup.sh` installs the slim build, because the cgo build links
against glibc. Without cgo, `--plugins` logs
`plugin support is missing from this build` and the server keeps running without
them.
:::

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

`plugin.Open` compares the Go packages shared by the server and the plugin. They
match only when three things line up: the same Go minor version, the same
`-trimpath` setting, and the same paths the shared packages were compiled from.

For a release binary (homebrew, `setup.sh`, release archive) that means building
against the same module version, with `-trimpath`:

```bash
go mod init myplugin
go get github.com/bavix/gripmock/v3@v3.18.4   # the version gripmock --version reports
CGO_ENABLED=1 go build -trimpath -buildmode=plugin -o myplugin.so .
gripmock --plugins=./myplugin.so service.proto
```

For the docker image the paths come from the image instead, so build in the
matching `:<tag>-builder` and point the module at the source it ships. See
[Builder Image](./builder-image.md).

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
