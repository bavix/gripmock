# Advanced Usage

## Working with Multiple Proto Files

### Project Structure
```
src/proto
├── common
│   └── address.proto  # Shared definitions
└── user
    └── user.proto     # Main service
```

### Docker Configuration
```yaml
services:
  gripmock:
    image: bavix/gripmock
    volumes:
      - ./src/proto:/proto:ro
      - ./mocks/user:/stubs:ro
    command: |
      --stub=/stubs \
      --imports=/proto \
      /proto/user/user.proto \
      /proto/common/address.proto
```

### Key Guidelines
- **Imports**: Use `--imports=/proto` to define the root directory for proto imports  
- **Explicit Files**: List all required `.proto` files in the command to prevent `File not found` errors  
- **Path Consistency**: Ensure volume paths (`./src/proto:/proto`) match import paths in your `.proto` files  

## Advanced Stub Configuration

### Parameterless Methods
For RPC methods with empty input (e.g., `rpc GetData(google.protobuf.Empty)`):
```json
{
  "service": "user.UserService",
  "method": "GetData",
  "input": { "matches": {} },
  "output": { 
    "data": { "content": "test" },
    "code": 0
  }
}
```

### Array Order Flexibility
Disable array sorting checks with `ignoreArrayOrder`:
```json
{
  "input": {
    "ignoreArrayOrder": true,
    "equals": {
      "ids": ["id2", "id1"]
    }
  }
}
```

### Custom gRPC Error Codes
Return errors with specific status codes:
```json
{
  "output": {
    "error": "Unauthorized",
    "code": 16  // gRPC 'Unauthenticated' code
  }
}
```

### Header Matching
Match requests based on headers:
```json
{
  "headers": {
    "contains": {
      "authorization": "Bearer token123"
    },
    "matches": {
      "user-agent": "^Mozilla.*$"
    }
  }
}
```

## Input/Output Matching Rules

### Input Matchers
| Rule       | Description                                                                 |
|------------|-----------------------------------------------------------------------------|
| `equals`   | Exact match for fields (case-sensitive)                                    |
| `contains` | Check for presence of fields (values ignored)                              |
| `matches`  | Regex matching for string fields (e.g., `"name": "^user_\\d+$"`)           |

### Output Configuration
| Field   | Description                                                                 |
|---------|-----------------------------------------------------------------------------|
| `data`  | Response payload matching your protobuf `message` structure                |
| `error` | gRPC error message (overrides `data` if `code` ≠ 0)                        |
| `code`  | gRPC status code (e.g., `3` for `InvalidArgument`, `5` for `NotFound`)     |

## Troubleshooting

### Common Issues

#### 1. Proto Import Errors
- **Error**: `common/address.proto: File not found`  
  **Fix**:  
  - Add `--imports=/proto` to specify the root directory  
  - Verify all dependencies are listed in the command  

#### 2. Docker Command Syntax
- **Error**: `unknown flag: --stub`  
  **Fix**: Use proper YAML formatting in `docker-compose.yml`:  
  ```yaml
  command: |
    --stub=/stubs \
    --imports=/proto \
    /proto/service.proto
  ```

#### 3. Path Mismatch
- **Error**: `File does not reside within any path specified using --proto_path`  
  **Fix**: Ensure all imported files are under directories specified in `--imports`

### Validation Steps
1. Check logs: `docker logs gripmock_container_id`  
2. Test proto compilation locally:  
   ```bash
   protoc --proto_path=./src/proto --go_out=. ./src/proto/user/user.proto
   ```

## Performance Tips
- **Stub Prioritization**: GripMock returns the **first matching stub**. Order stubs from most to least specific.  
- **Batch Operations**: Use `POST /api/stubs/batchDelete` for bulk deletions instead of individual API calls.  
- **Healthchecks**: Monitor with `GET /api/health/readiness` for production deployments.
