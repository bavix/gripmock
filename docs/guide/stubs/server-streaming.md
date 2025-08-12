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

### V2 API Format (Recommended)
```yaml
- service: TrackService
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

**Note**: Stream elements contain message data directly, without a `data` wrapper.

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

### Real-Time Sensor Data
```yaml
- service: SensorService
  method: StreamReadings
  input:
    equals:
      sensor_id: "TEMP_001"
      duration: 60
  output:
    delay: 1000ms
    stream:
      - sensor_id: "TEMP_001"
        temperature: 22.5
        humidity: 45.2
        pressure: 1013.25
        timestamp: "2024-01-01T10:00:00.000Z"
      - sensor_id: "TEMP_001"
        temperature: 22.7
        humidity: 45.8
        pressure: 1013.30
        timestamp: "2024-01-01T10:00:01.000Z"
      - sensor_id: "TEMP_001"
        temperature: 23.1
        humidity: 46.1
        pressure: 1013.35
        timestamp: "2024-01-01T10:00:02.000Z"
```

### Batch Processing Progress
```yaml
- service: ProcessingService
  method: StreamProgress
  input:
    equals:
      batch_id: "BATCH_001"
      total_items: 100
  output:
    delay: 500ms
    stream:
      - batch_id: "BATCH_001"
        status: "STARTED"
        progress: 0
        processed: 0
        total: 100
        message: "Processing started"
      - batch_id: "BATCH_001"
        status: "PROCESSING"
        progress: 25
        processed: 25
        total: 100
        message: "25% complete"
      - batch_id: "BATCH_001"
        status: "PROCESSING"
        progress: 50
        processed: 50
        total: 100
        message: "50% complete"
      - batch_id: "BATCH_001"
        status: "PROCESSING"
        progress: 75
        processed: 75
        total: 100
        message: "75% complete"
      - batch_id: "BATCH_001"
        status: "COMPLETED"
        progress: 100
        processed: 100
        total: 100
        message: "Processing completed"
```

### Event Notifications
```yaml
- service: NotificationService
  method: StreamNotifications
  input:
    equals:
      user_id: "12345"
      event_types: ["INFO", "WARNING", "SUCCESS"]
  output:
    delay: 2000ms
    stream:
      - type: "INFO"
        message: "Welcome to the service"
        timestamp: "2024-01-01T12:00:00.000Z"
        user_id: "12345"
      - type: "WARNING"
        message: "Your session will expire soon"
        timestamp: "2024-01-01T12:00:02.000Z"
        user_id: "12345"
      - type: "SUCCESS"
        message: "Operation completed successfully"
        timestamp: "2024-01-01T12:00:04.000Z"
        user_id: "12345"
```

## Performance and Ranking

### Stub Selection
GripMock uses sophisticated ranking algorithms to select the best matching stub for server streaming:

1. **Exact Input Match**: Highest priority for perfect input matches
2. **Field Overlap**: Ranked based on number of matching fields
3. **Specificity**: More specific stubs rank higher
4. **Priority**: Explicit priority values override ranking
5. **Stream Length**: Considered for optimal matching

### Priority Control
```yaml
# High priority stub for specific conditions
- service: DataService
  method: StreamData
  priority: 100
  input:
    equals:
      request_id: "req_001"
      exact_match: true
  output:
    stream:
      - message: "Exact match response"
        priority: "high"

# Lower priority fallback
- service: DataService
  method: StreamData
  priority: 50
  input:
    equals:
      request_id: "req_001"
  output:
    stream:
      - message: "Fallback response"
        priority: "low"
```

## Best Practices

### 1. Use Consistent Data Structure
```yaml
# Good: Consistent structure
output:
  stream:
    - message: "First"
      timestamp: "2024-01-01T12:00:00.000Z"
    - message: "Second"
      timestamp: "2024-01-01T12:00:01.000Z"

# Avoid: Inconsistent structure
output:
  stream:
    - message: "First"
    - message: "Second"
      timestamp: "2024-01-01T12:00:01.000Z"
```

### 2. Include Timestamps
```yaml
# Good: Include timestamps for time-sensitive data
output:
  stream:
    - sensor_reading: 22.5
      timestamp: "2024-01-01T12:00:00.000Z"
    - sensor_reading: 22.7
      timestamp: "2024-01-01T12:00:01.000Z"

# Avoid: Missing timestamps
output:
  stream:
    - sensor_reading: 22.5
    - sensor_reading: 22.7
```

### 3. Use Appropriate Delays
```yaml
# Good: Realistic delays for testing
output:
  delay: 1000ms  # 1 second between messages
  stream:
    - message: "Real-time data"

# Avoid: Excessive delays
output:
  delay: 30s  # Too long for most use cases
  stream:
    - message: "Slow data"
```

### 4. Handle Large Streams Efficiently
```yaml
# Good: Reasonable stream length
output:
  stream:
    - chunk: 1
      total: 10
    # ... up to 10 chunks

# Avoid: Extremely long streams
output:
  stream:
    - chunk: 1
      total: 10000
    # ... 10,000 chunks (too many)
```

### 5. Include Progress Information
```yaml
# Good: Include progress metadata
output:
  stream:
    - message: "Processing started"
      progress: 0
      total: 100
    - message: "25% complete"
        progress: 25
        total: 100
    - message: "50% complete"
      progress: 50
      total: 100
    - message: "Completed"
      progress: 100
      total: 100
```

## Error Handling

### Stream with Error Response
```yaml
- service: DataService
  method: StreamData
  input:
    equals:
      request_id: "invalid_request"
  output:
    error: "Invalid request ID format"
    code: 3  # INVALID_ARGUMENT
```

### Conditional Error in Stream
```yaml
- service: ProcessingService
  method: StreamProgress
  input:
    equals:
      batch_id: "BATCH_001"
  output:
    stream:
      - status: "STARTED"
        message: "Processing started"
      - status: "ERROR"
        message: "Processing failed"
        error_code: "INSUFFICIENT_RESOURCES"
```

## Testing Considerations

### 1. Test Different Stream Lengths
- Short streams (1-5 messages)
- Medium streams (10-50 messages)
- Long streams (100+ messages)

### 2. Test Timing Scenarios
- No delay
- Short delays (100-500ms)
- Long delays (1-5 seconds)

### 3. Test Error Conditions
- Invalid input parameters
- Resource exhaustion
- Network timeouts

### 4. Test Concurrent Streams
- Multiple clients streaming simultaneously
- Different stream types running concurrently

## Limitations

- **Maximum Stream Length**: Limited by available memory
- **Delay Precision**: Depends on system clock resolution
- **Concurrent Streams**: Limited by system resources
- **Message Size**: Limited by gRPC message size limits
- **Stream Duration**: No infinite streams - all streams complete naturally 