# Nested Messages Example

This example demonstrates GripMock's support for nested message types in Protocol Buffers.

## Overview

The example shows how to handle:
- Deeply nested message definitions (e.g., `Config.Settings.NetworkSettings`)
- Nested enums within messages
- Multiple levels of message nesting
- References to parent message types in nested structures

## Proto Definition

The `service.proto` file defines:
- `ConfigurationService` with methods accepting nested messages
- `Config.Settings.NetworkSettings` - a three-level nested message
- `UserProfile.Profile.Preferences.NotificationSettings` - a four-level nested message
- Enum types defined within message scopes

## Running the Example

Start the mock server:

```bash
gripmock --stub examples/projects/nested-messages examples/projects/nested-messages/service.proto
```

Or using Docker:

```bash
docker run -p 4770:4770 -p 4771:4771 \
  -v $(pwd)/examples/projects/nested-messages:/proto \
  bavix/gripmock /proto
```

## Testing with grpcurl

Update configuration:

```bash
grpcurl -plaintext \
  -d '{
    "host": "api.example.com",
    "port": 443,
    "use_tls": true,
    "env": "PRODUCTION"
  }' \
  localhost:4770 nested.ConfigurationService/UpdateConfig
```

Get user profile:

```bash
grpcurl -plaintext \
  -d '{"user_id": "user123"}' \
  localhost:4770 nested.ConfigurationService/GetProfile
```

## Key Features Demonstrated

1. **Nested Message Types**: Messages defined within other messages
2. **Nested Enums**: Enumerations scoped to message types
3. **Deep Nesting**: Support for multiple levels of nesting
4. **Type Resolution**: Proper resolution of nested type names

## Notes

GripMock uses modern Protocol Buffer compilation tools that natively support nested message types. This feature works out of the box with no special configuration required.
