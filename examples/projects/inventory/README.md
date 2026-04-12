# Inventory Service

Stateful stub example demonstrating sequential responses and retry scenario testing.

## What it does

- First 2 calls return `UNAVAILABLE` error (simulating transient failure)
- Third call onwards returns successful response
- Uses **priority-based stub matching** with `times` option to test retry logic
- Tests multiple consecutive requests (N requests) with `grpctestify`

## Run

```bash
gripmock --stub examples/projects/inventory examples/projects/inventory/service.proto
```

## Tests

```bash
grpctestify examples/projects/inventory/
```

## Structure

- `service.proto` - gRPC service definition
- `stubs.yaml` - stateful stubs with priority & times options
- `multi.gctf` - test with N sequential requests

## Features

- **Stateful Matching**: Stubs change behavior after being exhausted
- **Priority System**: High priority stub (priority: 10) used first, falls back to low priority (priority: 1)
- **Times Option**: Limits stub usage count (`times: 2`)
- **Retry Testing**: Validates client retry logic for transient failures
