### **Stub API. Get Stubs List**  
The `/api/stubs` endpoint retrieves a list of **all registered stubs**, regardless of whether they have been used or not. This is useful for debugging and auditing stub configurations.  

#### **Example Contract (`simple.proto`)**  
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

#### **Request**  
- **Method**: `GET`  
- **URL**: `/api/stubs`  
- **Parameters**: None required.  
- **Headers**: Standard headers (e.g., `Content-Type: application/json`).  

**Example Request**:  
```bash
curl http://127.0.0.1:4771/api/stubs
```

#### **Response**  
- **Status Code**: `200 OK`  
- **Content-Type**: `application/json`  
- **Body**: An array of `Stub` objects (see schema below).  

**Example Response**:  
```json
[
  {
    "id": "6c85b0fa-caaf-4640-a672-f56b7dd8074d",
    "service": "Gripmock",
    "method": "SayHello",
    "input": {
      "equals": { "name": "gripmock" }
    },
    "output": {
      "data": { "message": "Hello GripMock", "returnCode": 42 },
      "error": ""
    }
  }
]
```

#### **Stub Object Schema**  
| Field    | Type     | Description                                                                 |
|----------|----------|-----------------------------------------------------------------------------|
| `id`     | `string` | Unique identifier for the stub (UUID format).                              |
| `service`| `string` | Name of the gRPC service (e.g., `Gripmock`).                              |
| `method` | `string` | Name of the gRPC method (e.g., `SayHello`).                               |
| `headers`| `object` | Header matching rules (`equals`, `contains`, `matches`).                  |
| `input`  | `object` | Input matching criteria (`equals`, `contains`, `matches`, `ignoreArrayOrder`). |
| `output` | `object` | Response configuration, including `data`, `error`, and gRPC status `code`.|  

#### **Behavior**  
- **Comprehensive List**: Returns **all stubs**, including both used and unused ones.  
- **Order**: The order of stubs is not guaranteed.  
- **No Side Effects**: Fetching the list does **not** mark stubs as "used".  

#### **Example Workflow**  
1. **Create Stubs**:  
   ```bash
   curl -X POST -d '[
     {
       "service": "Gripmock",
       "method": "SayHello",
       "input": { "equals": { "name": "gripmock1" } }
     },
     {
       "service": "Gripmock",
       "method": "SayHello",
       "input": { "equals": { "name": "gripmock2" } }
     }
   ]' http://127.0.0.1:4771/api/stubs
   ```

2. **List All Stubs**:  
   ```bash
   curl http://127.0.0.1:4771/api/stubs
   ```
   **Response**:  
   ```json
   [
     { "id": "2378ccb8-f36e-48b0-a257-4309876bed47", ... },
     { "id": "0ee02a07-4cae-4a0b-b0c1-5e7c379bc858", ... }
   ]
   ```

#### **Notes**  
- **Edge Cases**:  
  - If no stubs exist, returns an empty array (`[]`).  
  - Includes stubs created via `POST /api/stubs` or static configurations.  
- **Related Endpoints**:  
  - `GET /api/stubs/used`: List stubs matched by searches.  
  - `GET /api/stubs/unused`: List stubs never matched by searches.  
  - `DELETE /api/stubs`: Purge all stubs.  

---

This endpoint is essential for debugging and verifying stub configurations during test setup.
