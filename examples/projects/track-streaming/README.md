# Track Streaming Example

This example demonstrates the new array streaming capability in gripmock.

## Service Definition

The `TrackService` has a server streaming method `StreamTrack` that returns a stream of `TrackData` messages.

## Stub Configuration

### Single Response Streaming (Original Behavior)
When the `data` field contains a single object, gripmock behaves as before - it sends the same response repeatedly for each client request.

### Array Response Streaming (New Behavior)
When the `data` field contains a `stream` array, gripmock will:
1. Continuously stream each item in the array
2. Loop back to the first item when reaching the end
3. Continue streaming until the client disconnects

## Stub Files

This example includes both YAML and JSON stub formats:

- `stubs.yaml` - YAML format with examples for both single and array streaming
- `stubs.json` - JSON format with additional streaming examples

## Stream Throttling

You can control the streaming speed using the `--stream-interval` flag:

```bash
# Default: 100ms between messages
gripmock --stub=./stubs.yaml ./service.proto

# Fast streaming: 50ms between messages  
gripmock --stream-interval=50ms --stub=./stubs.yaml ./service.proto

# Slow streaming: 1 second between messages
gripmock --stream-interval=1s --stub=./stubs.yaml ./service.proto

# Very fast streaming: 10ms between messages
gripmock --stream-interval=10ms --stub=./stubs.yaml ./service.proto
```

## Usage Examples

1. Start gripmock with custom stream interval:
   ```bash
   gripmock --stream-interval=200ms --stub=./stubs.yaml ./service.proto
   ```

2. Test with a gRPC client (like BloomRPC):
   - Request with `stn: "MS#00001"` - will get continuous array streaming (3 items)
   - Request with `stn: "MS#00002"` - will get single response repeated  
   - Request with `stn: "MS#00003"` - will get another continuous array streaming (2 items)
   - Request with `stn: "MS#00004"` - will get single response (from JSON)
   - Request with `stn: "MS#00005"` - will get continuous array streaming (4 items from JSON)
   - Request with `stn: "MS#00006"` - will get continuous array streaming (2 items from JSON)

## Stream Behavior

- **Array streaming**: Messages are sent continuously with the configured interval
- **Loop behavior**: When the array ends, it starts from the first item again
- **Delay priority**: If a stub specifies a `delay`, it takes precedence over `--stream-interval`
- **Context cancellation**: Streaming stops immediately when the client disconnects

The array streaming will loop indefinitely until the client closes the connection.
