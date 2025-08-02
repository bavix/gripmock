# Input Matching Rules  
GripMock supports three input-matching strategies to control stub responses: **equals**, **contains**, and **matches**. These rules apply to the `data` field in gRPC requests and support nested JSON structures.

## Input Matching Syntax  
```json
{
  "input": {
    "ignoreArrayOrder": false, // Optional: Disable array order checks
    "equals|contains|matches": {
      "field": "value"
    }
  }
}
```

### 1. **Exact Match (`equals`)**  
Matches **exact field names and values** (case-sensitive).  
**Example**:  
```json
{
  "input": {
    "equals": {
      "name": "gripmock",
      "details": {
        "code": 42,
        "active": true
      },
      "tags": ["grpc", "mock"]
    }
  }
}
```

**Behavior**:  
- Nested fields (e.g., `details.code`) must match exactly.  
- Arrays require **exact order** unless `ignoreArrayOrder: true` is set.  

### 2. **Regex Match (`matches`)**  
Matches fields using **regular expressions**.  
**Example**:  
```json
{
  "input": {
    "matches": {
      "name": "^grip.*$", // Starts with "grip"
      "cities": ["Jakarta", ".*grad$"] // Ends with "grad"
    }
  }
}
```

**Behavior**:  
- Arrays are matched element-wise (e.g., `["Jakarta", ".*grad$"]` requires at least one element matching each regex).  
- Uses the `github.com/gripmock/deeply` library for recursive regex evaluation.  

## Array Order Flexibility  
Use `ignoreArrayOrder: true` to disable array sorting checks:  
```json
{
  "input": {
    "ignoreArrayOrder": true,
    "equals": {
      "ids": ["id2", "id1"] // Matches ["id1", "id2"] or any order
    }
  }
}
```

**Default Behavior**:  
- Arrays are compared **in order** (e.g., `["a", "b"]` ≠ `["b", "a"]`).  
- Enable `ignoreArrayOrder` for order-agnostic comparisons.  

## Usage Example  
**Stub Definition**:  
```json
{
  "service": "Greeter",
  "method": "SayHello",
  "input": {
    "ignoreArrayOrder": true,
    "contains": { "id": "123", "tags": ["grpc", "mock"] },
    "matches": { "name": "^user_\\d+$" }
  },
  "output": { "data": { "message": "Matched!" } }
}
```

**Matching Request**:  
```bash
curl -X POST \
  -d '{
    "service": "Greeter",
    "method": "SayHello",
    "data": {
      "id": "123",
      "name": "user_456",
      "tags": ["mock", "grpc"]  // Order doesn't matter
    }
  }' \
  http://localhost:4771/api/stubs/search
```

## Key Notes  
- **Nested Fields**: All rules work recursively (e.g., `equals.details.code`).  
- **Data Types**: Supports `string`, `number`, `boolean`, `null`, and arrays.  
- **Combining Rules**: Use multiple rules (e.g., `contains` + `matches`) for stricter conditions.  
- **Performance**: Complex regex/nested structures may impact matching speed.
- **Priority**: When multiple stubs match, use `priority` field to control selection order.

## Related Documentation

- [Stub Priority](../stubs/priority.md) - Control stub matching order
- [Header Matching](./headers.md) - Match request headers
- [JSON Schema](../schema/) - Complete schema reference  
