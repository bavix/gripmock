# Input Matching Rules <VersionTag version="v2.0.0" />

A stub selects requests by their body fields, using **equals**, **contains**, **matches**, **glob** and **anyOf**. These apply to the `data` field of a gRPC request.

For the formal composition rules (AND/OR logic, `anyOf` semantics, `ignoreArrayOrder` scoping), see [Matching Logic](./logic).

## Basic Syntax

```json
{
  "input": {
    "ignoreArrayOrder": false,
    "anyOf": [
      { "equals": {"field": "value"} }
    ],
    "equals|contains|matches|glob": {
      "field": "value"
    }
  }
}
```

## Matching Strategies

### 1. Exact Match (`equals`)

Matches **exact field names and values**, case-sensitive.

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
- Every declared field must be present and equal; fields the request carries but
  the stub does not declare are ignored
- Case-sensitive string comparison
- Arrays require exact order and the same elements (unless `ignoreArrayOrder: true`,
  which still compares them as a multiset — `["a","a"]` does not match `["a","b"]`)
- Nested objects and arrays are compared exactly, without the top-level leniency

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

Request `{"ips": ["10.64.0.2", "10.64.0.1"], ...}` will **not** match — same elements, different order. Set `ignoreArrayOrder: true` to accept it.

### 2. Partial Match (`contains`)

Matches requests whose body **contains** the specified structure. Containment is
structural: it ignores fields you did not mention, not parts of the values you did.

**Example:**
```yaml
input:
  contains:
    name: "gripmock"
    tags: ["grpc"]
    details:
      category: "test"
```

**When to use:**
- Checking a few fields of a large request and ignoring the rest
- Asserting that an array carries certain elements
- Matching a nested object partially

**Behavior:**
- Objects are matched recursively as a subset — unlisted fields are ignored
- Arrays match when every listed element is present, in any order
- Every other value, **strings included**, must be equal

::: warning
`contains` does **not** do substring matching. `contains: {name: "grip"}` will not
match `"gripmock"` — use [`matches`](#_3-regex-match-matches) with a regular
expression, or [`glob`](#_4-glob-match-glob) with a wildcard, for that.
:::

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

Request `{"ips": ["10.0.1.1", "10.0.1.2", "10.0.1.3"], ...}` will also match — the request array **contains** both specified IPs, and the extra one is ignored.

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

**Important:** Matching expressions must be static. Do not use dynamic templates (<code v-pre>`{{ ... }}`</code>) inside `equals`, `contains`, `matches`, `glob` or `anyOf`.

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

### 4. Glob Match (`glob`) <VersionTag version="v3.12.0" />

Uses **shell-style glob patterns** for simple but powerful pattern matching. Based on Go's `path.Match`.

**Example:**
```yaml
input:
  glob:
    name: "user_*"
    filename: "*.json"
    path: "/api/v1/*"
```

**Pattern Syntax:**
| Pattern | Meaning | Example Match |
|---------|---------|---------------|
| `*` | Match any characters (except path separator) | `user_123`, `test` |
| `?` | Match exactly one character | `user_1`, `file_a` |
| `[abc]` | Match character class | `file_a`, `file_b` |
| `[!abc]` | Negate character class | `file_x`, `file_z` |

::: tip Tip
`?` matches **exactly one** character. To match "value" use `??lue` or `val*`, not `val?ue`.
:::

**When to use:**
- Simple wildcards
- File path patterns
- API version matching
- Quick pattern testing

**Behavior:**
- Uses Go's `path.Match` function
- Case-sensitive
- `*` matches any sequence of non-separator characters
- `?` matches any single character

**Comparison: glob vs regex**

| Feature | `glob` | `matches` (regex) |
|---------|--------|-------------------|
| Pattern `*` | Match any characters (non-separator) | Match any characters (including `/`) |
| Pattern `?` | Match single character | Match single character |
| Pattern `[abc]` | Match character class | Match character class |
| Complexity | Simple wildcards | Full regex power |
| Use case | File paths, prefixes, suffixes | Email validation, complex formats |

**When to use glob:**
- File name patterns: `*.pdf`, `report_*.xlsx`
- API paths: `/api/v1/*/users`, `/static/*`
- Simple prefixes/suffixes: `user_*`, `*.tmp`

**When to use regex (`matches`):**
- Email/URL validation with exact format rules
- Complex patterns with alternation, anchors, quantifiers
- Patterns requiring specific character classes (`\d`, `\w`)

**Equivalents:**

| glob | regex equivalent |
|------|------------------|
| `users/*` | `users/.*` |
| `api/v1/*/items` | `api/v1/.*/items` |
| `file?.txt` | `file.\.txt` |
| `data[12].json` | `data[12]\.json` |

::: tip Note
`glob` uses Go's `path.Match` — `*` does NOT match path separators (`/`). For patterns that need to cross slashes, use `matches` (regex) with `.*`.
:::

**Real-World Example:**
```yaml
service: FileService
method: GetFile
input:
  glob:
    filename: "report_*.pdf"
    path: "/reports/2024/*"
output:
  data:
    content: "Binary file content..."
    size: 1024
```

**Important:** Like `matches`, glob patterns must be static (no dynamic templates).

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
