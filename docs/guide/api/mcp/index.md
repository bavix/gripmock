# MCP API <VersionTag version="v3.7.0" />

⚠️ **EXPERIMENTAL FEATURE**: The MCP API is experimental and may change without notice.

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

- health: `health_liveness`, `health_readiness`, `health_status`
- dashboard: `dashboard_full`, `dashboard_overview`, `dashboard_info`
- sessions: `sessions_list`
- gripmock: `gripmock_info`
- reflection: `reflect_info`, `reflect_sources`
- descriptors: `descriptors_add`, `descriptors_list`
- services: `services_list`, `services_get`, `services_methods`, `services_method`, `services_delete`
- history/verify/debug: `history_list`, `history_errors`, `verify_calls`, `debug_call`
- stubs: `stubs_upsert`, `stubs_list`, `stubs_get`, `stubs_delete`, `stubs_batch_delete`, `stubs_purge`, `stubs_search`, `stubs_inspect`, `stubs_used`, `stubs_unused`
- schema: `schema_stub`

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

Call tool (`stubs_upsert`):

```json
{
  "jsonrpc": "2.0",
  "id": 10,
  "method": "tools/call",
  "params": {
    "name": "stubs_upsert",
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

Call tool (`stubs_inspect`):

```json
{
  "jsonrpc": "2.0",
  "id": 11,
  "method": "tools/call",
  "params": {
    "name": "stubs_inspect",
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

Call tool (`reflect_sources`) with filtering/pagination:

```json
{
  "jsonrpc": "2.0",
  "id": 12,
  "method": "tools/call",
  "params": {
    "name": "reflect_sources",
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
