# YAML Stubs

YAML provides human-readable syntax with advanced features like comments and multi-document support, while maintaining compatibility with JSON structures.

## Schema Validation

GripMock provides a JSON Schema for validating stub definitions. Add this to your YAML files for IDE support:

```yaml
# yaml-language-server: $schema=https://bavix.github.io/gripmock/schema/stub.json

service: MyService
method: MyMethod
output:
  data:
    result: success
```

## When to Use YAML  
Perfect for:  
- Tests that don't need HTTP dependencies  
- Immutable/Versioned stub configurations  
- Large-scale stub management  
- Teams who prefer YAML's readability  

## Project Structure  
```
project-root/  
├── proto/  
│   └── simple.proto    # gRPC contract  
└── stubs/  
    ├── single.yaml     # Single stub  
    ├── multi-stubs.yml # Multiple stubs  
    └── nested/         # Organize in subdirectories  
```

## Stub Syntax  

### Single Stub (`single.yaml`)  
```yaml  
service: Gripmock  
method: SayHello  
input:  
  equals:  
    name: yaml-single  
output:  
  data:  
    message: Hello YAML  
    returnCode: 1  
```  

### Multiple Stubs (`multi-stubs.yml`)  
```yaml  
- service: Gripmock  
  method: SayHello  
  priority: 100
  input:  
    equals:  
      name: alpha  
  output:  
    data:  
      message: Hello Alpha  
      returnCode: 1  

- service: Gripmock  
  method: SayHello  
  priority: 1
  input:  
    equals:  
      name: beta  
  output:  
    data:  
      message: Hello Beta  
      returnCode: 2  
```  

## Docker Execution  
```bash  
docker run \  
  -p 4770:4770 \  
  -p 4771:4771 \  
  -v $(pwd)/proto:/proto:ro \  
  -v $(pwd)/stubs:/stubs:ro \  
  bavix/gripmock \  
  --stub=/stubs \  
  /proto/simple.proto  
```  

## Verification  
Check loaded stubs:  
```bash  
curl http://localhost:4771/api/stubs  
```  

**Sample Response**:  
```json  
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "service": "Gripmock",
    "method": "SayHello",
    "input": { "equals": { "name": "yaml-single" } },
    "output": { "data": { "message": "Hello YAML" } }
  },
  ...
]
```  

## Advanced Features  

### Array Order Handling  
```yaml  
input:  
  ignoreArrayOrder: true  
  equals:  
    ids: [3, 1, 2]  
```  

### Nested Structures  
```yaml  
input:  
  contains:  
    metadata:  
      env: production  
      version: 2.1  
```  

## Key Notes  
- 🔄 Auto-reloading: Changes in stub files are detected on container restart  
- 📁 Recursive loading: All .yaml/.yml files in --stub directory are processed  
- 🔍 Validation: Syntax errors in YAML files prevent server startup  
- 🔄 API Compatibility: Works alongside HTTP API for hybrid setups
- 🎯 Priority System: Use `priority` field to control stub matching order  

For schema details, see [JSON Schema for stubs](https://bavix.github.io/gripmock/schema/stub.json).  
