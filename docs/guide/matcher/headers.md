---
title: Headers
---

# Header Matching Rules <VersionTag version="v2.1.0" />

A stub can match on gRPC request headers as well as on the request body — for authentication, versioning or routing. Headers accept the same strategies as input: **equals**, **contains**, **matches**, **glob**, and **anyOf**.

For the formal composition rules (AND/OR logic, `anyOf` semantics), see [Matching Logic](./logic). This page covers header-specific behavior.

## Basic Syntax

```json
{
  "headers": {
    "anyOf": [
      { "equals": {"x-user": "alice"} },
      { "matches": {"authorization": "^Bearer .+"} }
    ],
    "equals|contains|matches": {
      "header-name": "expected_value"
    }
  }
}
```

## Matching Strategies

### 1. Exact Match (`equals`)

Matches headers with **exact name and value**.

```yaml
headers:
  equals:
    authorization: "Bearer token123"
    x-api-version: "v2"
```

**Behavior:**
- Header name and value must match exactly
- Case-sensitive comparison
- All specified headers must be present

### 2. Partial Match (`contains`)

Matches a **subset of the headers**: the ones you list must be equal, the rest are
ignored. Containment applies to the set of headers, not to the inside of a value.

```yaml
headers:
  contains:
    x-tenant: "acme"
```

**Behavior:**
- Only the listed headers are checked; every other header is ignored
- Each listed header value must be equal — including multi-value headers, which
  are compared as the whole `;`-separated string
- Case-sensitive by default

::: warning
`contains` does **not** match part of a header value. `contains: {authorization: "Bearer"}`
will not match `Bearer eyJhbGci…` — use [`matches`](#_3-regex-match-matches), for example
`authorization: "^Bearer "`.
:::

### 3. Regex Match (`matches`)

Uses **regular expressions** for advanced header pattern matching.

```yaml
headers:
  matches:
    x-request-id: "[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}"
    authorization: "Bearer [A-Za-z0-9]+"
```

**Behavior:**
- Uses Go's regex engine
- Case-sensitive by default (use `(?i)` for case-insensitive)
- Multiple values in header are matched individually

## Header-Specific Notes

### Multi-Value Headers

Multiple header values are separated by `;`:

```yaml
headers:
  equals:
    accept: "application/json; text/plain"
    x-forwarded-for: "192.168.1.1; 10.0.0.1"
```

### Case-Insensitive Matching

Use regex with `(?i)` flag:

```yaml
headers:
  matches:
    user-agent: "(?i)chrome"
    x-client: "(?i)mobile"
```

### Header Names

gRPC metadata keys are typically lowercase. Match against lowercase names:

```yaml
headers:
  equals:
    x-token: "abc123"   # not X-Token
```

## Real-World Examples

### Authentication Scenarios

**Admin User:**
```yaml
service: UserService
method: GetProfile
headers:
  equals:
    authorization: "Bearer admin-token-123"
    x-role: "admin"
output:
  data:
    user_id: "admin_001"
    role: "administrator"
```

**Regular User:**
```yaml
service: UserService
method: GetProfile
headers:
  contains:
    authorization: "Bearer user-"
output:
  data:
    user_id: "user_456"
    role: "user"
```

### API Versioning

```yaml
service: ProductService
method: GetProduct
headers:
  equals:
    x-api-version: "v2"
output:
  data:
    id: "prod_123"
    name: "Product Name"
    currency: "USD"
```

### Client-Specific Responses

```yaml
service: ContentService
method: GetContent
headers:
  contains:
    user-agent: "Mobile"
    x-platform: "ios"
output:
  data:
    content: "Mobile-optimized content"
    layout: "mobile"
```

### Debug and Tracing

```yaml
service: DebugService
method: GetLogs
headers:
  equals:
    x-debug-mode: "true"
    x-trace-id: "debug-session-123"
output:
  data:
    logs: ["debug log 1", "debug log 2"]
    level: "debug"
```

## Testing Header Matching

### Using the Search API

```bash
curl -X POST http://localhost:4771/api/stubs/search \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer token123" \
  -H "X-API-Version: v2" \
  -d '{
    "service": "Greeter",
    "method": "SayHello",
    "data": { "name": "gripmock" }
  }'
```

### Using gRPC Testify

```yaml
--- ENDPOINT ---
Greeter/SayHello

--- REQUEST_HEADERS ---
Authorization: Bearer token123
X-API-Version: v2

--- REQUEST ---
{
  "name": "gripmock"
}

--- RESPONSE ---
{
  "message": "Hello GripMock!"
}
```

## Troubleshooting

**No matches found:**
- Check header names (case-sensitive, typically lowercase in gRPC)
- Verify header values (exact match required for `equals`)
- Ensure all required headers are present

**Unexpected matches:**
- Review regex patterns
- Check for partial matches with `contains`
- Verify header value format (multi-value separated by `;`)

## Related Documentation

- [Matching Logic](./logic) — formal AND/OR composition rules
- [Input Matching](./input) — match request data
- [Stub Priority](../stubs/priority) — control stub selection order
