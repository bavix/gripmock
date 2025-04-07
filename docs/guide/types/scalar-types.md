# Scalar Types in Protocol Buffers

Scalar types in Protobuf represent primitive values such as numbers, strings, and booleans. They form the building blocks of more complex data structures. This documentation covers **all scalar types** with examples, usage guidelines, and best practices.

## 1. Integer Types
### **`int32`**
- **Description**: 32-bit signed integer (varint encoding).
- **Range**: -2^31 to 2^31 - 1.
- **Use Case**: General-purpose integer values (e.g., counters, IDs).

**Proto File (`scalar_int32.proto`):**
```proto
syntax = "proto3";

package scalar;

service MathService {
  rpc AddInt32(Int32Request) returns (Int32Response) {}
}

message Int32Request {
  int32 a = 1;
  int32 b = 2;
}

message Int32Response {
  int32 result = 1;
}
```

**Stub Configuration (`scalar_int32.yaml`):**
```yaml
- service: MathService
  method: AddInt32
  input:
    equals:
      a: 5
      b: 10
  output:
    data:
      result: 15
```

**Test Command:**
```sh
grpcurl -plaintext -d '{"a": 5, "b": 10}' localhost:4770 scalar.MathService/AddInt32
```

**Output:**
```json
{
  "result": 15
}
```

### **`uint32`**
- **Description**: 32-bit unsigned integer (varint encoding).
- **Range**: 0 to 2^32 - 1.
- **Use Case**: Non-negative values (e.g., counts, sizes).

**Example:**
```proto
message ImageRequest {
  uint32 width = 1;
  uint32 height = 2;
}
```

### **`sint32`**
- **Description**: 32-bit signed integer with zigzag encoding.
- **Range**: -2^31 to 2^31 - 1.
- **Use Case**: Optimized for negative numbers (e.g., temperature, deltas).

### **`fixed32`**
- **Description**: 32-bit fixed-width integer (always 4 bytes).
- **Use Case**: High-performance scenarios (e.g., binary protocols).

## 2. Floating-Point Types
### **`float`**
- **Description**: 32-bit floating-point number.
- **Precision**: ~7 decimal digits.
- **Use Case**: Scientific calculations, approximate values.

**Proto File (`scalar_float.proto`):**
```proto
syntax = "proto3";

package scalar;

service CalculatorService {
  rpc MultiplyFloat(FloatRequest) returns (FloatResponse) {}
}

message FloatRequest {
  float a = 1;
  float b = 2;
}

message FloatResponse {
  float result = 1;
}
```

**Stub Configuration (`scalar_float.yaml`):**
```yaml
- service: CalculatorService
  method: MultiplyFloat
  input:
    equals:
      a: 3.5
      b: 2
  output:
    data:
      result: 7.0
```

**Test Command:**
```sh
grpcurl -plaintext -d '{"a": 3.5, "b": 2.0}' localhost:4770 scalar.CalculatorService/MultiplyFloat
```

**Output:**
```json
{
  "result": 7
}
```

### **`double`**
- **Description**: 64-bit floating-point number.
- **Precision**: ~15 decimal digits.
- **Use Case**: High-precision calculations (e.g., financial data).

## 3. Boolean Type
### **`bool`**
- **Description**: Boolean value (`true`/`false`).
- **JSON Mapping**: Serialized as `true` or `false`.

**Proto File (`scalar_bool.proto`):**
```proto
syntax = "proto3";

package scalar;

service AuthService {
  rpc IsAdmin(AdminRequest) returns (AdminResponse) {}
}

message AdminRequest {
  string username = 1;
}

message AdminResponse {
  bool isAdmin = 1;
}
```

**Stub Configuration (`scalar_bool.yaml`):**
```yaml
- service: AuthService
  method: IsAdmin
  input:
    equals:
      username: "admin_user"
  output:
    data:
      isAdmin: true
```

**Test Command:**
```sh
grpcurl -plaintext -d '{"username": "admin_user"}' localhost:4770 scalar.AuthService/IsAdmin
```

**Output:**
```json
{
  "isAdmin": true
}
```

## 4. String Type
### **`string`**
- **Description**: UTF-8 encoded text.
- **JSON Mapping**: Serialized as a JSON string.

**Proto File (`scalar_string.proto`):**
```proto
syntax = "proto3";

package scalar;

service GreetingService {
  rpc Greet(StringRequest) returns (StringResponse) {}
}

message StringRequest {
  string name = 1;
}

message StringResponse {
  string message = 1;
}
```

**Stub Configuration (`scalar_string.yaml`):**
```yaml
- service: GreetingService
  method: Greet
  input:
    equals:
      name: "Alice"
  output:
    data:
      message: "Hello, Alice!"
```

**Test Command:**
```sh
grpcurl -plaintext -d '{"name": "Alice"}' localhost:4770 scalar.GreetingService/Greet
```

**Output:**
```json
{
  "message": "Hello, Alice!"
}
```

## 5. Bytes Type
### **`bytes`**
- **Description**: Arbitrary binary data.
- **JSON Mapping**: Base64-encoded string.

**Proto File (`scalar_bytes.proto`):**
```proto
syntax = "proto3";

package scalar;

service FileService {
  rpc UploadFile(BytesRequest) returns (BytesResponse) {}
}

message BytesRequest {
  bytes content = 1;
}

message BytesResponse {
  string checksum = 1;
}
```

**Stub Configuration (`scalar_bytes.yaml`):**
```yaml
- service: FileService
  method: UploadFile
  input:
    equals:
      content: "aGVsbG8="  # "hello" in Base64
  output:
    data:
      checksum: "5d41402abc4b2a76b9719d911017c592"  # MD5 hash of "hello"
```

**Test Command:**
```sh
grpcurl -plaintext -d '{"content": "aGVsbG8="}' localhost:4770 scalar.FileService/UploadFile
```

**Output:**
```json
{
  "checksum": "5d41402abc4b2a76b9719d911017c592"
}
```

## 6. Specialized Integer Types
### **`sint64`, `uint64`, `fixed64`, `sfixed64`**
- **Description**: 64-bit variants of integer types.
- **Use Case**: Large numbers (e.g., timestamps, file sizes).

## Best Practices
1. **Precision**: Use `double` for financial calculations to avoid rounding errors.
2. **Encoding**: Prefer `sint32`/`sint64` for fields with frequent negative values.
3. **Strings**: Validate UTF-8 encoding for `string` fields.
4. **Bytes**: Document the format of binary data (e.g., images, serialized objects).

## Common Pitfalls
- **Integer Overflow**: Ensure values fit within the type’s range (e.g., `uint32` cannot be negative).
- **Floating-Point Accuracy**: Avoid equality checks with `float`/`double` due to precision loss.
- **Base64 Padding**: Ensure `bytes` fields are properly padded in JSON (e.g., `aGVsbG8` → `aGVsbG8=`).

## Further Reading
- [Protobuf Scalar Types Reference](https://protobuf.dev/programming-guides/proto3/#scalar)
- [GRPC Data Types](https://grpc.io/docs/guides/)
