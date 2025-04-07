# **Stub API. Stubs Delete**  
The `/api/stubs/{uuid}` endpoint with the `DELETE` method removes a **specific stub** by its unique identifier (UUID).  

## **Request**  
- **Method**: `DELETE`  
- **URL**: `/api/stubs/{uuid}`  
- **Parameters**:  
  - `uuid` (path parameter): The ID of the stub to delete (UUID format).  
- **Headers**: Standard headers (e.g., `Content-Type: application/json`).  

**Example Request**:  
```bash
curl -X DELETE http://127.0.0.1:4771/api/stubs/6c85b0fa-caaf-4640-a672-f56b7dd8074d
```

## **Response**  
- **Success**:  
  - **Status Code**: `204 No Content`  
  - **Body**: Empty (no content returned).  

- **Error**:  
  - **Status Code**: `404 Not Found` (if the stub does not exist).  
  - **Body**:  
    ```json
    { "error": "Stub with ID '6c85b0fa-caaf-4640-a672-f56b7dd8074d' not found" }
    ```

## **Behavior**  
- **Idempotency**:  
  - Returns `204` even if the stub does not exist (no error).  
- **Impact on Lists**:  
  - If the stub was marked as "used" or "unused," it is removed from both `/api/stubs/used` and `/api/stubs/unused`.  

## **Example Workflow**  
1. **Create a Stub**:  
   ```bash
   curl -X POST -d '{
     "service": "Gripmock",
     "method": "SayHello",
     "input": { "equals": { "name": "gripmock" } }
   }' http://127.0.0.1:4771/api/stubs
   ```
   **Response**:  
   ```json
   ["6c85b0fa-caaf-4640-a672-f56b7dd8074d"]
   ```

2. **Delete the Stub**:  
   ```bash
   curl -X DELETE http://127.0.0.1:4771/api/stubs/6c85b0fa-caaf-4640-a672-f56b7dd8074d
   ```
   **Response**: `204 No Content`

3. **Verify Deletion**:  
   ```bash
   curl http://127.0.0.1:4771/api/stubs/6c85b0fa-caaf-4640-a672-f56b7dd8074d
   ```
   **Response**:  
   ```http
   HTTP/1.1 404 Not Found
   Content-Type: application/json
   Date: Fri, 04 Apr 2025 22:31:30 GMT
   Content-Length: 0

   { "error": "Stub with ID '6c85b0fa-caaf-4640-a672-f56b7dd8074d' not found" }
   ```

## **Notes**  
- **Edge Cases**:  
  - Deleting a non-existent stub does **not** return an error (idempotent operation).  
  - Use `GET /api/stubs/{uuid}` to verify existence before deletion.  
- **Related Endpoints**:  
  - `DELETE /api/stubs`: Purge all stubs.  
  - `POST /api/stubs`: Create or update stubs.  
  - `GET /api/stubs/used` and `GET /api/stubs/unused`: Track stub usage.  

---

This endpoint is critical for precise management of stubs during testing. Use it to remove outdated or redundant configurations.
