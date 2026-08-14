# Streaming <VersionTag version="v3.3.0" />

GripMock supports all three gRPC streaming patterns.

## Types

| Type | Direction | Use Case |
|---|---|---|
| Server Streaming | Server → Client | Real-time feeds, progress updates |
| Client Streaming | Client → Server | File uploads, batch data |
| Bidirectional Streaming | Both ways | Chat, real-time collaboration |

## Server Streaming

Server sends multiple messages in response to a single request.

```yaml
service: TrackService
method: StreamTrack
input:
  equals:
    stn: "MS#00001"
output:
  stream:
    - stn: "MS#00001"
      latitude: 0.1
      longitude: 0.005
      speed: 45
    - stn: "MS#00001"
      latitude: 0.10001
      longitude: 0.00501
      speed: 46
```

### Delay Between Messages

```yaml
output:
  delay: 200ms
  stream:
    - message: "First"   # sent after 200ms
    - message: "Second"  # sent after another 200ms
    - message: "Third"   # sent after another 200ms
```

The delay applies before every message, including the first. Three messages at
200ms take 600ms.

## Client Streaming

Client sends multiple messages, server responds once.

```yaml
service: UploadService
method: UploadFile
inputs:
  - equals:
      chunk_id: "file_001"
      sequence: 1
      total_chunks: 3
  - equals:
      chunk_id: "file_001"
      sequence: 2
      total_chunks: 3
  - equals:
      chunk_id: "file_001"
      sequence: 3
      total_chunks: 3
output:
  data:
    upload_id: "upload_001"
    success: true
```

Use `inputs` (plural) for V2 API. The old `input` (singular) still works.

## Bidirectional Streaming

Both sides send messages independently. Responses are matched to incoming messages.

```yaml
service: ChatService
method: Chat
inputs:
  - equals:
      user_id: "alice"
      content: "Hello"
  - equals:
      user_id: "alice"
      content: "How are you?"
output:
  stream:
    - user_id: "bob"
      content: "Hello Alice!"
    - user_id: "bob"
      content: "I'm doing great!"
```

Each incoming message is matched against `inputs` patterns in order. Response index matches input index.

### Fallback Matching

If an incoming message doesn't match any `inputs` pattern, GripMock uses ranking to find the best match:

1. **Exact match** → highest rank
2. **Field overlap** → more matching fields = higher rank
3. **Specificity** → more specific stubs rank higher

## V1 vs V2 API

| Feature | V1 | V2 |
|---|---|---|
| Input matching | `input` (single) | `inputs` (array) |
| Server stream | `output.data` (array) | `output.stream` |
| Bidirectional | Not supported | `inputs` + `output.stream` |
| Client streaming | Not supported | `inputs` + `output.data` |

### V1 Example (Still Works)
```yaml
service: ChatService
method: SendMessage
input:
  equals:
    user: "alice"
output:
  data:
    success: true
```

### V2 Example (Recommended)
```yaml
service: ChatService
method: SendMessage
inputs:  # V2 indicator
  - equals:
      user: "alice"
output:
  data:
    success: true
```

GripMock picks V1 or V2 by which input field the stub carries: `input` or `inputs`.

## Error Handling

```yaml
output:
  error: "Something went wrong"
  code: 5  # NOT_FOUND
```

### Stream + Error

When both `stream` and `error` are specified:

- Non-empty stream → all messages sent, then error
- Empty stream → error immediately

This works for stubs loaded from a file. `POST /api/stubs` rejects the same stub
with `400`: over REST, `output` must carry either the unary side (`data`,
`error`, `code`, `details`) or `stream`, not both.

```yaml
output:
  stream:
    - message: "Processing..."
    - message: "Almost done"
  error: "Insufficient resources"
  code: 8  # RESOURCE_EXHAUSTED
```

## Related

- [Output Configuration](./output-stream) — all output fields
- [Delay](./delay) — response delays
- [Matching Logic](../matcher/logic) — input matching rules