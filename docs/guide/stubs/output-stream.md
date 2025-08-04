# Output Stream Configuration

Output stream configuration defines how GripMock responds to gRPC requests, supporting various response types including data, errors, headers, and streaming.

## Overview

The `output` section in stub configuration controls:
- Response data structure
- Error conditions and codes
- HTTP/gRPC headers
- Streaming behavior
- Response timing (delays)
- Bidirectional streaming responses

## Basic Output Structure

### Standard Response
```yaml
output:
  data:
    message: "Hello World"
    status: "success"
    timestamp: "2024-01-01T12:00:00.000Z"
```

### Error Response
```yaml
output:
  error: "Resource not found"
  code: 5  # NOT_FOUND
```

### Response with Headers
```yaml
output:
  headers:
    "x-request-id": "req-123"
    "x-cache-control": "no-cache"
  data:
    message: "Response with headers"
```

## Output Fields

### `data`
Contains the response payload for successful requests.

```yaml
output:
  data:
    userId: 12345
    name: "John Doe"
    email: "john@example.com"
    active: true
```

### `stream`
Defines server-side streaming responses (array of messages).

```yaml
output:
  stream:
    - data:
        message: "First message"
        timestamp: "2024-01-01T12:00:00.000Z"
    - data:
        message: "Second message"
        timestamp: "2024-01-01T12:00:01.000Z"
    - data:
        message: "Third message"
        timestamp: "2024-01-01T12:00:02.000Z"
```

**Note**: Each stream element should contain a `data` field for consistency with the V2 API.

### `error`
Error message for error responses.

```yaml
output:
  error: "User not found"
```

### `code`
gRPC status code for error responses.

```yaml
output:
  error: "Permission denied"
  code: 7  # PERMISSION_DENIED
```

### `headers`
HTTP/gRPC headers to include in the response.

```yaml
output:
  headers:
    "x-request-id": "req-123"
    "x-user-id": "user-456"
    "x-cache-control": "no-cache"
    "x-rate-limit-remaining": "100"
```

### `delay`
Artificial delay before sending the response.

```yaml
output:
  delay: 100ms
  data:
    message: "Delayed response"
```

## Streaming Response Types

### Server Streaming
For methods that return multiple responses over time:

```yaml
- service: DataService
  method: StreamData
  input:
    equals:
      request_id: "req_001"
      chunk_count: 5
  output:
    stream:
      - data:
          chunk_id: "chunk_001"
          sequence: 1
          content: "First chunk"
          timestamp: "2024-01-01T12:00:00.000Z"
      - data:
          chunk_id: "chunk_002"
          sequence: 2
          content: "Second chunk"
          timestamp: "2024-01-01T12:00:01.000Z"
      - data:
          chunk_id: "chunk_003"
          sequence: 3
          content: "Third chunk"
          timestamp: "2024-01-01T12:00:02.000Z"
```

### Bidirectional Streaming
For bidirectional streaming, responses are selected based on incoming messages:

```yaml
- service: ChatService
  method: Chat
  stream:
    - equals:
        user_id: "alice"
        content: "Hello"
        type: "MESSAGE_TYPE_TEXT"
    - equals:
        user_id: "alice"
        content: "How are you?"
        type: "MESSAGE_TYPE_TEXT"
  output:
    stream:
      - data:
          user_id: "bob"
          content: "Hello Alice!"
          type: "MESSAGE_TYPE_TEXT"
          timestamp: "2024-01-01T12:00:00.000Z"
      - data:
          user_id: "bob"
          content: "I'm doing great!"
          type: "MESSAGE_TYPE_TEXT"
          timestamp: "2024-01-01T12:00:01.000Z"
```

**Bidirectional Streaming Behavior:**
- Each incoming message is matched against the `stream` input patterns
- Responses are selected from the `output.stream` array based on message index
- If no exact match is found, the best matching stub is selected based on ranking
- Stub ranking considers exact matches, partial matches, and specificity

### Client Streaming Response
For client streaming methods, a single response is returned after all messages:

```yaml
- service: UploadService
  method: UploadFile
  stream:
    - equals:
        chunk_id: "chunk_001"
        sequence: 1
        total_chunks: 3
    - equals:
        chunk_id: "chunk_001"
        sequence: 2
        total_chunks: 3
    - equals:
        chunk_id: "chunk_001"
        sequence: 3
        total_chunks: 3
  output:
    data:
      upload_id: "upload_001"
      success: true
      total_chunks: 3
      total_size: "1500"
      status: "completed"
      completed_at: "2024-01-15T10:05:00Z"
```

## Advanced Output Features

### Conditional Responses
Use different outputs based on input conditions:

```yaml
- service: UserService
  method: GetUser
  input:
    equals:
      user_id: "12345"
  output:
    data:
      name: "John Doe"
      email: "john@example.com"
      status: "active"

- service: UserService
  method: GetUser
  input:
    equals:
      user_id: "99999"
  output:
    error: "User not found"
    code: 5  # NOT_FOUND
```

### Response with Metadata
Include additional metadata in responses:

```yaml
output:
  headers:
    "x-response-time": "150ms"
    "x-cache-hit": "true"
  data:
    result: "success"
    metadata:
      processed_at: "2024-01-01T12:00:00.000Z"
      version: "1.0.0"
      source: "mock"
```

### Error with Details
Provide detailed error information:

```yaml
output:
  error: "Validation failed"
  code: 3  # INVALID_ARGUMENT
  data:
    details:
      field: "email"
      reason: "Invalid email format"
      suggestion: "Use valid email format"
```

## API Version Compatibility

### V1 API (Legacy)
```yaml
- service: ChatService
  method: SendMessage
  input:
    equals:
      user: Alice
      text: "Hello"
  output:
    data:
      success: true
      message: "1 messages processed"
```

### V2 API (Streaming)
```yaml
- service: ChatService
  method: SendMessage
  stream:
    - equals:
        user: Alice
        text: "Hello"
  output:
    data:
      success: true
      message: "1 messages processed"
```

**Automatic Detection:**
- GripMock automatically detects V1 vs V2 format based on presence of `stream` field
- V1 stubs use `input` for matching
- V2 stubs use `stream` for matching
- Both formats are supported simultaneously for backward compatibility

## Performance Considerations

### Stub Ranking
GripMock uses sophisticated ranking algorithms to select the best matching stub:

1. **Exact Matches**: Highest priority for perfect matches
2. **Partial Matches**: Ranked based on field overlap
3. **Specificity**: More specific stubs rank higher
4. **Priority**: Explicit priority values override ranking
5. **Length Matching**: For streaming, length compatibility is considered

### Memory Efficiency
- Stubs are loaded once and cached in memory
- Streaming responses are generated on-demand
- Bidirectional streaming maintains minimal state per connection

## Best Practices

### 1. Use Consistent Data Structure
```yaml
# Good: Consistent structure
output:
  stream:
    - data:
        message: "First"
        timestamp: "2024-01-01T12:00:00.000Z"
    - data:
        message: "Second"
        timestamp: "2024-01-01T12:00:01.000Z"

# Avoid: Inconsistent structure
output:
  stream:
    - message: "First"
    - data:
        message: "Second"
```

### 2. Provide Meaningful Error Messages
```yaml
# Good: Descriptive error
output:
  error: "User with ID '12345' not found in database"
  code: 5

# Avoid: Generic error
output:
  error: "Not found"
  code: 5
```

### 3. Use Appropriate Delays
```yaml
# Good: Reasonable delay for testing
output:
  delay: 100ms
  data:
    message: "Response"

# Avoid: Excessive delays
output:
  delay: 30s
  data:
    message: "Response"
```

### 4. Leverage Priority for Complex Scenarios
```yaml
# High priority for specific cases
- service: UserService
  method: GetUser
  priority: 100
  input:
    equals:
      user_id: "12345"
      exact_match: true
  output:
    data:
      name: "John Doe"
      exact: true

# Lower priority fallback
- service: UserService
  method: GetUser
  priority: 50
  input:
    equals:
      user_id: "12345"
  output:
    data:
      name: "John Doe"
      fallback: true
``` 