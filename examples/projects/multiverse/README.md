# Multiverse - All Streaming Types Demo

This project demonstrates all four types of gRPC streaming in a single service:

## Streaming Types

### 1. Unary
- **Method**: `Ping`
- **Pattern**: Single request → Single response
- **Use Case**: Simple health checks, basic API calls

### 2. Client Streaming
- **Method**: `UploadData`
- **Pattern**: Multiple requests → Single response
- **Use Case**: File uploads, batch data collection

### 3. Server Streaming
- **Method**: `StreamData`
- **Pattern**: Single request → Multiple responses
- **Use Case**: Real-time data feeds, progress updates

### 4. Bidirectional Streaming
- **Method**: `Chat`
- **Pattern**: Multiple requests ↔ Multiple responses
- **Use Case**: Real-time chat, interactive sessions

## Project Structure

```
multiverse/
├── service.proto                    # Service definition with all streaming types
├── tests/                          # Test cases organized by streaming type
│   ├── unary/                      # Unary streaming tests
│   │   ├── case_*.gctf            # Test cases
│   │   └── stubs.yaml             # Stubs for unary tests
│   ├── client-streaming/           # Client streaming tests
│   │   ├── case_*.gctf            # Test cases
│   │   └── stubs.yaml             # Stubs for client streaming tests
│   ├── server-streaming/           # Server streaming tests
│   │   ├── case_*.gctf            # Test cases
│   │   └── stubs.yaml             # Stubs for server streaming tests
│   └── bidi-streaming/             # Bidirectional streaming tests
│       ├── case_*.gctf            # Test cases
│       └── stubs.yaml             # Stubs for bidirectional streaming tests
└── README.md                       # This file
```

## Running Tests

```bash
# Run all tests by type
grpctestify tests/unary/case_*.gctf           # Unary tests
grpctestify tests/client-streaming/case_*.gctf # Client streaming tests
grpctestify tests/server-streaming/case_*.gctf # Server streaming tests
grpctestify tests/bidi-streaming/case_*.gctf   # Bidirectional streaming tests

# Run individual tests
grpctestify tests/unary/case_unary_ping.gctf
grpctestify tests/client-streaming/case_client_streaming_upload.gctf
grpctestify tests/server-streaming/case_server_streaming_data.gctf
grpctestify tests/bidi-streaming/case_bidi_streaming_chat.gctf

# Run all tests (from project root)
grpctestify tests/*/case_*.gctf
```

## Features Demonstrated

- **Individual Stub Files**: Each test has its own `.stub.yaml` file for clarity
- **Stream Keys**: All streaming stubs use `inputs` keys for input and `stream` keys for output
- **Multiple Scenarios**: Each streaming type has multiple test scenarios
- **Real-world Patterns**: Practical examples like file uploads and chat
- **Large Data**: Tests with many chunks to verify stability
- **Stateful Matching**: Bidirectional streaming with conversation patterns

## Key Concepts

- **Unary**: Simple request-response pattern
- **Client Streaming**: Client sends multiple messages, server responds once
- **Server Streaming**: Client sends one message, server responds multiple times
- **Bidirectional**: Both client and server send multiple messages in real-time

This project serves as a comprehensive example of how to use GripMock with all streaming types using the modern `inputs` key format. 