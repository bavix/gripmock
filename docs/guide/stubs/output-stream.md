# Output Configuration <VersionTag version="v1.13.0" />

Reference for all output fields in stub configuration.

## Fields

| Field | Type | Description |
|---|---|---|
| `data` | object | Response payload for successful requests |
| `stream` | array | Server streaming messages |
| `error` | string | Error message |
| `code` | int | gRPC status code |
| `headers` | object | Response metadata |
| `delay` | duration | Response delay |
| `details` | array | Error details (gRPC status details) |

## `data` — Success Response <VersionTag version="v1.13.0" />

```yaml
output:
  data:
    message: "Hello World"
    status: "success"
```

## `stream` — Server Streaming <VersionTag version="v3.3.0" />

```yaml
output:
  stream:
    - message: "First"
    - message: "Second"
```

See [Streaming](./streaming) for full streaming guide.

## `error` + `code` — Error Response <VersionTag version="v2.0.0" />

```yaml
output:
  error: "Not found"
  code: 5  # NOT_FOUND
```

### gRPC Status Codes

| Code | Name | Description |
|---|---|---|
| 0 | OK | Success |
| 1 | CANCELED | Cancelled |
| 3 | INVALID_ARGUMENT | Invalid input |
| 4 | DEADLINE_EXCEEDED | Timeout |
| 5 | NOT_FOUND | Resource missing |
| 7 | PERMISSION_DENIED | Access denied |
| 8 | RESOURCE_EXHAUSTED | Quota exceeded |
| 13 | INTERNAL | Server error |
| 14 | UNAVAILABLE | Service unavailable |

## `headers` — Response Metadata <VersionTag version="v2.1.0" />

```yaml
output:
  headers:
    "x-request-id": "req-123"
    "x-cache-control": "no-cache"
  data:
    result: "ok"
```

## `delay` — Response Delay <VersionTag version="v3.2.16" />

```yaml
output:
  delay: 100ms
  data:
    message: "Delayed"
```

Formats: `100ms`, `1s`, `500ms`, `2.5s`

## `details` — Error Details <VersionTag version="v3.8.0" />

```yaml
output:
  error: "Validation failed"
  code: 3
  details:
    - type: "type.googleapis.com/google.rpc.ErrorInfo"
      reason: "API_DISABLED"
      domain: "example.service.local"
      metadata:
        service: "example.service.local"
```

Encoded as `google.protobuf.Any` in gRPC status details.

## Stream + Error

```yaml
output:
  stream:
    - message: "Starting"
    - message: "Done"
  error: "Completed with warnings"
  code: 2  # UNKNOWN
```

All stream messages are sent before the error terminates the RPC.

## Related

- [Streaming](./streaming) — streaming patterns
- [Delay](./delay) — response delays