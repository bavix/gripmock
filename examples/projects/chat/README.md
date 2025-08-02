# Chat Service

Chat service example with support for different streaming types.

## What it does

- Handles messages from different users
- Supports client, server and bidirectional streaming
- Shows real chat scenarios

## Run

```bash
gripmock --stub examples/projects/chat examples/projects/chat/service.proto
```

## Tests

```bash
grpctestify examples/projects/chat/
```

## Structure

- `service.proto` - gRPC service definition
- `stubs.yaml` - mock responses for testing
- `*.gctf` - test scenarios

## Features

- **Client Streaming**: Send multiple messages from one client
- **Server Streaming**: Receive message stream from server
- **Bidirectional**: Real-time two-way chat
- **User Context**: Different responses for different users 