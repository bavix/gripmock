### **Stub API. Stubs Purge**  
The `/api/stubs` endpoint with the `DELETE` method removes **all stubs** from the storage. This is a destructive operation and cannot be undone.  

#### **Request**  
- **Method**: `DELETE`  
- **URL**: `/api/stubs`  
- **Parameters**: None required.  
- **Headers**: Standard headers (e.g., `Content-Type: application/json`).  

**Example Request**:  
```bash
curl -X DELETE http://127.0.0.1:4771/api/stubs
```

#### **Response**  
- **Status Code**: `204 No Content`  
- **Body**: Empty (no content returned).  

#### **Behavior**  
- **Global Deletion**: Removes **all stubs** (both used and unused).  
- **Static Stubs**: Currently, all stubs are deleted. A future flag may allow excluding static stubs.  
- **Irreversible**: Deleted stubs cannot be recovered.  

#### **Example Workflow**  
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

#### **Notes**  
- **Edge Cases**:  
  - If no stubs exist, the endpoint still returns `204`.  
  - Does not affect the `/api/stubs/used` or `/api/stubs/unused` lists (they reset automatically).  
- **Related Endpoints**:  
  - `GET /api/stubs`: List all stubs.  
  - `POST /api/stubs`: Add new stubs.  
  - `POST /api/stubs/batchDelete`: Delete specific stubs by ID.  

---

This endpoint is useful for resetting stub storage between test runs or cleaning up outdated configurations. Use with caution.