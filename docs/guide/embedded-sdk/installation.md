# Installation <VersionTag version="v3.16.0" />

::: warning
⚠️ **EXPERIMENTAL FEATURE**: The GripMock Embedded SDK is currently experimental. The API is subject to change without notice, and functionality may be modified in future versions. Use at your own risk.
:::

::: info
**Minimum Requirements**: Go 1.26 or later
:::

> **Version history:** Embedded SDK introduced in <VersionTag version="v3.7.0" /> (legacy API: `sdk.Run`, `mock.Stub`). Current v2 API available since <VersionTag version="v3.16.0" />. See the [Upgrade Guide](./upgrade.md) for migration.

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
