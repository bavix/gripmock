# Header Matching Rules

GripMock supports powerful header matching capabilities to control stub responses based on gRPC request headers. Use **equals**, **contains**, and **matches** rules to create precise header matching patterns for authentication, versioning, and request routing.

## Overview

Header matching allows you to respond differently based on request headers, making it perfect for:
- **Authentication** - Different responses for different tokens
- **Versioning** - API version-specific responses
- **Client identification** - Different behavior for mobile vs web clients
- **Request tracing** - Debug-specific responses

## Basic Syntax

```json
{
  "headers": {
    "equals|contains|matches": {
      "header-name": "expected_value"
    }
  }
}
```

**Important Notes:**
- Multiple header values are separated by `;` (e.g., `"x-foo": "bar;baz"`)
- All rules are case-sensitive unless using regex with `(?i)` flag
- Header names are typically lowercase in gRPC

## Matching Strategies

### 1. Exact Match (`equals`)

Matches headers with **exact name and value**. Perfect for precise header validation.

**Example:**
```yaml
headers:
  equals:
    authorization: "Bearer token123"
    user-agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
    x-api-version: "v2"
    content-type: "application/json"
```

**When to use:**
- ✅ Authentication token validation
- ✅ API version checking
- ✅ Content type validation
- ✅ Exact client identification

**Behavior:**
- Header name and value must match exactly
- Case-sensitive comparison
- All specified headers must be present
- Missing or incorrect headers cause no match

### 2. Partial Match (`contains`)

Matches headers that **contain** the specified values. Great for flexible header matching.

**Example:**
```yaml
headers:
  contains:
    user-agent: "Mozilla"      # Matches any Mozilla-based browser
    x-client-id: "mobile"      # Matches mobile clients
    authorization: "Bearer"    # Matches any Bearer token
```

**When to use:**
- ✅ Client type detection (mobile, web, etc.)
- ✅ Partial token validation
- ✅ Flexible version matching
- ✅ Optional header checking

**Behavior:**
- Checks if header value contains the specified substring
- Case-sensitive by default
- Missing headers are ignored
- Multiple values in header are supported

### 3. Regex Match (`matches`)

Uses **regular expressions** for advanced header pattern matching.

**Example:**
```yaml
headers:
  matches:
    user-agent: "^Mozilla.*$"           # Starts with Mozilla
    x-version: "v\\d+\\.\\d+"          # v1.2, v2.0, etc.
    authorization: "Bearer [A-Za-z0-9]+"  # Valid Bearer token format
    x-request-id: "[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}"  # UUID format
```

**When to use:**
- ✅ Complex token format validation
- ✅ Version pattern matching
- ✅ UUID/trace ID validation
- ✅ Advanced client detection

**Behavior:**
- Uses Go's regex engine
- Case-sensitive by default (use `(?i)` for case-insensitive)
- Supports all standard regex features
- Multiple values in header are matched individually

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
    permissions: ["read", "write", "delete"]
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
    permissions: ["read"]
```

### API Versioning

**Version 1 API:**
```yaml
service: ProductService
method: GetProduct
headers:
  equals:
    x-api-version: "v1"
output:
  data:
    id: "prod_123"
    name: "Product Name"
    price: 100
```

**Version 2 API:**
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
    price: 100
    currency: "USD"
    metadata:
      created_at: "2024-01-01T12:00:00Z"
```

### Client-Specific Responses

**Mobile Client:**
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
    features: ["touch", "swipe"]
```

**Web Client:**
```yaml
service: ContentService
method: GetContent
headers:
  contains:
    user-agent: "Mozilla"
    x-platform: "web"
output:
  data:
    content: "Full web content"
    layout: "desktop"
    features: ["keyboard", "mouse"]
```

### Debug and Tracing

**Debug Mode:**
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
    timestamp: "2024-01-01T12:00:00Z"
```

**Production Mode:**
```yaml
service: DebugService
method: GetLogs
headers:
  matches:
    x-trace-id: "[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}"
output:
  data:
    logs: ["production log"]
    level: "info"
    timestamp: "2024-01-01T12:00:00Z"
```

## Advanced Patterns

### Multiple Header Values

Headers can have multiple values separated by semicolons:

```yaml
headers:
  equals:
    accept: "application/json; text/plain"
    x-forwarded-for: "192.168.1.1; 10.0.0.1"
```

### Case-Insensitive Matching

Use regex with `(?i)` flag for case-insensitive matching:

```yaml
headers:
  matches:
    user-agent: "(?i)chrome"  # Matches Chrome, CHROME, chrome, etc.
    x-client: "(?i)mobile"    # Matches Mobile, MOBILE, mobile, etc.
```

### Complex Authentication

```yaml
headers:
  equals:
    authorization: "Bearer valid-token"
  matches:
    x-request-id: "[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}"
    x-timestamp: "\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}Z"
  contains:
    x-client-version: "1."
```

## Common Header Patterns

### Authentication Headers

```yaml
# Bearer token
authorization: "Bearer [A-Za-z0-9._-]+"

# API key
x-api-key: "[A-Za-z0-9]{32}"

# JWT token
authorization: "Bearer eyJ[A-Za-z0-9-_=]+\\.[A-Za-z0-9-_=]+\\.?[A-Za-z0-9-_.+/=]*"
```

### Version Headers

```yaml
# Semantic versioning
x-api-version: "v\\d+\\.\\d+\\.\\d+"

# Simple version
x-version: "\\d+"

# Date-based version
x-version: "\\d{4}-\\d{2}-\\d{2}"
```

### Client Headers

```yaml
# User agent patterns
user-agent: "Mozilla.*"
user-agent: ".*Mobile.*"
user-agent: ".*Android.*"

# Platform headers
x-platform: "(ios|android|web|desktop)"
x-client-type: "(mobile|tablet|desktop)"
```

## Testing Header Matching

### Using the Search API

Test your header matching with the search endpoint:

```bash
curl -X POST http://localhost:4771/api/stubs/search \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer token123" \
  -H "X-API-Version: v2" \
  -H "User-Agent: Mozilla/5.0" \
  -d '{
    "service": "Greeter",
    "method": "SayHello",
    "data": {
      "name": "gripmock"
    }
  }'
```

### Using gRPC Testify

```yaml
--- ENDPOINT ---
Greeter/SayHello

--- HEADERS ---
Authorization: Bearer token123
X-API-Version: v2
User-Agent: Mozilla/5.0

--- REQUEST ---
{
  "name": "gripmock"
}

--- RESPONSE ---
{
  "message": "Hello GripMock!"
}
```

## Performance Considerations

### Best Practices

1. **Use `equals` for exact matches** - Fastest header matching
2. **Use `contains` for partial matches** - Good balance of flexibility and performance
3. **Use `matches` sparingly** - Regex matching is slower
4. **Limit header count** - Too many headers can impact performance
5. **Cache common patterns** - Frequently used headers are cached

### Optimization Tips

```yaml
# ✅ Good - Simple and fast
headers:
  equals:
    authorization: "Bearer token123"

# ⚠️ Avoid - Complex regex for simple cases
headers:
  matches:
    authorization: "^Bearer token123$"  # Use equals instead

# ✅ Good - Specific matching
headers:
  contains:
    user-agent: "Mozilla"

# ⚠️ Avoid - Too broad matching
headers:
  contains:
    user-agent: "a"  # Too generic
```

## Troubleshooting

### Common Issues

**No matches found:**
- Check header names (case-sensitive)
- Verify header values (exact match required)
- Ensure all required headers are present
- Check for extra spaces or special characters

**Unexpected matches:**
- Review regex patterns
- Check for partial matches with `contains`
- Verify header value format

**Performance problems:**
- Simplify complex regex patterns
- Reduce number of headers to match
- Use `equals` instead of `matches` when possible

### Debug Tips

1. **Log all headers** - Use the search API to see what headers are being sent
2. **Test incrementally** - Start with simple matches and add complexity
3. **Use exact values** - Copy header values exactly from your client
4. **Check encoding** - Ensure special characters are properly encoded

## Related Documentation

- [Input Matching](./input.md) - Match request data
- [Stub Priority](../stubs/priority.md) - Control stub selection order
- [JSON Schema](../schema/) - Complete schema reference
- [Examples](../schema/examples.md) - More header matching examples
