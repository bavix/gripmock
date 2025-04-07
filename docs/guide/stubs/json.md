# Static JSON Stubs  
Use static JSON/YAML files to predefine stubs without relying on the HTTP API. Ideal for:  
- Tests avoiding HTTP dependencies  
- Immutable stub configurations  
- Large-scale stub setups  

## Project Structure  
```
project-root/  
├── proto/  
│   └── simple.proto    # gRPC contract  
└── stubs/  
    ├── single.json     # Single stub  
    └── multi.json      # Multiple stubs  
```  

## Stub Examples  

### Single Stub (`single.json`)  
```json  
{
  "service": "Gripmock",
  "method": "SayHello",
  "input": {
    "equals": {
      "name": "single"
    }
  },
  "output": {
    "data": {
      "message": "Hello everyone",
      "return_code": 1
    }
  }
}
```  

### Multiple Stubs (`multi.json`)  
```json  
[
  {
    "service": "Gripmock",
    "method": "SayHello",
    "input": { "equals": { "name": "New York" } },
    "output": { "data": { "message": "Hello New York", "return_code": 1 } }
  },
  {
    "service": "Gripmock",
    "method": "SayHello",
    "input": { "equals": { "name": "world" } },
    "output": { "data": { "message": "Hello World", "return_code": 1 } }
  }
]
```  

## Docker Configuration  
Mount the `stubs` directory and specify the `--stub` flag:  
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
Check loaded stubs via the API:  
```bash  
curl http://localhost:4771/api/stubs  
```  

**Response**:  
```json  
[
  {
    "id": "a1b2c3d4-...",
    "service": "Gripmock",
    "method": "SayHello",
    "input": { "equals": { "name": "single" } },
    "output": { "data": { "message": "Hello everyone" } }
  },
  ...
]
```  

## Advanced Usage  

### YAML Support  
Use `.yaml`/`.yml` files as alternatives to JSON:  
```yaml  
# stubs/example.yaml  
service: Gripmock  
method: SayHello  
input:  
  equals:  
    name: yaml-stub  
output:  
  data:  
    message: Hello YAML  
```  

### Array Order Flag  
Disable array sorting checks with `ignoreArrayOrder`:  
```json  
{
  "input": {
    "ignoreArrayOrder": true,
    "equals": { "ids": [2, 1] }
  }
}
```  

---

**Key Notes**:  
- Stubs are loaded **on startup** from the `--stub` directory.  
- The HTTP API (`/api/stubs`) can still modify stubs dynamically.  
- File extensions: `.json`, `.yaml`, `.yml` (auto-detected).  

For schema details, see the [OpenAPI `Stub` definition](https://bavix.github.io/gripmock-openapi/).  
