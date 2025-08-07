# Client-Side Streaming

Client-side streaming lets you send multiple messages and get one response back - perfect for file uploads, batch processing, and data collection. Think of it like sending a package in multiple boxes and getting one confirmation when everything arrives.

## What is Client Streaming?

Imagine uploading a large file in pieces. Your client sends multiple messages (like file chunks), and the server responds with a single summary after receiving everything. It's that simple!

## When to Use It

- **File Uploads**: Upload large files in smaller, manageable chunks
- **Batch Processing**: Send multiple items and get one summary report
- **Data Collection**: Gather data from multiple sources at once
- **Progressive Forms**: Submit form data step by step
- **Sensor Data**: Collect readings from multiple sensors

## Getting Started

### New Format (V2) - Recommended
```yaml
- service: FileService
  method: UploadFile
  inputs:
    - equals:
        chunk_id: "chunk_001"
        sequence: 1
        total_chunks: 3
        content_type: "text/plain"
    - equals:
        chunk_id: "chunk_001"
        sequence: 2
        total_chunks: 3
        content_type: "text/plain"
    - equals:
        chunk_id: "chunk_001"
        sequence: 3
        total_chunks: 3
        content_type: "text/plain"
  output:
    data:
      upload_id: "upload_001"
      success: true
      total_chunks: 3
      total_size: "1500"
      status: "completed"
      completed_at: "2024-01-15T10:05:00Z"
```

### Old Format (V1) - Still Works
```yaml
- service: FileService
  method: UploadFile
  input:
    equals:
      chunk: 1
      data: "file_header"
  output:
    data:
      status: "uploaded"
      file_id: "abc123"
```

**Note**: The new V2 format is recommended, but the old V1 format still works perfectly!

## How It Works

### Step-by-Step Process
1. **Client sends first chunk** → Server starts collecting
2. **Client sends second chunk** → Server continues collecting
3. **Client sends third chunk** → Server continues collecting
4. **Client closes the stream** → Server processes all chunks
5. **Server sends single response** → Based on all collected data

### Real-World Example
```yaml
# Example: Uploading a file in 3 chunks
- service: UploadService
  method: UploadFile
  inputs:
    # First chunk pattern
    - equals:
        chunk_id: "chunk_001"
        sequence: 1
        total_chunks: 3
        content_type: "text/plain"
    # Second chunk pattern
    - equals:
        chunk_id: "chunk_001"
        sequence: 2
        total_chunks: 3
        content_type: "text/plain"
    # Third chunk pattern
    - equals:
        chunk_id: "chunk_001"
        sequence: 3
        total_chunks: 3
        content_type: "text/plain"
  output:
    data:
      upload_id: "upload_001"
      success: true
      total_chunks: 3
      total_size: "1500"
      status: "completed"
```

## Real-World Examples

### Large File Upload
```yaml
- service: UploadService
  method: UploadLargeFile
  inputs:
    - equals:
        chunk_id: "large_file_001"
        sequence: 1
        total_chunks: 10
        content_type: "video/mp4"
    - equals:
        chunk_id: "large_file_001"
        sequence: 2
        total_chunks: 10
        content_type: "video/mp4"
    # ... more chunks ...
    - equals:
        chunk_id: "large_file_001"
        sequence: 10
        total_chunks: 10
        content_type: "video/mp4"
  output:
    data:
      upload_id: "upload_large_001"
      success: true
      total_chunks: 10
      total_size: "10485760"
      status: "completed"
      completed_at: "2024-01-15T11:30:00Z"
```

### Sensor Data Collection
```yaml
- service: SensorService
  method: CollectReadings
  inputs:
    - equals:
        sensor_id: "TEMP_001"
        reading: 22.5
        timestamp: "2024-01-01T10:00:00Z"
        location: "room_1"
    - equals:
        sensor_id: "HUMIDITY_001"
        reading: 45.2
        timestamp: "2024-01-01T10:00:00Z"
        location: "room_1"
    - equals:
        sensor_id: "PRESSURE_001"
        reading: 1013.25
        timestamp: "2024-01-01T10:00:00Z"
        location: "room_1"
  output:
    data:
      collection_id: "collection_001"
      success: true
      sensors_processed: 3
      total_readings: 3
      timestamp: "2024-01-01T10:00:00Z"
      summary:
        temperature_avg: 22.5
        humidity_avg: 45.2
        pressure_avg: 1013.25
```

### Batch Processing
```yaml
- service: ProcessingService
  method: ProcessBatch
  inputs:
    - equals:
        item_id: "item_001"
        data: "first_item_data"
        type: "text"
    - equals:
        item_id: "item_002"
        data: "second_item_data"
        type: "text"
    - equals:
        item_id: "item_003"
        data: "third_item_data"
        type: "text"
  output:
    data:
      batch_id: "batch_001"
      success: true
      items_processed: 3
      processing_time_ms: 150
      status: "completed"
      results:
        - item_id: "item_001"
          status: "processed"
          result: "processed_first_item"
        - item_id: "item_002"
          status: "processed"
          result: "processed_second_item"
        - item_id: "item_003"
          status: "processed"
          result: "processed_third_item"
```

## Smart Stub Selection

GripMock uses clever algorithms to pick the best matching stub:

1. **Initial Matching**: All stubs are checked against the first message
2. **Progressive Filtering**: Each new message narrows down the options
3. **Cumulative Ranking**: Stubs are ranked based on all received messages
4. **Length Compatibility**: Stubs with matching length patterns rank higher
5. **Specificity**: More specific field matches get better rankings

### Priority Control
```yaml
# High priority for specific file types
- service: UploadService
  method: UploadFile
  priority: 100
  inputs:
    - equals:
        chunk_id: "chunk_001"
        content_type: "video/mp4"
        sequence: 1
        total_chunks: 3
  output:
    data:
      upload_id: "upload_001"
      success: true
      priority: "high"

# Lower priority fallback
- service: UploadService
  method: UploadFile
  priority: 50
  inputs:
    - equals:
        chunk_id: "chunk_001"
        sequence: 1
        total_chunks: 3
  output:
    data:
      upload_id: "upload_001"
      success: true
      priority: "normal"
```

## Best Practices

### 1. Keep Your Structure Consistent
```yaml
# Good: Consistent structure
inputs:
  - equals:
      chunk_id: "chunk_001"
      sequence: 1
      total_chunks: 3
      content_type: "text/plain"
  - equals:
      chunk_id: "chunk_001"
      sequence: 2
      total_chunks: 3
      content_type: "text/plain"
  - equals:
      chunk_id: "chunk_001"
      sequence: 3
      total_chunks: 3
      content_type: "text/plain"

# Avoid: Inconsistent structure
inputs:
  - equals:
      chunk: 1
      data: "file_header"
  - equals:
      chunk_id: "chunk_001"
      sequence: 2
      total_chunks: 3
  - equals:
      chunk: 3
      footer: "file_footer"
```

### 2. Include Sequence Information
```yaml
# Good: Clear sequence information
inputs:
  - equals:
      chunk_id: "chunk_001"
      sequence: 1
      total_chunks: 3
      content_type: "text/plain"
  - equals:
      chunk_id: "chunk_001"
      sequence: 2
      total_chunks: 3
      content_type: "text/plain"
  - equals:
      chunk_id: "chunk_001"
      sequence: 3
      total_chunks: 3
      content_type: "text/plain"

# Avoid: Missing sequence information
inputs:
  - equals:
      chunk_id: "chunk_001"
      content_type: "text/plain"
  - equals:
      chunk_id: "chunk_001"
      content_type: "text/plain"
  - equals:
      chunk_id: "chunk_001"
      content_type: "text/plain"
```

### 3. Provide Meaningful Responses
```yaml
# Good: Comprehensive response
output:
  data:
    upload_id: "upload_001"
    success: true
    total_chunks: 3
    total_size: "1500"
    status: "completed"
    completed_at: "2024-01-15T10:05:00Z"
    checksum: "abc123def456"
    metadata:
      original_filename: "document.txt"
      content_type: "text/plain"
      compression: "none"

# Avoid: Minimal response
output:
  data:
    success: true
```

### 4. Handle Different File Sizes
```yaml
# Small file (3 chunks)
- service: UploadService
  method: UploadFile
  inputs:
    - equals:
        chunk_id: "small_file_001"
        sequence: 1
        total_chunks: 3
        content_type: "text/plain"
    - equals:
        chunk_id: "small_file_001"
        sequence: 2
        total_chunks: 3
        content_type: "text/plain"
    - equals:
        chunk_id: "small_file_001"
        sequence: 3
        total_chunks: 3
        content_type: "text/plain"
  output:
    data:
      upload_id: "upload_small_001"
      success: true
      total_chunks: 3
      total_size: "1500"

# Large file (10 chunks)
- service: UploadService
  method: UploadFile
  inputs:
    - equals:
        chunk_id: "large_file_001"
        sequence: 1
        total_chunks: 10
        content_type: "video/mp4"
    # ... additional chunks ...
    - equals:
        chunk_id: "large_file_001"
        sequence: 10
        total_chunks: 10
        content_type: "video/mp4"
  output:
    data:
      upload_id: "upload_large_001"
      success: true
      total_chunks: 10
      total_size: "10485760"
```

### 5. Use Appropriate Content Types
```yaml
# Good: Specific content types
inputs:
  - equals:
      chunk_id: "chunk_001"
      content_type: "text/plain"
      sequence: 1
      total_chunks: 3
  - equals:
      chunk_id: "chunk_001"
      content_type: "video/mp4"
      sequence: 1
      total_chunks: 10

# Avoid: Generic content types
inputs:
  - equals:
      chunk_id: "chunk_001"
      content_type: "file"
      sequence: 1
      total_chunks: 3
```

## Error Handling

### Upload Failure
```yaml
- service: UploadService
  method: UploadFile
  inputs:
    - equals:
        chunk_id: "invalid_chunk"
        sequence: 1
        total_chunks: 3
  output:
    error: "Invalid chunk format"
    code: 3  # INVALID_ARGUMENT
```

### Partial Upload
```yaml
- service: UploadService
  method: UploadFile
  inputs:
    - equals:
        chunk_id: "chunk_001"
        sequence: 1
        total_chunks: 3
    - equals:
        chunk_id: "chunk_001"
        sequence: 2
        total_chunks: 3
  output:
    error: "Incomplete upload - missing chunk 3"
    code: 9  # FAILED_PRECONDITION
```

## Testing Tips

### 1. Test Different Chunk Sizes
- Small files (1-5 chunks)
- Medium files (10-50 chunks)
- Large files (100+ chunks)

### 2. Test Upload Scenarios
- Complete uploads
- Partial uploads
- Failed uploads
- Duplicate chunks
- Out-of-order chunks

### 3. Test Content Types
- Text files
- Binary files
- Images
- Videos
- Documents

### 4. Test Performance
- High-frequency uploads
- Large file uploads
- Concurrent uploads
- Memory usage under load

## What You Need to Know

- **Message Order**: Messages must be sent in the defined order
- **Stream Length**: Maximum stream length depends on available memory
- **Response Timing**: Response is sent only after stream is closed
- **State Management**: Limited state persistence between messages
- **Concurrent Uploads**: Each upload maintains separate state

## Migration Guide

### Old Format (V1)
```yaml
- service: FileService
  method: UploadFile
  input:
    equals:
      chunk: 1
      data: "file_header"
  output:
    data:
      status: "uploaded"
      file_id: "abc123"
```

### New Format (V2) - Recommended
```yaml
- service: FileService
  method: UploadFile
  inputs:
    - equals:
        chunk_id: "chunk_001"
        sequence: 1
        total_chunks: 3
        content_type: "text/plain"
  output:
    data:
      upload_id: "upload_001"
      success: true
      total_chunks: 3
      total_size: "1500"
      status: "completed"
```

**Migration Benefits:**
- Better support for multi-chunk uploads
- Improved stub matching and ranking
- Enhanced performance and scalability
- Future-proof API design 