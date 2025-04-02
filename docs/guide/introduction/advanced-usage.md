# Advanced Usage

## Working with Multiple Proto Files
**Project Structure Example**  
```  
src/proto  
├── common  
│   └── address.proto  # Shared definitions  
└── user  
    └── user.proto     # Main service  
```  

**Docker Compose Configuration**  
```yaml  
services:  
  gripmock:  
    image: bavix/gripmock  
    volumes:  
      - ./src/proto:/proto:ro  
      - ./mocks/user:/stubs:ro  
    command: --stub=/stubs --imports=/proto /proto/user/user.proto /proto/common/address.proto  
```  

**Key Points**  
- Use `--imports=/proto` to define the root directory for imports   
- Explicitly list all required `.proto` files in the command to avoid `File not found` errors   
- Ensure volume mappings (`./src/proto:/proto`) match the paths in your proto imports   

---

## Stub Configuration for Special Cases
**Stub for Parameterless Methods**  
For RPC methods with no input (e.g., `rpc GetData(Empty)`):  
```json  
{  
  "service": "user.UserService",  
  "method": "GetData",  
  "input": { "matches": {} },  // Empty input matcher  
  "output": { "data": "test" }  
}  
```  

**Notes**  
- Use `{ "matches": {} }` to match empty input   
- Avoid omitting `input` field to prevent validation errors  

---

## Troubleshooting
**Common Issues & Fixes**  

1. **Import Errors**  
   - **Error**: `common/address.proto: File not found`  
     **Fix**:  
     - Add `--imports=/proto` to specify the root directory   
     - Verify proto files are listed in the command   

2. **Docker Command Syntax**  
   - **Error**: `unknown flag` or `invalid syntax`  
     **Fix**: Use one of these formats in `docker-compose.yml`:  
     ```yaml  
     # Single-line  
     command: --stub=/stubs --imports=/proto /proto/user/user.proto  
     
     # Multi-line  
     command: |  
       --stub=/stubs \  
       --imports=/proto \  
       /proto/user/user.proto  
     ```  

3. **Proto Path Mismatch**  
   - **Error**: `File does not reside within any path specified using --proto_path`  
     **Fix**: Ensure all imported files are under directories specified in `--imports`   

**Validation Steps**  
1. Check Docker logs: `docker logs <container_id>`  
2. Verify proto files compile locally with `protoc`   
