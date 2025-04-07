### Expanded Documentation for `/api/stubs/used` Endpoint

#### **Overview**
The `/api/stubs/used` endpoint retrieves a list of **stubs that have been matched during search operations**. This list is dynamically updated whenever a stub is successfully found via the `/api/stubs/search` endpoint. It provides visibility into which stubs are actively being utilized in your testing workflows.

#### **Request**
- **Method**: `GET`
- **URL**: `/api/stubs/used`
- **Parameters**: None required.
- **Headers**: Standard headers (e.g., `Content-Type: application/json`).

**Example Request**:
```bash
curl http://127.0.0.1:4771/api/stubs/used
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
      "data": { "message": "Hello GripMock", "return_code": 42 },
      "error": ""
    }
  }
]
```

#### **Stub Object Schema**
| Field   | Type     | Description                                                                 |
|---------|----------|-----------------------------------------------------------------------------|
| `id`    | `string` | Unique identifier for the stub (UUID format).                              |
| `service`| `string` | Name of the gRPC service (e.g., `Gripmock`).                              |
| `method` | `string` | Name of the gRPC method (e.g., `SayHello`).                               |
| `input`  | `object` | Input matching criteria (e.g., `equals`, `contains`, `matches`).          |
| `output` | `object` | Response configuration, including `data`, `error`, and gRPC status `code`.|

#### **Behavior Explanation**
- **Usage Tracking**: A stub is marked as "used" **only when it is matched during a search operation** (e.g., via `POST /api/stubs/search`).
- **Persistence**: The "used" state is ephemeral and resets when the GripMock server restarts.
- **Inverse of Unused**: The `/api/stubs/unused` endpoint returns stubs that have **never** been matched by a search.

#### **Example Workflow**
1. **Create a Stub**:
   ```bash
   curl -X POST -d '{
     "service": "Gripmock",
     "method": "SayHello",
     "input": { "equals": { "name": "gripmock" } },
     "output": { "data": { "message": "Hello GripMock", "return_code": 42 } }
   }' http://127.0.0.1:4771/api/stubs
   ```

2. **Search for the Stub** (marks it as used):
   ```bash
   curl -X POST -d '{
     "service": "Gripmock",
     "method": "SayHello",
     "data": { "name": "gripmock" }
   }' http://127.0.0.1:4771/api/stubs/search
   ```

3. **Retrieve Used Stubs**:
   ```bash
   curl http://127.0.0.1:4771/api/stubs/used
   ```

#### **Notes**
- **Multiple Matches**: If a stub is matched multiple times, it appears **once** in the list (no duplicates).
- **Related Endpoints**:
  - `GET /api/stubs/unused`: List stubs never matched by a search.
  - `POST /api/stubs/search`: Mark stubs as used by matching criteria.
- **Edge Cases**:
  - If no stubs have been used, the response is an empty array (`[]`).
  - Stubs are **not** marked as used when fetched by `GET /api/stubs` or `GET /stubs/{uuid}`.
