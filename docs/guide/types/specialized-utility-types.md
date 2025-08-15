# Specialized Utility Types in Protocol Buffers <VersionTag version="v2.7.1" />

Specialized utility types in Protobuf address specific API patterns, such as partial updates or empty responses. This documentation covers **`FieldMask`** and **`google.protobuf.Empty`** with examples, usage guidelines, and best practices.

## 1. `google.protobuf.FieldMask`
Used to specify subsets of fields to update or retrieve (e.g., for PATCH operations).

### Syntax
```proto
import "google/protobuf/field_mask.proto";

message UpdateRequest {
  string resource_id = 1;
  google.protobuf.FieldMask update_mask = 2;
}
```

### Key Features
- **Paths**: A list of field paths (e.g., `"name"`, `"settings.theme"`).
- **JSON Format**: Serialized as a comma-separated string (e.g., `"name,settings.theme"`).

### Example: Partial Resource Update
**Proto File (`specialized_fieldmask.proto`):**
```proto
syntax = "proto3";

import "google/protobuf/field_mask.proto";

package specialized;

service ResourceService {
  rpc UpdateResource(UpdateRequest) returns (UpdateResponse) {}
}

message UpdateRequest {
  string resourceId = 1;
  google.protobuf.FieldMask updateMask = 2;
}

message UpdateResponse {
  bool success = 1;
}
```

**Stub Configuration (`specialized_fieldmask.yaml`):**
```yaml
- service: ResourceService
  method: UpdateResource
  input:
    equals:
      resourceId: "resource_123"
      updateMask:
        paths: ["name", "description"]
  output:
    data:
      success: true
```

**Test Command:**
```sh
grpcurl -plaintext -d '{
  "resourceId": "resource_123",
  "updateMask": {"paths": ["name", "description"]}
}' localhost:4770 specialized.ResourceService/UpdateResource
```

**Output:**
```json
{
  "success": true
}
```

## 2. `google.protobuf.Empty`
Represents an empty message (used for requests/responses with no data).

### Syntax
```proto
import "google/protobuf/empty.proto";

service HealthService {
  rpc CheckHealth(google.protobuf.Empty) returns (google.protobuf.Empty) {}
}
```

### Key Features
- **Zero Fields**: Cannot contain any fields.
- **Use Cases**: Health checks, delete operations, or notifications.

### Example: Health Check
**Proto File (`specialized_empty.proto`):**
```proto
syntax = "proto3";

import "google/protobuf/empty.proto";

package specialized;

service HealthService {
  rpc CheckHealth(google.protobuf.Empty) returns (google.protobuf.Empty) {}
}
```

**Stub Configuration (`specialized_empty.yaml`):**
```yaml
- service: HealthService
  method: CheckHealth
  input:
    equals: {}
  output:
    data: {}
```

**Test Command:**
```sh
grpcurl -plaintext localhost:4770 specialized.HealthService/CheckHealth
```

**Output:**
```json
{}
```

## Best Practices
1. **FieldMask**:
   - Use for partial updates (e.g., `PATCH /users/{id}`).
   - Validate paths against the resource schema to avoid errors.
   - Avoid using `FieldMask` for full updates; use standard messages instead.

2. **Empty**:
   - Use for fire-and-forget operations (e.g., logging, notifications).
   - Prefer `Empty` over custom empty messages for consistency.

## Common Pitfalls
- **FieldMask Paths**: Invalid paths (e.g., typos) may silently ignore fields.
- **Empty Misuse**: Avoid using `Empty` for operations that return meaningful data.
- **JSON Serialization**: Ensure `FieldMask` paths are comma-separated in JSON (e.g., `"name,description"`).

## Further Reading
- [FieldMask Documentation](https://protobuf.dev/reference/protobuf/google.protobuf/#fieldmask)
- [Empty Type Definition](https://protobuf.dev/reference/protobuf/google.protobuf/#empty)
- [API Design Patterns](https://cloud.google.com/apis/design/design_patterns)
