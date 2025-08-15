---
title: Well-known Types
---

# Well-Known Types (`google.protobuf.*`) in Protocol Buffers <VersionTag version="v2.7.1" />

Well-known types are predefined Protobuf types that provide common utility functionality. They are part of the `google.protobuf` package and are automatically included in most Protobuf implementations. This documentation covers **all major well-known types** with examples, usage guidelines, and best practices.

## 1. `google.protobuf.Timestamp`
Represents a point in time with nanosecond precision.

### Syntax
```proto
import "google/protobuf/timestamp.proto";

message EventResponse {
  google.protobuf.Timestamp event_time = 1;
}
```

### Key Features
- **JSON Format**: Serialized as an RFC 3339 string (e.g., `"2024-01-01T12:00:00Z"`).
- **Conversion**: Automatically handled by Protobuf libraries.

### Example: Event Timestamp
**Proto File (`wkt_timestamp.proto`):**
```proto
syntax = "proto3";

import "google/protobuf/timestamp.proto";

package wkt;

service EventService {
  rpc GetEventTime(EventRequest) returns (EventResponse) {}
}

message EventRequest {
  string eventId = 1;
}

message EventResponse {
  google.protobuf.Timestamp eventTime = 1;
}
```

**Stub Configuration (`wkt_timestamp.yaml`):**
```yaml
- service: EventService
  method: GetEventTime
  input:
    equals:
      eventId: "event_123"
  output:
    data:
      eventTime: "2024-01-01T12:00:00Z"
```

**Test Command:**
```sh
grpcurl -plaintext -d '{"eventId": "event_123"}' localhost:4770 wkt.EventService/GetEventTime
```

**Output:**
```json
{
  "eventTime": "2024-01-01T12:00:00Z"
}
```

## 2. `google.protobuf.Duration`
Represents a time interval with nanosecond precision.

### Syntax
```proto
import "google/protobuf/duration.proto";

message TaskResponse {
  google.protobuf.Duration timeTaken = 1;
}
```

### Key Features
- **JSON Format**: Serialized as a string (e.g., `"330s"`).
- **Range**: -315,576,000,000s to +315,576,000,000s.

### Example: Task Duration
**Proto File (`wkt_duration.proto`):**
```proto
syntax = "proto3";

import "google/protobuf/duration.proto";

package wkt;

service TaskService {
  rpc GetDuration(TaskRequest) returns (TaskResponse) {}
}

message TaskRequest {
  string taskId = 1;
}

message TaskResponse {
  google.protobuf.Duration timeTaken = 1;
}
```

**Stub Configuration (`wkt_duration.yaml`):**
```yaml
- service: TaskService
  method: GetDuration
  input:
    equals:
      taskId: "task_123"
  output:
    data:
      timeTaken: "330s"  # 5 minutes 30 seconds
```

**Test Command:**
```sh
grpcurl -plaintext -d '{"taskId": "task_123"}' localhost:4770 wkt.TaskService/GetDuration
```

**Output:**
```json
{
  "timeTaken": "330s"
}
```

## 3. `google.protobuf.Any`
A container for arbitrary serialized Protobuf messages.

### Syntax
```proto
import "google/protobuf/any.proto";

message DataRequest {
  google.protobuf.Any payload = 1;
}
```

### Key Features
- **Type URL**: Identifies the embedded message type (e.g., `type.googleapis.com/google.protobuf.StringValue`).
- **Dynamic Parsing**: Requires runtime type resolution.

### Example: Generic Data Storage
**Proto File (`wkt_any.proto`):**
```proto
syntax = "proto3";

import "google/protobuf/any.proto";

package wkt;

service DataService {
  rpc StoreData(DataRequest) returns (DataResponse) {}
}

message DataRequest {
  google.protobuf.Any payload = 1;
}

message DataResponse {
  bool success = 1;
}
```

**Stub Configuration (`wkt_any.yaml`):**
```yaml
- service: DataService
  method: StoreData
  input:
    equals:
      payload:
        type_url: "type.googleapis.com/google.protobuf.StringValue"
        value: "CgR0ZXN0" # "test" in base64
  output:
    data:
      success: true
```

**Test Command:**
```sh
grpcurl -plaintext -d '{
  "payload": {
    "@type": "type.googleapis.com/google.protobuf.StringValue",
    "value": "test"
  }
}' localhost:4770 wkt.DataService/StoreData
```

**Output:**
```json
{
  "success": true
}
```

## 4. `google.protobuf.Struct`, `Value`, and `ListValue`
Dynamic key-value structures for unstructured data.

### Syntax
```proto
import "google/protobuf/struct.proto";

message DynamicResponse {
  google.protobuf.Struct data = 1;
}
```

### Example: Flexible Configuration
**Proto File (`wkt_struct.proto`):**
```proto
syntax = "proto3";

import "google/protobuf/struct.proto";

package wkt;

service ConfigService {
  rpc GetConfig(ConfigRequest) returns (ConfigResponse) {}
}

message ConfigRequest {
  string configId = 1;
}

message ConfigResponse {
  google.protobuf.Struct settings = 1;
}
```

**Stub Configuration (`wkt_struct.yaml`):**
```yaml
- service: ConfigService
  method: GetConfig
  input:
    equals:
      configId: "config_123"
  output:
    data:
      settings:
        fields:
          theme:
            stringValue: "dark"
          max_users:
            numberValue: 100
          enabled:
            boolValue: true
```

**Test Command:**
```sh
grpcurl -plaintext -d '{"configId": "config_123"}' localhost:4770 wkt.ConfigService/GetConfig
```

**Output:**
```json
{
  "settings": {
    "theme": "dark",
    "max_users": 100,
    "enabled": true
  }
}
```

## 5. Wrapper Types (`StringValue`, `Int32Value`, etc.)
Optional scalar types for distinguishing `null` from default values.

### Syntax
```proto
import "google/protobuf/wrappers.proto";

message UserResponse {
  google.protobuf.StringValue nickname = 1;
}
```

### Example: Optional User Profile
**Proto File (`wkt_wrappers.proto`):**
```proto
syntax = "proto3";

import "google/protobuf/wrappers.proto";

package wkt;

service UserService {
  rpc GetUser(UserRequest) returns (UserResponse) {}
}

message UserRequest {
  string userId = 1;
}

message UserResponse {
  google.protobuf.StringValue nickname = 1;
  google.protobuf.Int32Value age = 2;
}
```

**Stub Configuration (`wkt_wrappers.yaml`):**
```yaml
- service: UserService
  method: GetUser
  input:
    equals:
      userId: "user_123"
  output:
    data:
      nickname: "alice"
      age: null # age is not set
```

**Test Command:**
```sh
grpcurl -plaintext -d '{"userId": "user_123"}' localhost:4770 wkt.UserService/GetUser
```

**Output:**
```json
{
  "nickname": "alice",
  "age": null
}
```

## 6. `google.type.Date`
Represents a calendar date (year, month, day) without time or timezone information.

### Syntax
```proto
import "google/type/date.proto";

message WeatherReport {
  google.type.Date observation_date = 1;
}
```

### Key Features
- **JSON Format**: Serialized as `"YYYY-MM-DD"` string (e.g., `"2023-10-05"`).
- **Validation**: Automatically checks for valid dates (no February 30th).
- **Common Uses**: Birthdays, event scheduling, historical records.

### Example: Weather Forecast Service
Demonstrates gRPC service with HTTP endpoint mapping via grpc-gateway.

**Proto File (`wkt_weather.proto`):**
```proto
syntax = "proto3";

package weather;

import "google/api/annotations.proto";
import "google/protobuf/empty.proto";
import "google/type/date.proto";

option go_package = "github.com/bavix/gripmock/example/weather;weather";

service WeatherService {
  rpc GetCurrentForecast(google.protobuf.Empty) returns (WeatherReport) {
    option (google.api.http) = {
      get: "/v1/weather/current"  // REST endpoint mapping
    };
  }
}

message WeatherReport {
  google.type.Date date = 1;
  string condition = 2;
  double temperature_c = 3;
  double precipitation_mm = 4;
}
```

**Stub Configuration (`wkt_weather.yaml`):**
```yaml
- service: WeatherService
  method: GetCurrentForecast
  input:
    equals: {}
  output:
    data:
      date: {year: 2023, month: 10, day: 5}
      condition: "Sunny"
      temperature_c: 22.5
      precipitation_mm: 0.0
```

**Test Command:**
```sh
grpcurl -plaintext localhost:4770 weather.WeatherService/GetCurrentForecast
```

**Output:**
```json
{
  "date": {
    "year": 2023,
    "month": 10,
    "day": 5
  },
  "condition": "Sunny",
  "temperatureC": 22.5
}
```

**grpc-gateway Integration:**
```sh
curl http://localhost:8080/v1/weather/current
```
```json
{
  "date": "2023-10-05",
  "condition": "Sunny",
  "temperatureC": 22.5,
  "precipitationMm": 0.0
}
```

## Best Practices
1. **Timestamp/Duration**:
   - Always use UTC for `Timestamp`.
   - Validate durations (e.g., negative values may be invalid).
2. **Any**:
   - Use `type_url` with full Protobuf type names.
   - Avoid `Any` for frequently accessed data (parsing overhead).
3. **Struct**:
   - Prefer strongly-typed messages for performance-critical APIs.
   - Use `Struct` for dynamic or rapidly changing data.
4. **Wrappers**:
   - Use for optional fields where `null` has semantic meaning.
5. **Date**:
   - Combine with `Timestamp` when exact time matters
   - Use for date-specific operations (e.g., birthday checks)
   - Store timezone information separately when needed

## Common Pitfalls
- **Timestamp Parsing**: Invalid RFC 3339 strings cause errors.
- **Duration Ranges**: Values outside ±10,000 years are rejected.
- **Any Type Safety**: Incorrect `type_url` leads to deserialization failures.
- **Wrapper Defaults**: `null` vs. `0`/`""` distinctions must be documented.
- **Date Validity**: Invalid dates like `2023-02-30` will cause errors
- **Time Zone Assumptions**: Always document expected timezone context

## Further Reading
- [Well-Known Types Reference](https://protobuf.dev/reference/protobuf/google.protobuf/)
- [Protobuf Struct Documentation](https://protobuf.dev/reference/protobuf/google.protobuf/#struct)
- [API Design Best Practices](https://cloud.google.com/apis/design/)
- [grpc-gateway Documentation](https://grpc-ecosystem.github.io/grpc-gateway/)
