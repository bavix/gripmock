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
- `bavix/gripmock:<tag>-debian` - Debian runtime, cgo, plugins work
- `bavix/gripmock:<tag>-debian-builder` - Debian toolchain for compiling plugins
- `bavix/gripmock:<tag>-debian-slim` - Debian runtime without cgo, **no plugin support**
- `bavix/gripmock:<tag>-nonroot` - Alpine runtime as UID/GID `65532:65532`
- `bavix/gripmock:<tag>-slim-nonroot` - Alpine slim runtime as UID/GID `65532:65532`
- `bavix/gripmock:<tag>-debian-nonroot` - Debian runtime as UID/GID `65532:65532`
- `bavix/gripmock:<tag>-debian-slim-nonroot` - Debian slim runtime as UID/GID `65532:65532`
- append `-legacy` for gzip-compressed variants, for example `:<tag>-nonroot-legacy`

Same tags are published to `ghcr.io/bavix/gripmock`.

Pick `-slim` only if you never pass `--plugins`; `plugin.Open` there fails with
`plugin: not implemented`.

Nonroot images run as UID/GID `65532:65532` and can read mounted proto and stub
files when that user has read permission. Commands that write files (for example
`gripmock dump`) need a writable output mount accessible to that UID/GID. The tag
describes the image's configured user; it does not require a rootless container
engine.

## CI behavior

- `.github/workflows/builder.yaml` owns the build matrix, credentials, digest artifacts, and release trigger
- `.github/actions/build-image-set` builds a compatible builder/runtime set and derives nonroot wrappers from the pushed base digests
- `.github/actions/publish-image-manifest` generates the established version tags and merges per-architecture digests
- `Dockerfile.builder` and `Dockerfile.builder.debian` build and publish the Alpine and Debian builder images
- `Dockerfile` and `Dockerfile.debian` build the corresponding runtime (`cgo=1`) and slim (`cgo=0`) images
- `Dockerfile.nonroot` derives nonroot tags from the matching runtime digest; it changes only the configured UID/GID and does not rebuild the binary
- runtime builds pass `BUILDER_IMAGE=ghcr.io/bavix/gripmock@<builder digest>` from the same pipeline run
- Build plugins with the builder image matching the runtime distro.
- nonroot and slim-nonroot tags are published for both zstd and gzip (`-legacy`) variants
- normal tags use zstd-compressed layers; Docker 23+ or containerd is required to pull them
- `-legacy` tags use gzip-compressed layers for older runtimes
- only the default Alpine zstd runtime publishes `latest`; Debian, builder, slim, and legacy tags do not compete for it

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
