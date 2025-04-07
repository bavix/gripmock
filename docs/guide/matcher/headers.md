# Headers Matching Rules  
GripMock supports three header-matching strategies to control stub responses: **equals**, **contains**, and **matches**. These rules apply to gRPC request headers and are evaluated recursively.

## Header Matching Syntax  
```json
{
  "headers": {
    "equals|contains|matches": {
      "header-name": "expected_value"
    }
  }
}
```

**Notes**:
- Multiple header values are separated by `;` (e.g., `"x-foo": "bar;baz"`).
- All rules are case-sensitive unless using regex (e.g., `(?i)` flag).

### 1. **Exact Match (`equals`)**  
Matches headers with **exact name and value**.  
**Example**:  
```json
{
  "headers": {
    "equals": {
      "authorization": "Bearer token123",
      "user-agent": "Mozilla/5.0"
    }
  }
}
```

**Behavior**:  
- The request must include **both** headers with the exact values.  
- Fails if headers are missing or values differ.

### 2. **Presence Match (`matches`)**  
Matches if the header **exists** (value is ignored).  
**Example**:  
```json
{
  "headers": {
    "matches": {
      "x-request-id": ".+"
    }
  }
}
```

**Behavior**:  
- Requires the header `X-Request-ID` to exist (any value allowed).  
- Use filter matches with string `.+` as a placeholder for value-agnostic checks.

### 3. **Regex Match (`matches`)**  
Matches headers using **regular expressions**.  
**Example**:  
```json
{
  "headers": {
    "matches": {
      "user-agent": "^Mozilla.*$",
      "x-version": "v\\d+"
    }
  }
}
```

**Behavior**:  
- `User-Agent` must start with `Mozilla`.  
- `X-Version` must match `v1`, `v2`, etc.  

## Usage Example  
**Stub Definition**:  
```json
{
  "service": "Greeter",
  "method": "SayHello",
  "headers": {
    "contains": { "authorization": "Bearer abc123" },
    "matches": { "user-agent": ".*Android.*" }
  },
  "output": { "data": { "message": "Hello Android User!" } }
}
```

## Key Notes  
- **Multiple Values**: Use `;` to separate values (e.g., `"X-Forwarded-For": "1.2.3.4;5.6.7.8"`).  
- **Recursive Matching**: GripMock checks all nested fields (if applicable).  
