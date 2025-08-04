# Better Performance and Streaming Support

GripMock is now faster and supports all types of gRPC streaming while keeping all your existing tests working.

## What's New?

The latest version brings you:
- **Faster Tests**: Your tests run 20-35% faster
- **All Streaming Types**: Test file uploads, chat apps, and real-time features
- **More Reliable**: Consistent results every time you run tests
- **No Breaking Changes**: Everything you have now keeps working

## Why Should You Care?

### For New Projects
Start with the latest version to get the best performance and all streaming features from the beginning.

### For Existing Projects
Your current setup keeps working. Add new streaming features when you need them.

### For Teams That Need Speed
Get faster test execution and more reliable results.

## Streaming Support

Now you can test all types of gRPC communication:

### File Uploads (Multiple Requests → Single Response)
Perfect for testing file uploads, batch processing, and data collection.

```yaml
service: FileService
method: UploadFile
stream:
  - equals:
      chunk: 1
      data: "file_header"
  - equals:
      chunk: 2
      data: "file_content"
  - equals:
      chunk: 3
      data: "file_footer"
output:
  data:
    status: "uploaded"
    file_id: "doc_123"
```

### Real-time Chat (Multiple Requests ↔ Multiple Responses)
Ideal for testing chat applications, live collaboration, and interactive features.

```yaml
service: ChatService
method: Chat
stream:
  - equals:
      user: Alice
      text: "Hello, how are you?"
  - equals:
      user: Alice
      text: "That's great to hear!"
output:
  data:
    user: Bot
    text: "Great conversation! Have a wonderful day!"
```

## Getting Started

### Your Existing Tests Keep Working
All your current test configurations continue to work exactly as before. No changes needed!

### Adding Streaming Tests
When you want to test streaming features, use the new `stream` field:

**Before (Simple requests):**
```yaml
service: ChatService
method: SendMessage
input:
  equals:
    user: Alice
    text: "Hello"
output:
  data:
    success: true
```

**After (Streaming requests):**
```yaml
service: ChatService
method: SendMessage
stream:
  - equals:
      user: Alice
      text: "Hello"
  - equals:
      user: Alice
      text: "How are you?"
output:
  data:
    success: true
    message: "2 messages processed"
```

## Real-World Examples

### File Upload Service
```yaml
service: FileService
method: UploadFile
stream:
  - equals:
      chunk: 1
      data: "file_header"
      filename: "document.pdf"
  - equals:
      chunk: 2
      data: "file_content"
      size: 1024
  - equals:
      chunk: 3
      data: "file_footer"
      checksum: "abc123"
output:
  data:
    status: "uploaded"
    file_id: "doc_123"
```

### Real-time Chat
```yaml
service: ChatService
method: Chat
stream:
  - equals:
      user: Alice
      text: "Hello, how are you?"
  - equals:
      user: Alice
      text: "That's great to hear!"
  - equals:
      user: Alice
      text: "See you later!"
output:
  data:
    user: Bot
    text: "Great conversation! Have a wonderful day!"
```

### Sensor Data Collection
```yaml
service: SensorService
method: CollectData
stream:
  - equals:
      sensor_id: "TEMP_001"
      reading: 22.5
      timestamp: "2024-01-01T10:00:00Z"
  - equals:
      sensor_id: "TEMP_001"
      reading: 22.7
      timestamp: "2024-01-01T10:01:00Z"
  - equals:
      sensor_id: "TEMP_001"
      reading: 23.1
      timestamp: "2024-01-01T10:02:00Z"
output:
  data:
    average_temp: 22.8
    readings_count: 3
    status: "processed"
```

## What You Get

### 1. No Action Required
Your existing setup continues to work. New features are automatically available.

### 2. Try New Features When Ready
Start using streaming features for new test scenarios.

### 3. Enjoy Better Performance
Experience faster test execution and more reliable results.

## Benefits Summary

| Feature | Before | Now | What This Means |
|---------|--------|-----|-----------------|
| Speed | Baseline | 20-35% faster | Your tests run quicker |
| File Uploads | ❌ | ✅ | Test chunked file uploads |
| Real-time Chat | ❌ | ✅ | Test chat apps and live features |
| Reliability | Good | Better | More consistent test results |
| Compatibility | ✅ | ✅ | All existing code works unchanged |

## When to Use New Features

- **New Projects**: Start with new features for best performance
- **Existing Projects**: Add streaming features when you need them
- **Performance Needs**: Use new features for faster test execution
- **Streaming Requirements**: Use new features for file uploads and chat testing

## What's Next?

We're always working on improvements:
- Even better performance
- More streaming patterns
- Better debugging tools
- Integration with popular testing frameworks 