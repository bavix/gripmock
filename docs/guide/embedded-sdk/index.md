# GripMock Embedded SDK <VersionTag version="v3.7.0" />

::: info
**Minimum Requirements**: Go 1.26 or later
:::

> **Version history:** Embedded SDK introduced in <VersionTag version="v3.7.0" />. Current API since <VersionTag version="v3.16.0" />; the legacy `sdk.Run` / `mock.Stub` / `mock.Verify` API was **removed in v3.20.0**. See the [Upgrade Guide](./upgrade.md).

The Embedded SDK runs GripMock inside the Go test process — no container to start, no external binary to orchestrate.

## Real-World Example

Looking for a full project that uses embedded mode in practice?

- [bavix/greeter-gripmock-embedded](https://github.com/bavix/greeter-gripmock-embedded) - End-to-end example of GripMock Embedded SDK usage

## What it changes

- **No container startup per test.** A Docker-based mock costs roughly 0.4s to
  become ready ([benchmark](/guide/introduction/performance-comparison)); an
  in-process server costs a function call.
- **One instance per test.** Nothing is shared, so tests cannot pollute each
  other through stub state.
- **Stubs are Go code.** The compiler checks them, and a rename in the proto
  surfaces at build time rather than as a silent no-match.
- **Same process as the test.** A debugger steps from the assertion into the
  mock without crossing a process boundary.

Stubs cover the same ground as file stubs: unary and all three streaming
patterns, headers, errors, delays and priority. Call history and verification
are available in both embedded and remote mode.

## How It Works

The Embedded SDK creates a GripMock server instance directly within your test process. This server:
- Listens on a local port (typically a random available port)
- Responds to gRPC requests according to your stub definitions
- Maintains its own state for the duration of the test
- Automatically cleans up when the test completes

Start with [Installation](./installation.md), then
[Quick Start](./quick-start.md). The sidebar lists the rest.
