# Union-like Constructs in Protocol Buffers

Union-like constructs in Protocol Buffers (Protobuf) are implemented using the `oneof` keyword, allowing a message to contain **exactly one** of multiple possible fields. This is useful for modeling mutually exclusive data structures (similar to `union` in C or `Either` in functional languages). This section covers `oneof` syntax, use cases, and best practices with examples.

## 1. `oneof` Syntax and Semantics
The `oneof` keyword groups fields that cannot be set simultaneously. Only one field in the `oneof` block can be present in a message.

### Syntax
```proto
message MyMessage {
  oneof my_oneof {
    string field_a = 1;
    int32 field_b = 2;
    MyOtherMessage field_c = 3;
  }
}
```

### Key Features
- **Mutual Exclusivity**: Only one field in the `oneof` can be set.
- **Type Safety**: Fields in `oneof` can be of any type (scalar, message, enum).
- **Serialization**: Only the **set field** is serialized.
- **Generated Code**: Accessors return `bool` for presence checks (e.g., `has_field_a()`).

## 2. Example: Task Service with `oneof`
This example demonstrates a task creation service where notifications can be sent via email or SMS (but not both).

### Proto File (`task_service.proto`)
```proto
syntax = "proto3";

package task.v1;

service TaskService {
  rpc CreateTask(CreateTaskRequest) returns (CreateTaskResponse) {}
}

message CreateTaskRequest {
  string title = 1;
  string description = 2;
  oneof notification {
    EmailNotification email = 3;
    SmsNotification sms = 4;
  }
}

message CreateTaskResponse {
  oneof result {
    SuccessResponse success = 1;
    ErrorResponse error = 2;
  }
}

message EmailNotification {
  string email = 1;
}

message SmsNotification {
  string phone_number = 1;
}

message SuccessResponse {
  string task_id = 1;
  TaskStatus status = 2;
}

enum TaskStatus {
  TASK_STATUS_UNSPECIFIED = 0;
  TASK_CREATED = 1;
  NOTIFICATION_SENT = 2;
}

message ErrorResponse {
  ErrorCode code = 1;
  string message = 2;
}

enum ErrorCode {
  ERROR_CODE_UNSPECIFIED = 0;
  INVALID_INPUT = 1;
  NOTIFICATION_FAILED = 2;
}
```

### Stub Configuration (`union_like_constructs_oneof.yaml`):

```yaml
- service: TaskService
  method: CreateTask
  input:
    equals:
      title: Buy groceries
      description: Milk, eggs, bread
      email:
          email: user@example.com
  output:
    data:
      success:
        task_id: TASK-456
        status: TASK_CREATED

- service: TaskService
  method: CreateTask
  input:
    equals:
      title: Team meeting
      description: Project sync at 3 PM
      sms:
          phone_number: "+14155550123"
  output:
    data:
      success:
        task_id: TASK-789
        status: NOTIFICATION_SENT

- service: TaskService
  method: CreateTask
  input:
    equals:
      description: Invalid task
      sms:
        phone_number: invalid-phone
  output:
    data:
      error:
        code: INVALID_INPUT
        message: Validation failed
        errors:
            - field: title
              description: Title cannot be empty
            - field: sms.phone_number
              description: Invalid phone number format
```

## 3. Usage Examples

### Example 1: Valid Email Notification Request
**Request:**
```bash
grpcurl -plaintext \
  -d '{
    "title": "Buy groceries",
    "description": "Milk, eggs, bread",
    "email": {
      "email": "user@example.com"
    }
  }' \
  localhost:4770 task.v1.TaskService/CreateTask
```

**Response:**
```json
{
  "success": {
    "task_id": "TASK-456",
    "status": "NOTIFICATION_SENT"
  }
}
```

### Example 2: Valid SMS Notification Request
**Request:**
```bash
grpcurl -plaintext \
  -d '{
    "title": "Team meeting",
    "description": "Project sync at 3 PM",
    "sms": {
        "phone_number": "+14155550123"
    }
  }' \
  localhost:4770 task.v1.TaskService/CreateTask
```

**Response:**
```json
{
  "success": {
    "taskId": "TASK-789",
    "status": "NOTIFICATION_SENT"
  }
}
```

### Example 3: SMS Notification with Error
**Request:**
```bash
grpcurl -plaintext \
  -d '{
    "title": "",
    "description": "Invalid task",
    "sms": {
        "phone_number": "invalid-phone"
    }
  }' \
  localhost:4770 task.v1.TaskService/CreateTask
```

**Response:**
```json
{
  "error": {
    "code": "INVALID_INPUT",
    "message": "Validation failed",
    "errors": [
      {
        "field": "title",
        "description": "Title cannot be empty"
      },
      {
        "field": "sms.phone_number",
        "description": "Invalid phone number format"
      }
    ]
  }
}
```

## 4. Best Practices
- **Use for Mutual Exclusion**: Model scenarios where only one field can be valid (e.g., payment methods, notification channels).
- **Avoid Mixing with `optional`**: `oneof` fields are implicitly optional; avoid wrapping them in `optional`.
- **Document Field Semantics**: Clearly specify which fields are part of the `oneof` in comments.
- **Handle Unknown Cases**: Check for unset fields in your code (e.g., `if request.notification_case == NONE`).

## 5. Common Pitfalls
- **Multiple Fields Set**: Setting more than one `oneof` field will result in **only the last field** being serialized.
- **Default Values**: If no field is set, the `oneof` case is considered "none" (no default value).
- **Enum Conflicts**: Ensure enum values in `oneof` blocks have unique numeric identifiers.

## 6. Further Reading
- [Protobuf `oneof` Documentation](https://protobuf.dev/programming-guides/proto3/#oneof)
- [API Design Best Practices](https://cloud.google.com/apis/design/proto3)
- [GRPC Error Handling](https://grpc.io/docs/guides/error/)

This pattern is widely used in APIs for payment processing, notifications, and polymorphic data structures.
