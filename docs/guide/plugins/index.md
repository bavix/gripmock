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
A plugin is a `.so` loaded through `plugin.Open`, except on Windows, where Go has
neither `plugin.Open` nor `-buildmode=plugin` and the plugin is a `.dll` instead.
Either way the server has to be a `CGO_ENABLED=1` build:

| build | plugins |
| --- | --- |
| `bavix/gripmock:<tag>` | yes, `.so` |
| `brew install --cask gripmock` | yes, `.so` |
| release archives, `setup.sh` (glibc) | yes, `.so` |
| windows archives | yes, `.dll` |
| `bavix/gripmock:<tag>-slim`, `gripmock-slim` cask, `gripmock-slim_*` archives | no |

On musl (Alpine) `setup.sh` installs the slim build, because the cgo build links
against glibc. Where plugins are unavailable, `--plugins` logs
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

`gripmock plugin build` produces the artifact for the platform it runs on. The
plugin source is the same everywhere:

```bash
gripmock plugin build ./path/to/plugin --out myplugin.so
gripmock --plugins=./myplugin.so service.proto
```

```powershell
gripmock plugin build .\path\to\plugin --out myplugin.dll
gripmock --plugins=.\myplugin.dll service.proto
```

On Windows the package is built with `-buildmode=c-shared` and the C entry points
the server calls are added through a build overlay, so nothing is written into
the package. Both builds need a C toolchain (`CGO_ENABLED=1`); on Windows the
MSYS2 mingw64 gcc works.

### Matching the server (.so only)

`plugin.Open` compares the Go packages shared by the server and the plugin. They
match only when three things line up: the same Go minor version, the same
`-trimpath` setting, and the same paths the shared packages were compiled from.

For a release binary (homebrew, `setup.sh`, release archive) that means building
against the same module version, with `-trimpath`:

```bash
go mod init myplugin
go get github.com/bavix/gripmock/v3@v3.18.4   # the version gripmock --version reports
CGO_ENABLED=1 go build -trimpath -buildmode=plugin -o myplugin.so .
```

For the docker image the paths come from the image instead, so build in the
matching `:<tag>-builder` and point the module at the source it ships. See
[Builder Image](./builder-image.md).

A `.dll` has none of these constraints: it carries its own Go runtime, so the Go
version, `-trimpath` and module paths are free. What it gives up instead:

- arguments and results travel as JSON, so every number reaches the plugin as a
  `float64` and comes back as one
- functions that decorate a server function are not exported, because the DLL has
  no way to call back into the server

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
