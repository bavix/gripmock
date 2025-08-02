# Server-Side Streaming

Server-side streaming allows you to send multiple messages in a single gRPC call, simulating real-time data feeds, batch processing, or progressive data delivery.

## Overview

Server-side streaming is useful for:
- Real-time data feeds (GPS tracking, sensor data)
- Large dataset delivery in chunks
- Progressive data processing
- Event streaming and notifications
- Batch operations with progress updates

## Basic Configuration

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
    "stream": [
      {
        "stn": "MS#00001",
        "identity": "00",
        "latitude": 0.1,
        "longitude": 0.005,
        "speed": 45,
        "updatedAt": "2024-01-01T12:00:00.000Z"
      },
      {
        "stn": "MS#00001",
        "identity": "01",
        "latitude": 0.10001,
        "longitude": 0.00501,
        "speed": 46,
        "updatedAt": "2024-01-01T12:00:01.000Z"
      }
    ]
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
  stream:
    - stn: "MS#00001"
      identity: "00"
      latitude: 0.1
      longitude: 0.005
      speed: 45
      updatedAt: "2024-01-01T12:00:00.000Z"
    - stn: "MS#00001"
      identity: "01"
      latitude: 0.10001
      longitude: 0.00501
      speed: 46
      updatedAt: "2024-01-01T12:00:01.000Z"
```

## Stream Behavior

### Message Delivery
- Messages are sent sequentially in the order defined in the array
- Each message is sent as a separate gRPC message
- The stream terminates after sending all messages
- No infinite loops - streams complete naturally

### Delay Between Messages
```yaml
output:
  delay: 200ms  # Delay between each message
  stream:
    - message: "First message (sent immediately)"
    - message: "Second message (sent after 200ms delay)"
    - message: "Third message (sent after another 200ms delay)"
```

**Timing for 3 messages with 200ms delay:**
- Message 1: sent immediately
- 200ms delay
- Message 2: sent
- 200ms delay  
- Message 3: sent

**Total delay**: 400ms (2 × 200ms)

## Advanced Examples

### Real-Time GPS Tracking
```yaml
service: TrackService
method: StreamTrack
input:
  equals:
    stn: "MS#00005"
output:
  delay: 100ms
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
    - stn: "MS#00005"
      identity: "03"
      latitude: 0.11003
      longitude: 0.00603
      speed: 53
      updatedAt: "2024-01-01T13:00:03.000Z"
```

### Sensor Data Feed
```yaml
service: SensorService
method: StreamReadings
input:
  equals:
    sensorId: "TEMP_001"
output:
  delay: 500ms
  stream:
    - sensorId: "TEMP_001"
      temperature: 22.5
      humidity: 45.2
      timestamp: "2024-01-01T10:00:00.000Z"
    - sensorId: "TEMP_001"
      temperature: 22.7
      humidity: 45.8
      timestamp: "2024-01-01T10:00:05.000Z"
    - sensorId: "TEMP_001"
      temperature: 23.1
      humidity: 46.1
      timestamp: "2024-01-01T10:00:10.000Z"
```

### Batch Processing with Progress
```yaml
service: ProcessingService
method: StreamProgress
input:
  equals:
    batchId: "BATCH_001"
output:
  delay: 1s
  stream:
    - batchId: "BATCH_001"
      status: "STARTED"
      progress: 0
      message: "Processing started"
    - batchId: "BATCH_001"
      status: "PROCESSING"
      progress: 25
      message: "25% complete"
    - batchId: "BATCH_001"
      status: "PROCESSING"
      progress: 50
      message: "50% complete"
    - batchId: "BATCH_001"
      status: "PROCESSING"
      progress: 75
      message: "75% complete"
    - batchId: "BATCH_001"
      status: "COMPLETED"
      progress: 100
      message: "Processing completed"
```

## Headers in Streaming

You can set headers for streaming responses:

```yaml
output:
  headers:
    "x-stream-id": "stream-123"
    "x-message-count": "3"
  stream:
    - message: "First message"
    - message: "Second message"
    - message: "Third message"
```

## Error Handling

### Stream with Error
```yaml
output:
  stream:
    - message: "First message"
    - message: "Second message"
  error: "Stream interrupted"
  code: 13  # INTERNAL
```

### Early Termination
If an error occurs during streaming, the stream terminates immediately.

## Testing Server Streaming

### Using grpcurl
```bash
# Test server streaming
grpcurl -plaintext -d '{"stn":"MS#00001"}' localhost:4770 TrackService/StreamTrack
```

### Using gRPC Testify
```gctf
--- ENDPOINT ---
TrackService/StreamTrack

--- REQUEST ---
{
  "stn": "MS#00001"
}

--- RESPONSE ---
{
  "stn": "MS#00001",
  "identity": "00",
  "latitude": 0.1,
  "longitude": 0.005,
  "speed": 45,
  "updatedAt": "2024-01-01T12:00:00.000Z"
}
{
  "stn": "MS#00001",
  "identity": "01",
  "latitude": 0.10001,
  "longitude": 0.00501,
  "speed": 46,
  "updatedAt": "2024-01-01T12:00:01.000Z"
}
```

## Best Practices

### 1. Message Structure
- Keep messages consistent in structure
- Use meaningful field names
- Include timestamps for time-series data

### 2. Performance
- Use appropriate delays between messages
- Consider client timeout settings
- Test with various message counts

### 3. Error Scenarios
- Test with empty streams
- Test with single message streams
- Test error conditions during streaming

### 4. Realistic Data
- Use realistic data patterns
- Include proper timestamps
- Simulate real-world scenarios

## Limitations

- Streams terminate after sending all messages
- No dynamic message generation during streaming
- Maximum stream length depends on available memory
- Client must handle stream termination gracefully 