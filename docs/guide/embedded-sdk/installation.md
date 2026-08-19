# Installation <VersionTag version="v3.16.0" />

::: info
**Minimum Requirements**: Go 1.26 or later
:::

> **Version history:** Embedded SDK introduced in <VersionTag version="v3.7.0" />. Current API since <VersionTag version="v3.16.0" />; the legacy `sdk.Run` / `mock.Stub` / `mock.Verify` API was **removed in v3.20.0**. See the [Upgrade Guide](./upgrade.md).

Add GripMock SDK to your Go project:

```bash
go get github.com/bavix/gripmock/v3/pkg/sdk
```

## Prerequisites

- Go 1.26 or later
- Protocol Buffer files (.proto) for your gRPC services

## Import

Import the SDK in your Go code:

```go
import sdk "github.com/bavix/gripmock/v3/pkg/sdk"
```
