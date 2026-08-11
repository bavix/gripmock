# Check <VersionTag version="v3.0.0" />

`gripmock check` verifies that a running GripMock server is healthy.

It uses gRPC health checks and waits until service `gripmock` becomes `SERVING`.

## Usage

```bash
gripmock check
```

## Options

| Flag | Short | Default | Description |
|---|---|---|---|
| `--timeout` | `-t` | `10s` | Total time to wait for readiness. |
| `--interval` | — | `500ms` | Delay between health check attempts. |
| `--silent` | — | `false` | Suppress error output from command. |

## Examples

Wait up to 30 seconds:

```bash
gripmock check --timeout 30s
```

Poll every 100ms instead of the default interval:

```bash
gripmock check --timeout 10s --interval 100ms
```

Use in CI script:

```bash
gripmock check --timeout 20s --silent
```

## Typical CI pattern

```bash
gripmock --stub ./stubs ./proto &
gripmock check --timeout 20s
go test ./...
```
