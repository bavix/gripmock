---
title: Builder Image
---

# Builder Image <VersionTag version="v3.7.1" />

## Main rule

- build plugins in `bavix/gripmock:<tag>-builder`
- run GripMock in `bavix/gripmock:<tag>`
- always keep the same `<tag>`

This is required because Go plugins are sensitive to toolchain and platform differences (`plugin.Open`).

## Published images

For each release tag `<tag>`:

- `bavix/gripmock:<tag>` - runtime, cgo, plugins work
- `bavix/gripmock:<tag>-builder` - toolchain for compiling plugins
- `bavix/gripmock:<tag>-slim` - runtime without cgo, **no plugin support**

Same tags are published to `ghcr.io/bavix/gripmock`.

Pick `-slim` only if you never pass `--plugins`; `plugin.Open` there fails with
`plugin: not implemented`.

## CI behavior

- `Dockerfile.builder` builds and publishes `:<tag>-builder`
- `Dockerfile` builds and publishes `:<tag>` (`cgo=1`) and `:<tag>-slim` (`cgo=0`)
- runtime builds pass `BUILDER_IMAGE=ghcr.io/bavix/gripmock@<builder digest>` from the same pipeline run
- layers are pushed with zstd compression; Docker 23+ or containerd is required to pull them

This keeps runtime and builder strictly aligned.

## Usage

The builder image ships the runtime sources at `/gripmock-src`. Point your module
at them: `plugin.Open` matches packages by the path they were compiled from, and
`/gripmock-src` is where the runtime image compiled them too.

```bash
docker run --rm \
  -v "$PWD":/work \
  -w /work \
  bavix/gripmock:3.18.4-builder \
  sh -c 'go mod edit -replace github.com/bavix/gripmock/v3=/gripmock-src \
    && go mod tidy \
    && go build -buildmode=plugin -o ./plugins/myplugin.so ./cmd/myplugin'
```

Use `sh -c`, not `sh -lc`: a login shell resets `PATH` and `go` is not on it.
Drop the replace afterwards with `go mod edit -dropreplace github.com/bavix/gripmock/v3`.

```bash
docker run --rm \
  -p 4770:4770 -p 4771:4771 \
  -v "$PWD/plugins":/plugins \
  -v "$PWD/proto":/proto \
  bavix/gripmock:3.18.4 \
  --plugins=/plugins/myplugin.so /proto/service.proto
```

## If plugin does not load

- verify the runtime is not the `-slim` image
- verify runtime and builder use the same base tag
- verify the module points at `/gripmock-src` (`go mod edit -replace ...`)
- do not pass `-trimpath`: the runtime image is built without it, and the flag
  has to match on both sides
- rebuild plugin in matching `:<tag>-builder`
- verify architecture (`amd64`/`arm64`)
