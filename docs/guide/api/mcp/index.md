# MCP API <VersionTag version="v3.7.0" />

⚠️ **EXPERIMENTAL FEATURE**: The MCP API is currently experimental and primarily designed for AI/agent integrations. The API is subject to change without notice and may be modified or removed in future versions. Plan your integrations for breakage tolerance and version drift.

GripMock supports MCP over HTTP JSON-RPC at `POST /api/mcp`.

MCP gives AI agents a single integration point to inspect services, manage dynamic descriptors, and run debug/history workflows.

## Available tools

Use `tools/list` to discover the exact runtime set.

- `descriptors.add`, `descriptors.list`
- `services.list`, `services.delete`
- `stubs.upsert`, `stubs.list`, `stubs.get`, `stubs.delete`, `stubs.batchDelete`, `stubs.purge`, `stubs.search`, `stubs.used`, `stubs.unused`
- `schema.stub`
- `history.list`, `history.errors`
- `debug.call`

Minimal end-to-end MCP flow:

1. `tools/call` -> `descriptors.add`
2. `tools/call` -> `stubs.upsert`
3. external `grpcurl` call to `localhost:4770`
4. `tools/call` -> `history.list` or `stubs.used`

## JSON-RPC examples

Initialize session:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize"
}
```

List available tools:

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list"
}
```

`stubs.upsert`:

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

`stubs.list`:

```json
{
  "jsonrpc": "2.0",
  "id": 11,
  "method": "tools/call",
  "params": {
    "name": "stubs.list",
    "arguments": {
      "service": "unitconverter.v1.UnitConversionService",
      "method": "ConvertWeight",
      "limit": 10
    }
  }
}
```

`stubs.get`:

```json
{
  "jsonrpc": "2.0",
  "id": 12,
  "method": "tools/call",
  "params": {
    "name": "stubs.get",
    "arguments": {
      "id": "fc800277-9bbb-4e0b-988e-4cf01b525085"
    }
  }
}
```

`stubs.delete`:

```json
{
  "jsonrpc": "2.0",
  "id": 13,
  "method": "tools/call",
  "params": {
    "name": "stubs.delete",
    "arguments": {
      "id": "fc800277-9bbb-4e0b-988e-4cf01b525085"
    }
  }
}
```

`stubs.batchDelete`:

```json
{
  "jsonrpc": "2.0",
  "id": 14,
  "method": "tools/call",
  "params": {
    "name": "stubs.batchDelete",
    "arguments": {
      "ids": [
        "fc800277-9bbb-4e0b-988e-4cf01b525085",
        "a6d58d6c-43ce-4c6e-8b2a-9f9a9ed6c8e1"
      ]
    }
  }
}
```

`stubs.purge`:

```json
{
  "jsonrpc": "2.0",
  "id": 15,
  "method": "tools/call",
  "params": {
    "name": "stubs.purge",
    "arguments": {}
  }
}
```

`stubs.search`:

```json
{
  "jsonrpc": "2.0",
  "id": 16,
  "method": "tools/call",
  "params": {
    "name": "stubs.search",
    "arguments": {
      "service": "unitconverter.v1.UnitConversionService",
      "method": "ConvertWeight",
      "payload": {
        "value": 1,
        "from_unit": "POUNDS",
        "to_unit": "KILOGRAMS"
      }
    }
  }
}
```

`stubs.used`:

```json
{
  "jsonrpc": "2.0",
  "id": 17,
  "method": "tools/call",
  "params": {
    "name": "stubs.used",
    "arguments": {
      "service": "unitconverter.v1.UnitConversionService"
    }
  }
}
```

`stubs.unused`:

```json
{
  "jsonrpc": "2.0",
  "id": 18,
  "method": "tools/call",
  "params": {
    "name": "stubs.unused",
    "arguments": {
      "service": "unitconverter.v1.UnitConversionService"
    }
  }
}
```

`history.list` (post-call verification):

```json
{
  "jsonrpc": "2.0",
  "id": 19,
  "method": "tools/call",
  "params": {
    "name": "history.list",
    "arguments": {
      "service": "unitconverter.v1.UnitConversionService",
      "method": "ConvertWeight",
      "limit": 5
    }
  }
}
```

`schema.stub` (stub schema URL discovery):

```json
{
  "jsonrpc": "2.0",
  "id": 20,
  "method": "tools/call",
  "params": {
    "name": "schema.stub",
    "arguments": {}
  }
}
```

This page focuses on connecting GripMock MCP to OpenCode.

## Endpoints

- `GET /api/mcp` — MCP transport metadata.
- `POST /api/mcp` — JSON-RPC request handler.

## Endpoint shape

MCP for GripMock is exposed as a regular HTTP endpoint:

```text
POST http://127.0.0.1:4771/api/mcp
Content-Type: application/json
X-Gripmock-Session: <optional-session>
```

## Session support

Session source priority:

1. explicit `arguments.session`
2. `X-Gripmock-Session` header

The transport-level session is applied when `arguments.session` is omitted.

## Client setup examples

### OpenCode (config file example)

In OpenCode, add GripMock MCP server to `~/.config/opencode/config.json`:

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

For session-scoped behavior, send this header:

```text
X-Gripmock-Session: qa-run-42
```

Use the same MCP endpoint URL: `http://localhost:4771/api/mcp`.
