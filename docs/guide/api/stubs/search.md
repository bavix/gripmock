# **Stub API. Stubs Search**  
The `/api/stubs/search` endpoint allows flexible searching of stubs based on input criteria, headers, service, method, or ID. It returns the **first matching stub's output** for testing gRPC services.  

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
- **URL**: `/api/stubs/search`  
- **Headers**: `Content-Type: application/json`  
- **Body**:  
  ```json
  {
    "id": "string (optional, UUID)",
    "service": "string (required)",
    "method": "string (required)",
    "headers": { "equals|contains|matches": { ... } },
    "data": { ... }
  }
  ```

## **Response**  
- **Status Code**: `200 OK`  
- **Content-Type**: `application/json`  
- **Body**:  
  ```json
  {
    "data": { ... },    // Matches your protobuf `Reply` structure
    "error": "string",  // gRPC error message (if applicable)
    "code": 0,          // gRPC status code (e.g., `0` for OK)
    "headers": { ... }  // Response headers (if defined)
  }
  ```

## **Input Matching Rules**  
Stubs are matched based on **input criteria**. Rules are evaluated in order: `equals` → `contains` → `matches`.  

### **1. `equals` (Exact Match)**  
Matches fields **exactly** (case-sensitive).  
**Example Stub**:  
```json
{
  "input": {
    "equals": {
      "name": "gripmock",
      "details": { "code": 42 }
    }
  }
}
```

### **2. `contains` (Partial Match)**
For example, you need a minimum of keys from those passed in the request, and not all of them.
**Example Stub**:
```json
{
  "input": {
    "contains": {
      "name": "gripmock",
      "details": { "code": 42 }
    }
  }
}
```

**Example Request**:
```json
{
  "name": "gripmock",
  "details": { "code": 42 },
  "tags": ["grpc", "mock"]
}
```

The above stub will match if the request contains **both** `name` and `details.code`.

**Note**: This is different from `equals` in that it checks for **partial** matches.

### **3. `matches` (Regular Expression)**  
Matches if the input **contains the specified fields**.  
**Example Stub**:  
```json
{
  "input": {
    "contains": {
      "address": { "city": ".*" }
    }
  }
}
```

Uses regex patterns for matching.  
**Example Stub**:  
```json
{
  "input": {
    "matches": {
      "name": "^grip.*$",
      "cities": ["Jakarta", ".*grad$"]
    }
  }
}
```

### **`ignoreArrayOrder` Flag**  
Disable array order checks:  
```json
{
  "input": {
    "ignoreArrayOrder": true,
    "equals": {
      "ids": ["id1", "id2"] // Order doesn't matter
    }
  }
}
```

## **Headers Matching Rules**  
Headers are matched similarly to input:  

### **1. `equals` (Exact Header Match)**  
```json
{
  "headers": {
    "equals": {
      "authorization": "Bearer token123"
    }
  }
}
```

### **2. `contains` (Header Presence)**  

**Example Stub**:
```json
{
  "headers": {
    "contains": {
      "authorization": "Bearer token123",
      "user-agent": "curl/7.64.1"
    }
  }
}
```

**Example Request**:
```json
{
  "headers": {
    "authorization": "Bearer token123",
    "user-agent": "curl/7.64.1",
    "x-api-key": "abc123",
    "x-foo": "bar"
  }
}
```

The above stub will match if the request contains **both** `authorization` and `user-agent`.

**Note**: This is different from `equals` in that it checks for **partial** matches.

### **3. `matches` (Header Regex)**  
```json
{
  "headers": {
    "matches": {
      "x-api-key": ".+" // any value
    }
  }
}
```

```json
{
  "headers": {
    "matches": {
      "user-agent": "^Mozilla.*$"
    }
  }
}
```

## **Examples**  

**1. Search by Data**  
```bash
curl -X POST -d '{
  "service": "Gripmock",
  "method": "SayHello",
  "data": { "name": "gripmock" }
}' http://127.0.0.1:4771/api/stubs/search
```

**Response**:  
```json
{
  "data": { "message": "Hello GripMock", "returnCode": 42 },
  "error": "",
  "code": 0
}
```

**2. Search by ID**  
```bash
curl -X POST -d '{
  "id": "6c85b0fa-caaf-4640-a672-f56b7dd8074d",
  "service": "Gripmock",
  "method": "SayHello"
}' http://127.0.0.1:4771/api/stubs/search
```

## **Behavior**  
- **ID Priority**: If `id` is provided, it takes precedence (ignores other fields).  
- **Stub Priority**: When multiple stubs match, higher `priority` values are selected first.
- **First Match**: Returns the **first stub** that matches the criteria.  
- **No Match**: Returns `error` with code `5` (Not Found) if no stub matches.  

## **Notes**  
- **Edge Cases**:  
  - If multiple stubs match, the **first created** stub is returned.  
  - Use `ignoreArrayOrder` to ignore array element order in `equals`.  
- **Related Endpoints**:  
  - `GET /api/stubs/used`: Track stubs matched by this endpoint.  
  - `POST /api/stubs`: Create/update stubs for testing.  

## **Schema References**
For complete schema details, see:
- [OpenAPI Stub Definition](https://bavix.github.io/gripmock-openapi/)
- [JSON Schema for Stubs](https://bavix.github.io/gripmock/schema/stub.json)

---

This endpoint is essential for validating stub behavior during testing. Use it to simulate gRPC responses dynamically.
