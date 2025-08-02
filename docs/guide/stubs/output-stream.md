# Output Stream Configuration

Output stream configuration defines how GripMock responds to gRPC requests, supporting various response types including data, errors, headers, and streaming.

## Overview

The `output` section in stub configuration controls:
- Response data structure
- Error conditions and codes
- HTTP/gRPC headers
- Streaming behavior
- Response timing (delays)

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
    - message: "First message"
      timestamp: "2024-01-01T12:00:00.000Z"
    - message: "Second message"
      timestamp: "2024-01-01T12:00:01.000Z"
    - message: "Third message"
      timestamp: "2024-01-01T12:00:02.000Z"
```

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
  delay: 500ms
  data:
    message: "Delayed response"
```

## Response Types

### 1. Data Response
```yaml
service: UserService
method: GetUser
input:
  equals:
    userId: 123
output:
  data:
    userId: 123
    name: "John Doe"
    email: "john@example.com"
    createdAt: "2024-01-01T12:00:00.000Z"
```

### 2. Error Response
```yaml
service: UserService
method: GetUser
input:
  equals:
    userId: 999
output:
  error: "User not found"
  code: 5  # NOT_FOUND
```

### 3. Streaming Response
```yaml
service: NotificationService
method: StreamNotifications
input:
  equals:
    userId: 123
output:
  delay: 100ms
  stream:
    - type: "INFO"
      message: "Welcome to the service"
      timestamp: "2024-01-01T12:00:00.000Z"
    - type: "WARNING"
      message: "Your session will expire soon"
      timestamp: "2024-01-01T12:00:01.000Z"
    - type: "SUCCESS"
      message: "Operation completed successfully"
      timestamp: "2024-01-01T12:00:02.000Z"
```

### 4. Response with Headers
```yaml
service: AuthService
method: Login
input:
  equals:
    username: "admin"
    password: "secret"
output:
  headers:
    "x-auth-token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
    "x-refresh-token": "refresh_token_123"
    "x-session-id": "session_456"
  data:
    userId: 123
    username: "admin"
    role: "administrator"
    lastLogin: "2024-01-01T12:00:00.000Z"
```

## Advanced Examples

### Conditional Responses
```yaml
# Success case
- service: PaymentService
  method: ProcessPayment
  input:
    equals:
      amount: 100
  output:
    data:
      transactionId: "txn-123"
      status: "SUCCESS"
      amount: 100
      timestamp: "2024-01-01T12:00:00.000Z"

# Insufficient funds
- service: PaymentService
  method: ProcessPayment
  input:
    equals:
      amount: 10000
  output:
    error: "Insufficient funds"
    code: 9  # FAILED_PRECONDITION
```

### Real-Time Data Feed
```yaml
service: SensorService
method: StreamReadings
input:
  equals:
    sensorId: "TEMP_001"
output:
  delay: 200ms
  headers:
    "x-sensor-id": "TEMP_001"
    "x-stream-type": "temperature"
  stream:
    - sensorId: "TEMP_001"
      temperature: 22.5
      humidity: 45.2
      pressure: 1013.25
      timestamp: "2024-01-01T10:00:00.000Z"
    - sensorId: "TEMP_001"
      temperature: 22.7
      humidity: 45.8
      pressure: 1013.30
      timestamp: "2024-01-01T10:00:02.000Z"
    - sensorId: "TEMP_001"
      temperature: 23.1
      humidity: 46.1
      pressure: 1013.35
      timestamp: "2024-01-01T10:00:04.000Z"
```

### Batch Processing
```yaml
service: ProcessingService
method: StreamProgress
input:
  equals:
    batchId: "BATCH_001"
output:
  delay: 1s
  headers:
    "x-batch-id": "BATCH_001"
    "x-total-items": "100"
  stream:
    - batchId: "BATCH_001"
      status: "STARTED"
      progress: 0
      processed: 0
      total: 100
      message: "Processing started"
    - batchId: "BATCH_001"
      status: "PROCESSING"
      progress: 25
      processed: 25
      total: 100
      message: "25% complete"
    - batchId: "BATCH_001"
      status: "PROCESSING"
      progress: 50
      processed: 50
      total: 100
      message: "50% complete"
    - batchId: "BATCH_001"
      status: "PROCESSING"
      progress: 75
      processed: 75
      total: 100
      message: "75% complete"
    - batchId: "BATCH_001"
      status: "COMPLETED"
      progress: 100
      processed: 100
      total: 100
      message: "Processing completed"
```

## Priority and Matching

### Priority Field
```yaml
output:
  priority: 100  # Higher priority (higher number = higher priority)
  data:
    message: "High priority response"
```

### Multiple Matching Stubs
When multiple stubs match a request, GripMock uses:
1. Priority (higher number = higher priority)
2. Order of definition (first defined wins)

## Error Codes

Common gRPC status codes:

| Code | Name | Description |
|------|------|-------------|
| 0 | OK | Success |
| 1 | CANCELLED | Request cancelled |
| 2 | UNKNOWN | Unknown error |
| 3 | INVALID_ARGUMENT | Invalid argument |
| 4 | DEADLINE_EXCEEDED | Deadline exceeded |
| 5 | NOT_FOUND | Resource not found |
| 6 | ALREADY_EXISTS | Resource already exists |
| 7 | PERMISSION_DENIED | Permission denied |
| 8 | RESOURCE_EXHAUSTED | Resource exhausted |
| 9 | FAILED_PRECONDITION | Failed precondition |
| 10 | ABORTED | Operation aborted |
| 11 | OUT_OF_RANGE | Out of range |
| 12 | UNIMPLEMENTED | Not implemented |
| 13 | INTERNAL | Internal error |
| 14 | UNAVAILABLE | Service unavailable |
| 15 | DATA_LOSS | Data loss |

## Best Practices

### 1. Response Structure
- Use consistent data structures
- Include meaningful field names
- Add timestamps for time-sensitive data

### 2. Error Handling
- Use appropriate error codes
- Provide descriptive error messages
- Test error scenarios

### 3. Headers
- Use headers for metadata
- Follow naming conventions (e.g., `x-` prefix)
- Keep headers relevant to the response

### 4. Streaming
- Use realistic delays
- Include proper message structure
- Test with various stream lengths

### 5. Testing
- Test all response types
- Verify error conditions
- Check header delivery
- Validate streaming behavior

## Limitations

- Stream responses terminate after sending all messages
- Maximum response size depends on gRPC limits
- Headers are set once per response/stream
- Delay affects the entire response timing 