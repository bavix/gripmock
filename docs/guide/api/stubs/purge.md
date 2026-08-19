# Stub API: Purge Stubs
The `/api/stubs` endpoint with the `DELETE` method removes stubs from the storage. This is a destructive operation and cannot be undone.

## Request
- **Method**: `DELETE`
- **URL**: `/api/stubs`
- **Parameters**: None required.
- **Headers**: `X-Gripmock-Session` narrows the purge to one session; without it every stub is removed.

**Example Request**:
```bash
# Everything
curl -X DELETE http://127.0.0.1:4771/api/stubs

# Only the fixtures of one session
curl -X DELETE -H 'X-Gripmock-Session: team-a' http://127.0.0.1:4771/api/stubs
```

## Response
- **Status Code**: `204 No Content`  
- **Body**: Empty (no content returned).  

## Behavior
- **Unscoped**: Removes **all stubs** (both used and unused, every session).
- **Session-scoped**: With `X-Gripmock-Session`, only that session's stubs go; global stubs and
  other sessions are untouched. This is what the admin UI uses, so one client cannot wipe
  another session's fixtures.
- **Static Stubs**: Currently, all stubs in scope are deleted. A future flag may allow excluding static stubs.
- **Irreversible**: Deleted stubs cannot be recovered.

## Example Workflow
1. **Create Stubs**:  
   ```bash
   curl -X POST -d '[{"service":"Gripmock","method":"SayHello","input":{"equals":{"name":"test"}}}]' http://127.0.0.1:4771/api/stubs
   ```

2. **Verify Stubs Exist**:  
   ```bash
   curl http://127.0.0.1:4771/api/stubs
   ```
   **Response**:  
   ```json
   [{"id": "...", ...}]
   ```

3. **Purge All Stubs**:  
   ```bash
   curl -X DELETE http://127.0.0.1:4771/api/stubs
   ```

4. **Verify Deletion**:  
   ```bash
   curl http://127.0.0.1:4771/api/stubs
   ```
   **Response**:  
   ```json
   []
   ```

## Notes
- **Edge Cases**:  
  - If no stubs exist, the endpoint still returns `204`.  
  - `/api/stubs/used` and `/api/stubs/unused` both go empty: the stubs they reported are gone.  
- **Related Endpoints**:  
  - `GET /api/stubs`: List all stubs.  
  - `POST /api/stubs`: Add new stubs.  
  - `POST /api/stubs/batchDelete`: Delete specific stubs by ID.
  - `DELETE /api/history`: The same scoping rule for recorded calls.

## Schema References
For complete schema details, see:
- [OpenAPI Stub Definition](https://bavix.github.io/gripmock-openapi/)
- [JSON Schema for Stubs](https://bavix.github.io/gripmock/schema/stub.json)
