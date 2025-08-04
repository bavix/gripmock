# Delay Configuration

Delay configuration allows you to simulate network latency and processing time in your gRPC mock responses, making your tests more realistic.

## Overview

The `delay` field in stub output configuration lets you add artificial delays before sending responses. This is useful for:
- Testing timeout scenarios
- Simulating slow network conditions
- Validating client-side retry logic
- Performance testing under realistic conditions

## Basic Usage

### JSON Format
```json
{
  "service": "TrackService",
  "method": "StreamTrack",
  "input": {
    "equals": {
      "stn": "MS#00001"
    }
  },
  "output": {
    "delay": "100ms",
    "data": {
      "stn": "MS#00001",
      "identity": "00",
      "latitude": 0.1,
      "longitude": 0.005,
      "speed": 45,
      "updatedAt": "2024-01-01T12:00:00.000Z"
    }
  }
}
```

### YAML Format
```yaml
service: TrackService
method: StreamTrack
input:
  equals:
    stn: "MS#00001"
output:
  delay: 100ms
  data:
    stn: "MS#00001"
    identity: "00"
    latitude: 0.1
    longitude: 0.005
    speed: 45
    updatedAt: "2024-01-01T12:00:00.000Z"
```

## Delay Formats

GripMock supports various time duration formats:

### Supported Units
- **Milliseconds**: `100ms`, `500ms`, `1.5ms`
- **Seconds**: `1s`, `2.5s`, `30s`
- **Minutes**: `1m`, `5m`
- **Hours**: `1h`, `2h`

### Examples
```yaml
output:
  delay: 100ms    # 100 milliseconds
  delay: 2.5s     # 2.5 seconds
  delay: 1m       # 1 minute
  delay: 1h30m    # 1 hour 30 minutes
```

## Streaming Responses

For streaming responses, delay is applied **between** each message in the stream (not before each message):

### Server-Side Streaming
```yaml
service: TrackService
method: StreamTrack
input:
  equals:
    stn: "MS#00005"
output:
  delay: 200ms
  stream:
    - stn: "MS#00005"
      identity: "00"
      latitude: 0.11
      longitude: 0.006
      speed: 50
      updatedAt: "2024-01-01T13:00:00.000Z"
    - stn: "MS#00005"
      identity: "01"
      latitude: 0.11001
      longitude: 0.00601
      speed: 51
      updatedAt: "2024-01-01T13:00:01.000Z"
    - stn: "MS#00005"
      identity: "02"
      latitude: 0.11002
      longitude: 0.00602
      speed: 52
      updatedAt: "2024-01-01T13:00:02.000Z"
```

In this example, there will be a 200ms delay **between** each message in the stream. The timing will be:
- Message 1 sent immediately
- 200ms delay
- Message 2 sent
- 200ms delay  
- Message 3 sent

## Use Cases

### 1. Timeout Testing
```yaml
output:
  delay: 5s
  data:
    message: "Slow response"
```
Use long delays to test client timeout handling.

### 2. Realistic Network Simulation
```yaml
output:
  delay: 150ms
  data:
    message: "Typical network latency"
```
Simulate realistic network conditions for performance testing.

### 3. Rate Limiting Simulation
```yaml
output:
  delay: 1s
  data:
    message: "Rate limited response"
```
Test client behavior under rate limiting scenarios.

## Error Responses with Delay

Delay can be combined with error responses:

```yaml
output:
  delay: 500ms
  error: "Service temporarily unavailable"
  code: 14  # UNAVAILABLE
```

## Best Practices

### 1. Realistic Delays
- Use delays that reflect real-world conditions
- Consider your application's timeout settings
- Test with various delay values

### 2. Testing Strategy
- Test with zero delay for fast responses
- Test with moderate delays (100-500ms) for normal conditions
- Test with long delays (1s+) for timeout scenarios

### 3. Performance Considerations
- Avoid extremely long delays in automated tests
- Use appropriate delays for your testing environment
- Consider the impact on test execution time

## Delay Logic

### Unary Calls
- Delay is applied **before** sending the single response
- Total delay = configured delay value

### Streaming Calls  
- Delay is applied **between** each message in the stream
- First message is sent immediately
- Subsequent messages are sent after the configured delay
- Total delay = (number of messages - 1) × configured delay

### Examples

#### Unary Response (1 response)
```yaml
output:
  delay: 200ms
  data:
    message: "Hello"
```
**Timing**: 200ms delay → response sent

#### Streaming Response (3 messages)
```yaml
output:
  delay: 200ms
  stream:
    - message: "First"
    - message: "Second" 
    - message: "Third"
```
**Timing**:
- Message 1 sent immediately
- 200ms delay
- Message 2 sent
- 200ms delay
- Message 3 sent

**Total delay**: 400ms (2 × 200ms)

## Limitations

- Maximum delay is limited by client timeout settings
- Delay affects all response types (data, error, stream)
- Delay is applied consistently across all gRPC call types

## Verification

You can verify delay behavior using gRPC clients or tools like `grpcurl`:

```bash
# Test with delay
grpcurl -plaintext -d '{"stn":"MS#00001"}' localhost:4770 TrackService/StreamTrack
```

The response time should include the configured delay plus processing time. 