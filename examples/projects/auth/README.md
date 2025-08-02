# Auth Service

Simple authentication and authorization example using gRPC and GripMock.

## What it does

- Validates API keys through headers
- Checks resource access permissions
- Shows how to test security scenarios

## Run

```bash
gripmock --stub examples/projects/auth examples/projects/auth/service.proto
```

## Tests

```bash
grpctestify examples/projects/auth/
```

## Structure

- `service.proto` - gRPC service definition
- `stubs.yml` - mock responses for testing
- `*.gctf` - test scenarios

## Features

- **API Key Auth**: Validation via `x-api-key` header
- **RBAC**: Resource-action permission model
- **Fallback**: General rules for unknown requests 