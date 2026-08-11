# Stub API: List Used Stubs

The `/api/stubs/used` endpoint returns the stubs that have been matched at least once since the server started. Use it to see which stubs a test run actually exercised.

## Request
- **Method**: `GET`
- **URL**: `/api/stubs/used`
- **Parameters**: None required.
- **Headers**: Standard headers (e.g., `Content-Type: application/json`).

**Example Request**:
```bash
curl http://127.0.0.1:4771/api/stubs/used
```

## Response
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

## Stub Object Schema
| Field   | Type     | Description                                                                 |
|---------|----------|-----------------------------------------------------------------------------|
| `id`    | `string` | Unique identifier for the stub (UUID format).                              |
| `service`| `string` | Name of the gRPC service (e.g., `Gripmock`).                              |
| `method` | `string` | Name of the gRPC method (e.g., `SayHello`).                               |
| `input`  | `object` | Input matching criteria (`equals`, `contains`, `matches`, `glob`, `anyOf`, `ignoreArrayOrder`). |
| `output` | `object` | Response configuration, including `data`, `error`, and gRPC status `code`.|

## Behavior
- **Usage Tracking**: A stub is marked as "used" the first time it is matched — by a real gRPC call, by `POST /api/stubs/search`, or through the MCP API.
- **Persistence**: The "used" state is ephemeral and resets when the GripMock server restarts.
- **Inverse of Unused**: The `/api/stubs/unused` endpoint returns stubs that have **never** been matched.

## Example Workflow
1. **Create a Stub**:
   ```bash
   curl -X POST -d '{
     "service": "Gripmock",
     "method": "SayHello",
     "input": { "equals": { "name": "gripmock" } },
     "output": { "data": { "message": "Hello GripMock", "returnCode": 42 } }
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

## Notes
- **Multiple Matches**: If a stub is matched multiple times, it appears **once** in the list (no duplicates).
- **Related Endpoints**:
  - `GET /api/stubs/unused`: List stubs never matched by a search.
  - `POST /api/stubs/search`: Mark stubs as used by matching criteria.
- **Edge Cases**:
  - If no stubs have been used, the response is an empty array (`[]`).
  - Stubs are **not** marked as used when fetched by `GET /api/stubs` or `GET /api/stubs/{uuid}`.

## Schema References
For complete schema details, see:
- [OpenAPI Stub Definition](https://bavix.github.io/gripmock-openapi/)
- [JSON Schema for Stubs](https://bavix.github.io/gripmock/schema/stub.json)
