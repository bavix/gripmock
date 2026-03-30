# GitHub Actions (CI/CD)

GripMock has an official GitHub Action: [`bavix/gripmock-action`](https://github.com/bavix/gripmock-action).

It is designed for CI/CD workflows and handles the full lifecycle for you:

- Downloads GripMock from GitHub Releases (`latest` or pinned version)
- Starts GripMock in background
- Waits for readiness (`/api/health/readiness`)
- Exposes connection outputs for test steps
- Stops GripMock automatically in the post step

## Quick start

```yaml
name: test

on: [push, pull_request]

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5

      - name: Start GripMock
        uses: bavix/gripmock-action@v1
        with:
          source: proto/service.proto
          stub: stubs

      - name: Run tests
        run: go test ./...
```

## Common CI/CD scenarios

### Pinned version for reproducible pipelines

```yaml
- name: Start GripMock (pinned)
  uses: bavix/gripmock-action@v1
  with:
    version: v3.9.0
    source: proto/service.proto
    stub: stubs
```

### Buf Schema Registry (BSR)

```yaml
- name: Start GripMock from BSR
  uses: bavix/gripmock-action@v1
  with:
    source: buf.build/connectrpc/eliza
    stub: stubs
    env: |
      BSR_BUF_TOKEN=${{ secrets.BSR_BUF_TOKEN }}
```

### Reflection with upstream replay mode

```yaml
- name: Start replay mode
  uses: bavix/gripmock-action@v1
  with:
    source: grpc+replay://localhost:50051
    wait-timeout: 60s
```

## Useful outputs

The action provides outputs you can pass into test steps:

- `grpc-addr` - `<grpc-host>:<grpc-port>`
- `http-addr` - `<http-host>:<http-port>`
- `readiness-url` - health endpoint used by readiness waiter
- `log-file` - path to GripMock logs on the runner

Example:

```yaml
- name: Start GripMock
  id: gripmock
  uses: bavix/gripmock-action@v1
  with:
    source: proto/service.proto
    stub: stubs

- name: Run integration tests
  env:
    GRIPMOCK_GRPC_ADDR: ${{ steps.gripmock.outputs.grpc-addr }}
    GRIPMOCK_HTTP_ADDR: ${{ steps.gripmock.outputs.http-addr }}
  run: go test ./... -v
```

## Recommendations

- Pin `version` in CI for reproducible runs
- Keep `wait: true` (default) to avoid race conditions in tests
- Use `log-file` output for debugging failed startup in CI logs

Full action API (inputs/outputs):

- [`bavix/gripmock-action` README](https://github.com/bavix/gripmock-action)
