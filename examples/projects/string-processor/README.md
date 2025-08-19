# String Processor Service - Bidirectional Streaming

This example demonstrates bidirectional streaming with gRPC using Gripmock.

## Service Definition

The `StringProcessorService` provides a `BidiStream` method that:
- Accepts a stream of `TextRequest` messages
- Returns a stream of `TextResponse` messages
- Processes text input and can return multiple responses for a single request

## Test Cases

### Bidirectional Streaming Test

The test case demonstrates:
1. Sending "Hello" and receiving "Hello" back
2. Sending "World from gRPC" and receiving three separate responses: "World", "from", "gRPC"

## Running the Example

```bash
# Start gripmock with the string processor service
go run main.go examples/projects/string-processor --stub examples/projects/string-processor

# Run the test case
grpctestify examples/projects/string-processor
```

## Message Types

- `TextRequest`: Contains a `text` field with the input message
- `TextResponse`: Contains a `result` field with the processed output
