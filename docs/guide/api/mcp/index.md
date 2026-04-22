# MCP API <VersionTag version="v3.7.0" />

GripMock exposes MCP over HTTP at `POST /api/mcp` using `github.com/modelcontextprotocol/go-sdk`.

## Protocol

- MCP protocol version: `2025-11-25`
- Transport: Streamable HTTP (stateless JSON mode)
- Endpoint: `http://127.0.0.1:4771/api/mcp`

## Session behavior

Session source priority:

1. explicit `arguments.session`
2. `X-Gripmock-Session` request header

The header session is injected by middleware into MCP tool execution context.

## Available tools

Use `tools/list` to discover runtime tool metadata. Current tool surface:

- health: `health.liveness`, `health.readiness`, `health.status`
- dashboard: `dashboard.full`, `dashboard.overview`, `dashboard.info`
- sessions: `sessions.list`
- gripmock: `gripmock.info`
- reflection: `reflect.info`, `reflect.sources`
- descriptors: `descriptors.add`, `descriptors.list`
- services: `services.list`, `services.get`, `services.methods`, `services.method`, `services.delete`
- history/verify/debug: `history.list`, `history.errors`, `verify.calls`, `debug.call`
- stubs: `stubs.upsert`, `stubs.list`, `stubs.get`, `stubs.delete`, `stubs.batchDelete`, `stubs.purge`, `stubs.search`, `stubs.inspect`, `stubs.used`, `stubs.unused`
- schema: `schema.stub`

## JSON-RPC examples

Initialize:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-11-25",
    "capabilities": {},
    "clientInfo": {
      "name": "example-client",
      "version": "1.0.0"
    }
  }
}
```

List tools:

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list",
  "params": {}
}
```

Call tool (`stubs.upsert`):

```json
{
  "jsonrpc": "2.0",
  "id": 10,
  "method": "tools/call",
  "params": {
    "name": "stubs.upsert",
    "arguments": {
      "stubs": {
        "service": "unitconverter.v1.UnitConversionService",
        "method": "ConvertWeight",
        "input": {
          "equals": {
            "value": 1,
            "from_unit": "POUNDS",
            "to_unit": "KILOGRAMS"
          }
        },
        "output": {
          "data": {
            "converted_value": 0.453592
          }
        }
      }
    }
  }
}
```

Call tool (`stubs.inspect`):

```json
{
  "jsonrpc": "2.0",
  "id": 11,
  "method": "tools/call",
  "params": {
    "name": "stubs.inspect",
    "arguments": {
      "service": "unitconverter.v1.UnitConversionService",
      "method": "ConvertWeight",
      "input": [
        {
          "value": 1,
          "from_unit": "POUNDS",
          "to_unit": "KILOGRAMS"
        }
      ]
    }
  }
}
```

Call tool (`reflect.sources`) with filtering/pagination:

```json
{
  "jsonrpc": "2.0",
  "id": 12,
  "method": "tools/call",
  "params": {
    "name": "reflect.sources",
    "arguments": {
      "kind": "dynamic",
      "offset": 0,
      "limit": 50
    }
  }
}
```

Notification (`notifications/initialized`):

```json
{
  "jsonrpc": "2.0",
  "method": "notifications/initialized",
  "params": {}
}
```

## Client setup example (OpenCode)

```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "gripmock": {
      "type": "remote",
      "url": "http://localhost:4771/api/mcp",
      "enabled": true
    }
  }
}
```

Optional request header for session-scoped calls:

```text
X-Gripmock-Session: qa-run-42
```
