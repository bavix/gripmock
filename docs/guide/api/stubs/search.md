# Stub API: Search Stubs
The `/api/stubs/search` endpoint resolves a request against the loaded stubs — by input criteria, headers, service, method or ID — and returns the winning stub's output. It answers "which stub would this call hit?" without making the gRPC call.  

## Example Contract (`simple.proto`)
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

## Request
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

## Response
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

## Input Matching Rules

A stub matches when **every** strategy it declares passes — `equals`, `contains`,
`matches` and `glob` are AND-ed, with no ordering between them. An omitted
strategy always passes. See [Matching Logic](../../matcher/logic) for the formal
rules.

### 1. `equals` (Exact Match)
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

### 2. `contains` (Partial Match)
Matches when the request carries at least the listed keys and values. Extra keys
in the request are ignored; string values match on substring, arrays on
containment.
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

### 3. `matches` (Regular Expression)
Matches fields using **regular expressions**.  
**Example Stub**:  
```json
{
  "input": {
    "matches": {
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

### `ignoreArrayOrder` Flag
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

## Headers Matching Rules
Headers are matched similarly to input:  

### 1. `equals` (Exact Header Match)
```json
{
  "headers": {
    "equals": {
      "authorization": "Bearer token123"
    }
  }
}
```

### 2. `contains` (Header Presence)

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

### 3. `matches` (Header Regex)
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

## Examples

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

**Note**: When searching by ID, the `id` field is used for exact ID matching, but `service` and `method` fields are still required as they are mandatory in the SearchRequest structure.

## Behavior
- **Selection**: among matching stubs, the most specific one wins; `priority` breaks ties, then declared field count, then stub ID. See [Priority](../../stubs/priority).
- **No Match**: Returns `error` with code `5` (Not Found) if no stub matches.  

## Notes
- **Edge Cases**:  
  - A fully tied pair resolves by stub ID, so the outcome is stable across runs but not tied to creation order.  
  - Use `ignoreArrayOrder` to ignore array element order in `equals`.  
- **Related Endpoints**:  
  - `GET /api/stubs/used`: Track stubs matched by this endpoint.  
  - `POST /api/stubs`: Create/update stubs for testing.  

## Schema References
For complete schema details, see:
- [OpenAPI Stub Definition](https://bavix.github.io/gripmock-openapi/)
- [JSON Schema for Stubs](https://bavix.github.io/gripmock/schema/stub.json)
