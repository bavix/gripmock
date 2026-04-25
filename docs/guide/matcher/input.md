# Input Matching Rules <VersionTag version="v2.0.0" />

GripMock provides powerful input matching capabilities to control stub responses. Use **equals**, **contains**, **matches**, and **anyOf** rules to create precise request matching patterns that work with the `data` field in gRPC requests.

For the formal composition rules (AND/OR logic, `anyOf` semantics, `ignoreArrayOrder` scoping), see [Matching Logic](./logic).

## Basic Syntax

```json
{
  "input": {
    "ignoreArrayOrder": false,
    "anyOf": [
      { "equals": {"field": "value"} }
    ],
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
- Exact value validation
- Required field checking
- Numeric comparisons
- Boolean flags
- Nested object matching

**Behavior:**
- All fields must match exactly
- Case-sensitive string comparison
- Arrays require exact order (unless `ignoreArrayOrder: true`)
- Nested objects are compared recursively

**Array Example** — exact match with `repeated` field, order matters:

```yaml
service: inventory.InventoryService
method: GetResourceDecorationByIPsStream
input:
  equals:
    k8s_cluster_id: "scale-test-cluster"
    ips: ["10.64.0.1", "10.64.0.2"]
output:
  data:
    ips_to_decorations:
      "10.64.0.1":
        decoration: "web-frontend"
        environment: "production"
      "10.64.0.2":
        decoration: "api-backend"
        environment: "staging"
```

Request `{"ips": ["10.0.3.2", "10.0.3.1"], ...}` will **not** match — array order differs.

### 2. Partial Match (`contains`)

Matches requests that **contain** the specified values. Great for flexible matching scenarios.

**Example:**
```yaml
input:
  contains:
    name: "grip"
    tags: ["grpc"]
    details:
      category: "test"
```

**When to use:**
- Partial string matching
- Array element checking
- Optional field validation

**Behavior:**
- String values are checked for substring inclusion
- Array values check if elements exist (order doesn't matter)
- Nested objects are matched recursively
- Missing fields are ignored

**Array Example** — `repeated` field contains specified elements:

```yaml
service: inventory.InventoryService
method: GetResourceDecorationByIPsStream
input:
  contains:
    k8s_cluster_id: "test-contains"
    ips: ["10.0.1.1", "10.0.1.2"]
output:
  data:
    ips_to_decorations:
      "10.0.1.1":
        decoration: "web-frontend"
        environment: "production"
      "10.0.1.2":
        decoration: "web-frontend"
        environment: "production"
```

Request `{"ips": ["10.0.1.1", "10.0.1.2", "10.0.1.3"], ...}` will also match — the response array **contains** both specified IPs.

### 3. Regex Match (`matches`)

Uses **regular expressions** for advanced pattern matching.

**Example:**
```yaml
input:
  matches:
    email: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
    phone: "^\\+?[1-9]\\d{1,14}$"
    name: "^[A-Z][a-z]+$"
```

**When to use:**
- Email validation
- Phone number formats
- Version string patterns
- Complex string patterns

**Behavior:**
- Uses Go's regex engine
- Case-sensitive by default (use `(?i)` for case-insensitive)
- Arrays are matched element-wise
- Supports all standard regex features

**Array Example** — regex matching on `repeated` field elements:

```yaml
service: inventory.InventoryService
method: GetResourceDecorationByIPsStream
input:
  matches:
    k8s_cluster_id: "^test-matches$"
    ips: ["^10\\.0\\.2\\.[0-9]+$", "^10\\.0\\.2\\.[0-9]+$"]
output:
  data:
    ips_to_decorations:
      "10.0.2.77":
        decoration: "api-backend"
        environment: "staging"
      "10.0.2.88":
        decoration: "api-backend"
        environment: "staging"
```

**Important:** Matching expressions must be static. Do not use dynamic templates (<code v-pre>`{{ ... }}`</code>) inside `equals`, `contains`, or `matches`.

::: v-pre
```yaml
input:
  matches:
    value: "{{someDynamic}}"   # NOT allowed
```
:::

Use static regex strings instead:

```yaml
input:
  matches:
    value: "^\\d+(\\.\\d+)?$"  # OK
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

Works with all three matchers:

```yaml
input:
  ignoreArrayOrder: true
  equals:
    k8s_cluster_id: "test-equals-ignore"
    ips: ["10.0.3.1", "10.0.3.2"]

input:
  ignoreArrayOrder: true
  contains:
    k8s_cluster_id: "test-contains-ignore"
    ips: ["10.0.4.1", "10.0.4.2"]

input:
  ignoreArrayOrder: true
  matches:
    k8s_cluster_id: "^test-matches-ignore$"
    ips: ["^10\\.0\\.5\\.[0-9]+$", "^10\\.0\\.5\\.[0-9]+$"]
```

`ignoreArrayOrder` is scoped per-block — see [Matching Logic](./logic#ignorearrayorder) for scoping details.

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
    price_range: "^\\d+-\\d+$"
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

## Troubleshooting

**No matches found:**
- Check field names (case-sensitive)
- Verify data types (string vs number)
- Ensure array order matches (unless using `ignoreArrayOrder`)

**Unexpected matches:**
- Review regex patterns
- Check for partial matches with `contains`
- Verify nested object structure

## Related Documentation

- [Matching Logic](./logic) — formal AND/OR composition rules
- [Header Matching](./headers) — match request headers
- [Stub Priority](../stubs/priority) — control stub selection order
- [Examples](../schema/examples) — more input matching examples
