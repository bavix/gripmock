# **Stub API. Stubs Upsert**  
**Upsert** (insert or update) stubs via the `/api/stubs` endpoint. This allows adding new stubs or updating existing ones by specifying their `id`.  

## **Example Contract (`simple.proto`)**  
```proto
syntax = "proto3";

package simple;

service Gripmock {
  rpc SayHello (Request) returns (Reply);
}

message Request {
  string name = 1;
}

message Reply {
  string message = 1;
  int32 returnCode = 2;
}
```

## **Request**  
- **Method**: `POST`  
- **URL**: `/api/stubs`  
- **Headers**: `Content-Type: application/json`  
- **Body**: A single `Stub` object **or** an array of `Stub` objects.  

## **Examples**  

**1. Create a Single Stub**  
```bash
curl -X POST -d '{
  "service": "Gripmock",
  "method": "SayHello",
  "input": {
    "equals": { "name": "gripmock" }
  },
  "output": {
    "data": { "message": "Hello GripMock", "returnCode": 42 },
    "error": ""
  }
}' http://127.0.0.1:4771/api/stubs
```

**Response** (returns the generated `id`):  
```json
["6c85b0fa-caaf-4640-a672-f56b7dd8074d"]
```

---

**2. Create Multiple Stubs (Batch)**  
```bash
curl -X POST -d '[
  {
    "service": "Gripmock",
    "method": "SayHello",
    "input": { "equals": { "name": "gripmock1" } },
    "output": { "data": { "message": "Hello GripMock. stab1", "returnCode": 42 } }
  },
  {
    "service": "Gripmock",
    "method": "SayHello",
    "input": { "equals": { "name": "gripmock2" } },
    "output": { "data": { "message": "Hello GripMock. stab2", "returnCode": 42 } }
  }
]' http://127.0.0.1:4771/api/stubs
```

**Response** (returns generated `id`s for all stubs):  
```json
["2378ccb8-f36e-48b0-a257-4309876bed47", "0ee02a07-4cae-4a0b-b0c1-5e7c379bc858"]
```

## **Schema Details**  
- **Stub Object**:  
  ```json
  {
    "id": "string (optional, UUID format)",
    "service": "string (required, e.g., 'Gripmock')",
    "method": "string (required, e.g., 'SayHello')",
    "input": {
      "ignoreArrayOrder": "boolean (default: false)",
      "equals": { "key": "exact match value" },
      "contains": { "key": "partial match value" },
      "matches": { "key": "regex pattern" }
    },
    "output": {
      "data": { "key": "value (matches your protobuf Reply type)" },
      "error": "string (gRPC error message)",
      "code": "integer (gRPC status code, e.g., 3 for 'InvalidArgument')"
    }
  }
  ```

## **Behavior**  
- **ID Handling**:  
  - If `id` is omitted, a new UUID is generated.  
  - If `id` is provided and exists, the stub is **updated**.  
- **Input Matching**:  
  - `equals`: Exact match for fields.  
  - `contains`: Partial match (substring).  
  - `matches`: Regex match.  
- **Output**:  
  - `data`: Must align with your protobuf `Reply` message structure.  
  - `error` and `code`: Define gRPC error responses.  

## **Notes**  
- **Upsert Logic**: Use this endpoint to **create or update** stubs.  
- **Batch Support**: Send an array of stubs in a single request.  
- **Validation**: Invalid stubs (e.g., missing `service`/`method`) return `400 Bad Request`.  

## **Related Endpoints**  
- `GET /api/stubs`: List all stubs.  
- `DELETE /api/stubs`: Delete all stubs.  
- `DELETE /api/stubs/{uuid}`: Delete a specific stub by ID.  

This endpoint is critical for dynamically managing stubs during testing.
