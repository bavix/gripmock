# Composite and Collection Types in Protocol Buffers <VersionTag version="v1.13.0" />

Composite and collection types in Protocol Buffers (Protobuf) allow you to define complex data structures beyond primitive scalar values. These types are essential for modeling hierarchical or nested data in APIs. This documentation covers **enums**, **repeated fields**, and **maps** with examples, usage guidelines, and best practices.

## 1. Enum Types
Enums define a set of named integer values. They are useful for representing states, options, or categories.

### Syntax
```proto
enum Status {
  UNKNOWN = 0;
  ACTIVE = 1;
  INACTIVE = 2;
}
```

### Key Features
- **Default Value**: The first enum value **MUST be 0** (default in proto3).
- **JSON Mapping**: Enums are serialized as **strings** in JSON (e.g., `"ACTIVE"`), not integers.
- **Unknown Values**: If an unknown enum value is received, it is treated as `UNKNOWN` (proto3).

### Example: Enum in a Service
**Proto File (`composite_enum.proto`):**
```proto
syntax = "proto3";

package composite;

enum Status {
  UNKNOWN = 0;
  ACTIVE = 1;
  INACTIVE = 2;
}

service StatusService {
  rpc GetStatus(StatusRequest) returns (StatusResponse) {}
}

message StatusRequest {
  string userId = 1;
}

message StatusResponse {
  Status status = 1;
}
```

**Stub Configuration (`composite_enum.yaml`):**
```yaml
- service: StatusService
  method: GetStatus
  input:
    equals:
      userId: "user_123"
  output:
    data:
      status: ACTIVE  # Use the enum name, not the integer value
```

**Test Command:**
```sh
grpcurl -plaintext -d '{"userId": "user_123"}' localhost:4770 composite.StatusService/GetStatus
```

**Output:**
```json
{
  "status": "ACTIVE"
}
```

## 2. Repeated Fields
`repeated` fields represent **lists/arrays** of values (scalar, enum, or message types).

### Syntax
```proto
message ListRequest {
  repeated string items = 1;
}
```

### Key Features
- **Order Preservation**: Elements are ordered and can be duplicated.
- **Empty Lists**: Serialized as empty arrays in JSON (`[]`), not `null`.
- **Memory Efficiency**: Use `repeated` instead of `bytes` for structured data.

### Example: Repeated Strings
**Proto File (`composite_repeated.proto`):**
```proto
syntax = "proto3";

package composite;

service ListService {
  rpc ProcessList(ListRequest) returns (ListResponse) {}
}

message ListRequest {
  repeated string items = 1;
}

message ListResponse {
  repeated int32 lengths = 1;
}
```

**Stub Configuration (`composite_repeated.yaml`):**
```yaml
- service: ListService
  method: ProcessList
  input:
    equals:
      items: ["apple", "banana", "cherry"]
  output:
    data:
      lengths: [5, 6, 6]  # Lengths of each input string
```

**Test Command:**
```sh
grpcurl -plaintext -d '{"items": ["apple", "banana", "cherry"]}' localhost:4770 composite.ListService/ProcessList
```

**Output:**
```json
{
  "lengths": [5, 6, 6]
}
```

## 3. Map Types
`map` fields represent **key-value pairs** with unique keys. Keys must be scalar types (e.g., `string`, `int32`).

### Syntax
```proto
message MapRequest {
  map<string, int32> scores = 1;
}
```

### Key Features
- **Key Restrictions**: Keys cannot be `float`, `bytes`, or `enum`.
- **Order Unspecified**: Map iteration order is not guaranteed.
- **JSON Representation**: Serialized as a JSON object.

### Example: Map of Scores
**Proto File (`composite_map.proto`):**
```proto
syntax = "proto3";

package composite;

service MapService {
  rpc ProcessMap(MapRequest) returns (MapResponse) {}
}

message MapRequest {
  map<string, int32> scores = 1;
}

message MapResponse {
  map<string, bool> passed = 1;
}
```

**Stub Configuration (`composite_map.yaml`):**
```yaml
- service: MapService
  method: ProcessMap
  input:
    equals:
      scores:
        alice: 90
        bob: 75
        charlie: 50
  output:
    data:
      passed:
        alice: true
        bob: true
        charlie: false
```

**Test Command:**
```sh
grpcurl -plaintext -d '{
  "scores": {
    "alice": 90,
    "bob": 75,
    "charlie": 50
  }
}' localhost:4770 composite.MapService/ProcessMap
```

**Output:**
```json
{
  "passed": {
    "alice": true,
    "bob": true,
    "charlie": false
  }
}
```

## Best Practices
1. **Enums**:
   - Use `UNKNOWN = 0` as the default value.
   - Avoid reusing enum numbers to prevent backward compatibility issues.

2. **Repeated Fields**:
   - Prefer `repeated` over `bytes` for structured data.
   - Document whether the order of elements matters.

3. **Maps**:
   - Use `string` keys for readability in JSON.
   - Avoid maps with large numbers of entries (consider `repeated` for performance).

## Common Pitfalls
- **Enum Name Conflicts**: Ensure enum names are unique within their scope.
- **Map Key Types**: Using `float` or `bytes` as map keys will result in compilation errors.
- **Repeated Field Defaults**: Empty lists are valid; avoid `null` in JSON.

## Further Reading
- [Protobuf Language Guide: Enums](https://protobuf.dev/programming-guides/proto3/#enum)
- [Protobuf Language Guide: Maps](https://protobuf.dev/programming-guides/proto3/#maps)
- [GRPC Best Practices](https://grpc.io/docs/guides/)
