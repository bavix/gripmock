# YAML Stubs

YAML stubs describe the same structure as JSON stubs, plus comments, anchors and
multi-document files. See [Why YAML?](./benefits-yaml) for the comparison.

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

Stub files load from disk at startup, so a test suite runs without touching the
HTTP API, the stub set is versioned alongside the code, and it scales to large
numbers of stubs.

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

## Key notes

- Auto-reloading: the stub watcher is on by default and picks up file changes
  without a restart. See [Why IDs matter](./why-ids-are-crucial).
- Recursive loading: every `.yaml` / `.yml` file under `--stub` is read.
- Validation: a syntax error in a stub file prevents server startup.
- File stubs and the HTTP API coexist; a hybrid setup is fine.
- `priority` controls which stub wins when several match.

For schema details, see [JSON Schema for stubs](https://bavix.github.io/gripmock/schema/stub.json).  
