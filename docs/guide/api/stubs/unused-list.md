### **Stub API. Get Stubs Unused List**  
The `/api/stubs/unused` endpoint retrieves a list of **stubs that have never been matched by a search operation**. This helps identify "dead" stubs that are defined but not actively used in testing workflows.  

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
- **URL**: `/api/stubs/unused`  
- **Parameters**: None required.  
- **Headers**: Standard headers (e.g., `Content-Type: application/json`).  

**Example Request**:  
```bash
curl http://127.0.0.1:4771/api/stubs/unused
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
| `input`  | `object` | Input matching criteria (e.g., `equals`, `contains`, `matches`).          |
| `output` | `object` | Response configuration, including `data`, `error`, and gRPC status `code`.|  

#### **Behavior Explanation**  
- **Unused Definition**: A stub is considered "unused" if it has **never** been matched by a `POST /api/stubs/search` request.  
- **Inverse of Used**: The `/api/stubs/used` endpoint returns stubs that **have** been matched by searches.  
- **Persistence**: The "unused" state resets when the GripMock server restarts.  

#### **Example Workflow**  
1. **Create a Stub**:  
   ```bash
   curl -X POST -d '{
     "service": "Gripmock",
     "method": "SayHello",
     "input": { "equals": { "name": "gripmock" } },
     "output": { "data": { "message": "Hello GripMock", "returnCode": 42 } }
   }' http://127.0.0.1:4771/api/stubs
   ```

2. **Check Unused Stubs** (stub is unused):  
   ```bash
   curl http://127.0.0.1:4771/api/stubs/unused
   ```
   **Response**:  
   ```json
   [{"id": "...", ...}]
   ```

3. **Search for the Stub** (marks it as used):  
   ```bash
   curl -X POST -d '{
     "service": "Gripmock",
     "method": "SayHello",
     "data": { "name": "gripmock" }
   }' http://127.0.0.1:4771/api/stubs/search
   ```

4. **Check Unused Stubs Again** (stub is now used):  
   ```bash
   curl http://127.0.0.1:4771/api/stubs/unused
   ```
   **Response**:  
   ```json
   []
   ```

#### **Notes**  
- **Edge Cases**:  
  - If no stubs exist, the response is an empty array (`[]`).  
  - Stubs added but never searched for will always appear in the unused list.  
- **Related Endpoints**:  
  - `GET /api/stubs/used`: List stubs that have been matched by searches.  
  - `POST /api/stubs/search`: Mark stubs as used by matching criteria.  
  - `POST /api/stubs`: Create or update stubs.  

---

This endpoint is essential for maintaining clean stub configurations by identifying and removing unused stubs.
