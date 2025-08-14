# Input Matching Rules

GripMock provides powerful input matching capabilities to control stub responses. Use **equals**, **contains**, and **matches** rules to create precise request matching patterns that work with the `data` field in gRPC requests.

## Overview

Input matching is the core mechanism that determines which stub responds to your gRPC requests. GripMock supports three matching strategies, each with different use cases:

- **`equals`** - Exact value matching
- **`contains`** - Partial value matching  
- **`matches`** - Regular expression matching

## Basic Syntax

```json
{
  "input": {
    "ignoreArrayOrder": false,  // Optional: Disable array order checks
    "equals|contains|matches": {
      "field": "value"
    }
  }
}
```

## Matching Strategies

### 1. Exact Match (`equals`)

Matches **exact field names and values** (case-sensitive). Perfect for precise request matching.

**Example:**
```yaml
input:
  equals:
    name: "gripmock"
    age: 25
    active: true
    details:
      code: 42
      tags: ["grpc", "mock"]
```

**When to use:**
- ✅ Exact value validation
- ✅ Required field checking
- ✅ Numeric comparisons
- ✅ Boolean flags
- ✅ Nested object matching

**Behavior:**
- All fields must match exactly
- Case-sensitive string comparison
- Arrays require exact order (unless `ignoreArrayOrder: true`)
- Nested objects are compared recursively

### 2. Partial Match (`contains`)

Matches requests that **contain** the specified values. Great for flexible matching scenarios.

**Example:**
```yaml
input:
  contains:
    name: "grip"        # Matches "gripmock", "gripster", etc.
    tags: ["grpc"]      # Matches if array contains "grpc"
    details:
      category: "test"  # Matches nested fields
```

**When to use:**
- ✅ Partial string matching
- ✅ Array element checking
- ✅ Optional field validation
- ✅ Flexible matching requirements

**Behavior:**
- String values are checked for substring inclusion
- Array values check if elements exist (order doesn't matter)
- Nested objects are matched recursively
- Missing fields are ignored

### 3. Regex Match (`matches`)

Uses **regular expressions** for advanced pattern matching. Most powerful but requires regex knowledge.

**Example:**
```yaml
input:
  matches:
    email: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
    phone: "^\\+?[1-9]\\d{1,14}$"
    name: "^[A-Z][a-z]+$"  # Starts with capital letter
    version: "v\\d+\\.\\d+\\.\\d+"  # v1.2.3 format
```

**When to use:**
- ✅ Email validation
- ✅ Phone number formats
- ✅ Version string patterns
- ✅ Complex string patterns

**Behavior:**
- Uses Go's regex engine
- Case-sensitive by default (use `(?i)` for case-insensitive)
- Arrays are matched element-wise
- Supports all standard regex features

**Important:** Matching expressions must be static. Do not use dynamic templates (`{{ ... }}`) inside `equals`, `contains`, or `matches`. Example of incorrect usage:

```yaml
input:
  matches:
    value: "{{someDynamic}}"   # ❌ not allowed
```

Use static regex strings instead:

```yaml
input:
  matches:
    value: "^\\d+(\\.\\d+)?$"  # ✅ allowed
```

## Array Handling

### Order-Sensitive Matching (Default)

By default, arrays are compared in exact order:

```yaml
input:
  equals:
    tags: ["grpc", "mock", "test"]
```

**Matches:** `["grpc", "mock", "test"]`  
**Doesn't match:** `["mock", "grpc", "test"]`

### Order-Agnostic Matching

Use `ignoreArrayOrder: true` to ignore array element order:

```yaml
input:
  ignoreArrayOrder: true
  equals:
    tags: ["grpc", "mock", "test"]
```

**Matches:** `["grpc", "mock", "test"]`, `["mock", "grpc", "test"]`, `["test", "grpc", "mock"]`

## Real-World Examples

### User Authentication

```yaml
service: AuthService
method: Login
input:
  equals:
    username: "admin"
    password: "secret123"
  contains:
    client_id: "web"
output:
  data:
    token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
    expires_in: 3600
```

### Product Search

```yaml
service: ProductService
method: SearchProducts
input:
  contains:
    category: "electronics"
    tags: ["wireless", "bluetooth"]
  matches:
    price_range: "^\\d+-\\d+$"  # e.g., "100-500"
output:
  data:
    products:
      - id: "prod_123"
        name: "Wireless Headphones"
        price: 299
```

### Data Validation

```yaml
service: UserService
method: CreateUser
input:
  equals:
    status: "active"
  matches:
    email: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
    phone: "^\\+?[1-9]\\d{1,14}$"
  contains:
    preferences:
      notifications: true
output:
  data:
    user_id: "user_456"
    created_at: "2024-01-01T12:00:00Z"
```

## Advanced Patterns

### Combining Multiple Rules

You can combine different matching strategies for complex scenarios:

```yaml
input:
  equals:
    type: "premium"
    status: "active"
  contains:
    features: ["api", "support"]
  matches:
    domain: "^[a-zA-Z0-9][a-zA-Z0-9-]{1,61}[a-zA-Z0-9]\\.[a-zA-Z]{2,}$"
```

### Nested Object Matching

```yaml
input:
  equals:
    user:
      id: "user123"
      profile:
        name: "John Doe"
        preferences:
          theme: "dark"
          language: "en"
```

### Array with Complex Objects

```yaml
input:
  ignoreArrayOrder: true
  equals:
    items:
      - id: "item1"
        quantity: 2
      - id: "item2"
        quantity: 1
```

## Performance Considerations

### Best Practices

1. **Use `equals` for exact matches** - Fastest matching strategy
2. **Use `contains` for partial matches** - Good balance of flexibility and performance
3. **Use `matches` sparingly** - Regex matching is slower
4. **Limit nested depth** - Deep nesting can impact performance
5. **Use `ignoreArrayOrder` only when needed** - Adds processing overhead

### Optimization Tips

```yaml
# ✅ Good - Simple and fast
input:
  equals:
    user_id: "123"
    action: "create"

# ⚠️ Avoid - Complex regex for simple cases
input:
  matches:
    user_id: "^123$"  # Use equals instead

# ✅ Good - Specific matching
input:
  contains:
    tags: ["important"]

# ⚠️ Avoid - Too broad matching
input:
  contains:
    tags: ["a"]  # Too generic
```

## Testing Your Matches

### Using the Search API

Test your input matching with the search endpoint:

```bash
curl -X POST http://localhost:4771/api/stubs/search \
  -H "Content-Type: application/json" \
  -d '{
    "service": "Greeter",
    "method": "SayHello",
    "data": {
      "name": "gripmock",
      "age": 25,
      "tags": ["grpc", "mock"]
    }
  }'
```

### Using gRPC Testify

```yaml
--- ENDPOINT ---
Greeter/SayHello

--- REQUEST ---
{
  "name": "gripmock",
  "age": 25,
  "tags": ["grpc", "mock"]
}

--- RESPONSE ---
{
  "message": "Hello GripMock!"
}
```

## Troubleshooting

### Common Issues

**No matches found:**
- Check field names (case-sensitive)
- Verify data types (string vs number)
- Ensure array order matches (unless using `ignoreArrayOrder`)

**Unexpected matches:**
- Review regex patterns
- Check for partial matches with `contains`
- Verify nested object structure

**Performance problems:**
- Simplify complex regex patterns
- Reduce nested object depth
- Use `equals` instead of `matches` when possible

## Related Documentation

- [Header Matching](./headers.md) - Match request headers
- [Stub Priority](../stubs/priority.md) - Control stub selection order
- [JSON Schema](../schema/) - Complete schema reference
- [Examples](../schema/examples.md) - More input matching examples  
