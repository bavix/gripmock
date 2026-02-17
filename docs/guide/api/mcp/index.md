# MCP API <VersionTag version="v3.7.0" />

⚠️ **EXPERIMENTAL FEATURE**: The MCP API is currently experimental and primarily designed for AI/agent integrations. The API is subject to change without notice and may be modified or removed in future versions. Plan your integrations for breakage tolerance and version drift.

GripMock supports MCP over HTTP JSON-RPC at `POST /api/mcp`.

MCP gives AI agents a single integration point to inspect services, manage dynamic descriptors, and run debug/history workflows.

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
