# Schema Examples

Ready to see the JSON Schema in action? This guide shows you real examples of how to use it for different scenarios. We'll start simple and work our way up to more complex cases.

## Basic Examples

### Single Stub (JSON)

```json
{
  "$schema": "https://bavix.github.io/gripmock/schema/stub.json",
  "service": "UserService",
  "method": "GetUser",
  "input": {
    "equals": {
      "id": "user123"
    }
  },
  "output": {
    "data": {
      "id": "user123",
      "name": "John Doe",
      "email": "john@example.com"
    }
  }
}
```

### Single Stub (YAML)

```yaml
# yaml-language-server: $schema=https://bavix.github.io/gripmock/schema/stub.json

service: UserService
method: GetUser
input:
  equals:
    id: "user123"
output:
  data:
    id: "user123"
    name: "John Doe"
    email: "john@example.com"
```

### Multiple Stubs (Array)

```yaml
# yaml-language-server: $schema=https://bavix.github.io/gripmock/schema/stub.json

- service: UserService
  method: GetUser
  priority: 100
  input:
    equals:
      id: "admin"
  output:
    data:
      id: "admin"
      name: "Administrator"
      role: "admin"

- service: UserService
  method: GetUser
  priority: 1
  input:
    contains:
      id: "user"
  output:
    data:
      id: "unknown"
      name: "Unknown User"
      role: "user"
```

## Input Matching Examples

### Exact Match

```yaml
input:
  equals:
    id: "user123"
    type: "premium"
```

### Partial Match

```yaml
input:
  contains:
    name: "john"  # Matches "john", "johnny", "johnson"
```

### Regex Match

```yaml
input:
  matches:
    email: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
    phone: "^\\+?[1-9]\\d{1,14}$"
```

### Array Order Ignoring

```yaml
input:
  ignoreArrayOrder: true
  equals:
    tags: ["tag1", "tag2", "tag3"]  # Order doesn't matter
```

## Header Matching Examples

### Exact Header Match

```yaml
headers:
  equals:
    "Authorization": "Bearer token123"
    "Content-Type": "application/json"
```

### Partial Header Match

```yaml
headers:
  contains:
    "User-Agent": "Chrome"  # Matches any Chrome user agent
```

### Regex Header Match

```yaml
headers:
  matches:
    "X-Request-ID": "^req-[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$"
```

## Output Examples

### Simple Data Response

```yaml
output:
  data:
    success: true
    message: "Operation completed"
    timestamp: "2024-01-01T12:00:00Z"
```

### Error Response

```yaml
output:
  error: "User not found"
  code: 5  # NOT_FOUND
```

### Response with Delay

```yaml
output:
  data:
    result: "success"
  delay: "2.5s"  # 2.5 second delay
```

### Response with Headers

```yaml
output:
  data:
    token: "jwt-token-here"
  headers:
    "Set-Cookie": "session=abc123; HttpOnly"
    "X-Custom-Header": "custom-value"
```

## Streaming Examples

### Server-Side Streaming

```yaml
output:
  stream:
    - message: "First message"
      timestamp: "2024-01-01T12:00:01Z"
    - message: "Second message"
      timestamp: "2024-01-01T12:00:02Z"
    - message: "Final message"
      timestamp: "2024-01-01T12:00:03Z"
  delay: "1s"  # Delay between messages
```

### Streaming with Complex Data

```yaml
output:
  stream:
    - user:
        id: "user1"
        name: "Alice"
        status: "online"
    - user:
        id: "user2"
        name: "Bob"
        status: "offline"
    - user:
        id: "user3"
        name: "Charlie"
        status: "away"
  delay: "500ms"
```

## Priority Examples

### Specific vs General

```yaml
# High priority: Specific user
- service: UserService
  method: GetUser
  priority: 100
  input:
    equals:
      id: "admin"
  output:
    data:
      id: "admin"
      role: "administrator"

# Low priority: General fallback
- service: UserService
  method: GetUser
  priority: 1
  input:
    contains: {}  # Matches any input
  output:
    data:
      id: "unknown"
      role: "user"
```

### Error Handling with Priority

```yaml
# High priority: Specific error
- service: UserService
  method: GetUser
  priority: 100
  input:
    equals:
      id: "invalid"
  output:
    error: "Invalid user ID format"
    code: 3

# Medium priority: General validation
- service: UserService
  method: GetUser
  priority: 50
  input:
    contains:
      id: ""
  output:
    error: "User ID cannot be empty"
    code: 3

# Low priority: Generic error
- service: UserService
  method: GetUser
  priority: 1
  input:
    contains: {}
  output:
    error: "User not found"
    code: 5
```

## Complex Examples

### E-commerce Order Service

```yaml
# yaml-language-server: $schema=https://bavix.github.io/gripmock/schema/stub.json

- service: OrderService
  method: CreateOrder
  priority: 100
  input:
    equals:
      userId: "premium_user"
      items:
        - productId: "prod123"
          quantity: 2
  output:
    data:
      orderId: "order_12345"
      status: "confirmed"
      total: 199.99
      discount: 20.00
    delay: "1s"

- service: OrderService
  method: CreateOrder
  priority: 50
  input:
    contains:
      userId: "user"
  output:
    data:
      orderId: "order_67890"
      status: "pending"
      total: 99.99
      discount: 0.00
    delay: "500ms"

- service: OrderService
  method: CreateOrder
  priority: 1
  input:
    contains: {}
  output:
    error: "Invalid order data"
    code: 3
```

### Authentication Service

```yaml
# yaml-language-server: $schema=https://bavix.github.io/gripmock/schema/stub.json

- service: AuthService
  method: Login
  priority: 100
  input:
    equals:
      username: "admin"
      password: "admin123"
  output:
    data:
      token: "admin-jwt-token"
      expiresIn: 3600
      user:
        id: "admin"
        role: "administrator"
    headers:
      "Set-Cookie": "session=admin-session; HttpOnly; Secure"

- service: AuthService
  method: Login
  priority: 50
  input:
    contains:
      username: "user"
  output:
    data:
      token: "user-jwt-token"
      expiresIn: 1800
      user:
        id: "user123"
        role: "user"
    headers:
      "Set-Cookie": "session=user-session; HttpOnly"

- service: AuthService
  method: Login
  priority: 1
  input:
    contains: {}
  output:
    error: "Invalid credentials"
    code: 16  # UNAUTHENTICATED
```

## Best Practices

Here are some tips we've learned from real-world usage:

1. **Use meaningful priorities**: Higher numbers for specific cases, lower for fallbacks - it makes your logic easier to understand
2. **Validate your stubs**: Always test with the schema before deployment - it's like spell-checking your configuration
3. **Use descriptive service/method names**: Makes debugging easier when things go wrong
4. **Include realistic data**: Use data that matches your actual API - it makes your tests more reliable
5. **Test edge cases**: Include error scenarios and boundary conditions - real APIs have edge cases
6. **Document complex stubs**: Add comments for non-obvious logic - your future self will thank you
7. **Use consistent naming**: Follow a consistent pattern for IDs and fields - it helps with maintenance 