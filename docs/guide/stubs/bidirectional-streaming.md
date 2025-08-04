# Bidirectional Streaming

Bidirectional streaming enables real-time two-way communication between client and server, where both sides can send multiple messages independently. This is ideal for chat applications, real-time collaboration, and interactive data processing.

## Overview

Bidirectional streaming is useful for:
- Real-time chat applications
- Live collaboration tools
- Interactive data processing
- Gaming and real-time applications
- Multi-party communication systems

## Basic Configuration

### V2 API Format (Recommended)
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

### Legacy V1 API Format (Backward Compatible)
```yaml
- service: ChatService
  method: Chat
  input:
    equals:
      user_id: "alice"
      content: "Hello"
      type: "MESSAGE_TYPE_TEXT"
  output:
    data:
      user_id: "bob"
      content: "Hello Alice!"
      type: "MESSAGE_TYPE_TEXT"
```

**Note**: V2 API format with `stream` input is recommended for bidirectional streaming. V1 format is supported for backward compatibility.

## How Bidirectional Streaming Works

### Message Flow
1. **Client sends first message** → Server matches against stub patterns
2. **Server responds** → Based on matched stub and message index
3. **Client sends second message** → Server re-evaluates matching stubs
4. **Server responds** → Based on updated stub selection
5. **Process continues** → Until client closes the stream

### Stub Matching Process
```yaml
# Example: Chat conversation
- service: ChatService
  method: Chat
  stream:
    # First message pattern
    - equals:
        user_id: "alice"
        content: "Hello"
        type: "MESSAGE_TYPE_TEXT"
    # Second message pattern
    - equals:
        user_id: "alice"
        content: "How are you?"
        type: "MESSAGE_TYPE_TEXT"
    # Third message pattern
    - equals:
        user_id: "alice"
        content: "Let's chat"
        type: "MESSAGE_TYPE_TEXT"
  output:
    stream:
      # Response to first message
      - data:
          user_id: "bob"
          content: "Hello Alice!"
          type: "MESSAGE_TYPE_TEXT"
      # Response to second message
      - data:
          user_id: "bob"
          content: "I'm doing great!"
          type: "MESSAGE_TYPE_TEXT"
      # Response to third message
      - data:
          user_id: "bob"
          content: "Sure, let's chat!"
          type: "MESSAGE_TYPE_TEXT"
```

## Advanced Examples

### Multi-User Chat Room
```yaml
- service: ChatService
  method: Chat
  stream:
    - equals:
        user_id: "alice"
        room_id: "room_001"
        content: "Hello everyone!"
        type: "MESSAGE_TYPE_TEXT"
    - equals:
        user_id: "bob"
        room_id: "room_001"
        content: "Hi Alice!"
        type: "MESSAGE_TYPE_TEXT"
    - equals:
        user_id: "charlie"
        room_id: "room_001"
        content: "Hello there!"
        type: "MESSAGE_TYPE_TEXT"
  output:
    stream:
      - data:
          user_id: "system"
          content: "Alice joined the chat"
          type: "MESSAGE_TYPE_SYSTEM"
          timestamp: "2024-01-01T12:00:00.000Z"
      - data:
          user_id: "bob"
          content: "Hi Alice!"
          type: "MESSAGE_TYPE_TEXT"
          timestamp: "2024-01-01T12:00:01.000Z"
      - data:
          user_id: "charlie"
          content: "Hello there!"
          type: "MESSAGE_TYPE_TEXT"
          timestamp: "2024-01-01T12:00:02.000Z"
```

### Interactive Data Processing
```yaml
- service: DataProcessorService
  method: ProcessData
  stream:
    - equals:
        command: "START"
        dataset: "large_dataset"
        format: "JSON"
    - equals:
        command: "PROCESS"
        batch_size: 1000
    - equals:
        command: "COMPLETE"
        total_records: 50000
  output:
    stream:
      - data:
          status: "STARTED"
          message: "Processing started"
          dataset: "large_dataset"
          timestamp: "2024-01-01T12:00:00.000Z"
      - data:
          status: "PROCESSING"
          message: "Processing batch of 1000 records"
          progress: 20
          timestamp: "2024-01-01T12:00:01.000Z"
      - data:
          status: "COMPLETED"
          message: "Processing completed"
          total_processed: 50000
          timestamp: "2024-01-01T12:00:02.000Z"
```

### Real-Time Collaboration
```yaml
- service: CollaborationService
  method: Collaborate
  stream:
    - equals:
        user_id: "editor_1"
        action: "CURSOR_MOVE"
        position: {"x": 100, "y": 200}
    - equals:
        user_id: "editor_1"
        action: "TEXT_INSERT"
        text: "Hello world"
        position: {"line": 1, "column": 1}
    - equals:
        user_id: "editor_2"
        action: "CURSOR_MOVE"
        position: {"x": 150, "y": 250}
  output:
    stream:
      - data:
          user_id: "editor_1"
          action: "CURSOR_MOVE"
          position: {"x": 100, "y": 200}
          timestamp: "2024-01-01T12:00:00.000Z"
      - data:
          user_id: "editor_1"
          action: "TEXT_INSERT"
          text: "Hello world"
          position: {"line": 1, "column": 1}
          timestamp: "2024-01-01T12:00:01.000Z"
      - data:
          user_id: "editor_2"
          action: "CURSOR_MOVE"
          position: {"x": 150, "y": 250}
          timestamp: "2024-01-01T12:00:02.000Z"
```

## Stub Ranking and Selection

### Dynamic Stub Filtering
GripMock uses sophisticated algorithms for bidirectional streaming:

1. **Initial Matching**: All stubs are evaluated for the first message
2. **Progressive Filtering**: Each subsequent message narrows down matching stubs
3. **Ranking**: Stubs are ranked based on:
   - Exact message matches
   - Field overlap
   - Specificity
   - Priority values
4. **Selection**: Best matching stub is selected for response

### Priority Control
```yaml
# High priority stub for specific conversation
- service: ChatService
  method: Chat
  priority: 100
  stream:
    - equals:
        user_id: "alice"
        content: "Hello"
        type: "MESSAGE_TYPE_TEXT"
  output:
    stream:
      - data:
          user_id: "bob"
          content: "Hello Alice! (High priority)"
          type: "MESSAGE_TYPE_TEXT"

# Lower priority fallback
- service: ChatService
  method: Chat
  priority: 50
  stream:
    - equals:
        user_id: "alice"
        content: "Hello"
        type: "MESSAGE_TYPE_TEXT"
  output:
    stream:
      - data:
          user_id: "bob"
          content: "Hello Alice! (Fallback)"
          type: "MESSAGE_TYPE_TEXT"
```

## Best Practices

### 1. Use Consistent Message Structure
```yaml
# Good: Consistent structure
stream:
  - equals:
      user_id: "alice"
      content: "Hello"
      type: "MESSAGE_TYPE_TEXT"
      timestamp: "2024-01-01T12:00:00.000Z"
  - equals:
      user_id: "alice"
      content: "How are you?"
      type: "MESSAGE_TYPE_TEXT"
      timestamp: "2024-01-01T12:00:01.000Z"

# Avoid: Inconsistent structure
stream:
  - equals:
      user_id: "alice"
      content: "Hello"
  - equals:
      user_id: "alice"
      message: "How are you?"  # Different field name
```

### 2. Include Timestamps
```yaml
# Good: Include timestamps for message ordering
output:
  stream:
    - data:
        user_id: "bob"
        content: "Hello Alice!"
        timestamp: "2024-01-01T12:00:00.000Z"
    - data:
        user_id: "bob"
        content: "I'm doing great!"
        timestamp: "2024-01-01T12:00:01.000Z"

# Avoid: Missing timestamps
output:
  stream:
    - data:
        user_id: "bob"
        content: "Hello Alice!"
    - data:
        user_id: "bob"
        content: "I'm doing great!"
```

### 3. Handle Message Sequences
```yaml
# Good: Clear message sequence
stream:
  - equals:
      sequence: 1
      content: "First message"
  - equals:
      sequence: 2
      content: "Second message"
  - equals:
      sequence: 3
      content: "Third message"

# Avoid: Ambiguous sequence
stream:
  - equals:
      content: "First message"
  - equals:
      content: "Second message"
  - equals:
      content: "Third message"
```

### 4. Use Appropriate Message Types
```yaml
# Good: Clear message types
stream:
  - equals:
      type: "MESSAGE_TYPE_TEXT"
      content: "Hello"
  - equals:
      type: "MESSAGE_TYPE_SYSTEM"
      content: "User joined"
  - equals:
      type: "MESSAGE_TYPE_FILE"
      file_id: "file_001"

# Avoid: Mixed message types
stream:
  - equals:
      content: "Hello"
  - equals:
      system_message: "User joined"
  - equals:
      file: "file_001"
```

### 5. Provide Meaningful Responses
```yaml
# Good: Contextual responses
output:
  stream:
    - data:
        user_id: "bob"
        content: "Hello Alice! Nice to see you."
        type: "MESSAGE_TYPE_TEXT"
    - data:
        user_id: "bob"
        content: "I'm doing great! How about you?"
        type: "MESSAGE_TYPE_TEXT"

# Avoid: Generic responses
output:
  stream:
    - data:
        user_id: "bob"
        content: "Response 1"
        type: "MESSAGE_TYPE_TEXT"
    - data:
        user_id: "bob"
        content: "Response 2"
        type: "MESSAGE_TYPE_TEXT"
```

## Error Handling

### Stream Interruption
```yaml
- service: ChatService
  method: Chat
  stream:
    - equals:
        user_id: "alice"
        content: "Hello"
  output:
    error: "Connection lost"
    code: 14  # UNAVAILABLE
```

### Invalid Message Format
```yaml
- service: ChatService
  method: Chat
  stream:
    - equals:
        user_id: "alice"
        content: "invalid_message"
  output:
    error: "Invalid message format"
    code: 3  # INVALID_ARGUMENT
```

## Testing Considerations

### 1. Test Message Sequences
- Single message conversations
- Multi-message conversations
- Long conversation threads
- Mixed message types

### 2. Test Concurrent Users
- Multiple users in same conversation
- Different user roles and permissions
- User join/leave scenarios

### 3. Test Error Scenarios
- Network interruptions
- Invalid message formats
- Authentication failures
- Rate limiting

### 4. Test Performance
- High message frequency
- Large message payloads
- Extended conversation duration
- Memory usage under load

## Limitations

- **Message Order**: Messages are processed in the order received
- **State Management**: Limited state persistence between messages
- **Concurrent Users**: Each connection maintains separate state
- **Message Size**: Limited by gRPC message size limits
- **Connection Duration**: Limited by client/server timeout settings

## Migration from V1 to V2

### V1 Format (Legacy)
```yaml
- service: ChatService
  method: Chat
  input:
    equals:
      user_id: "alice"
      content: "Hello"
  output:
    data:
      user_id: "bob"
      content: "Hello Alice!"
```

### V2 Format (Recommended)
```yaml
- service: ChatService
  method: Chat
  stream:
    - equals:
        user_id: "alice"
        content: "Hello"
  output:
    stream:
      - data:
          user_id: "bob"
          content: "Hello Alice!"
```

**Migration Benefits:**
- Better support for multi-message conversations
- Improved stub matching and ranking
- Enhanced performance and scalability
- Future-proof API design 