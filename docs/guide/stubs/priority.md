# Stub Priority

Stub priority allows you to control which stub is selected when multiple stubs match the same request. This is essential for creating complex mocking scenarios with fallback behaviors.

## Overview

When multiple stubs match a gRPC request, GripMock uses priority to determine which stub to use:

1. **Higher priority** stubs are selected first
2. **Lower priority** stubs serve as fallbacks
3. **Default priority** is 0 (if not specified)

## Priority Rules

### **Priority Values**
- **Higher numbers = Higher priority**
- **Default priority = 0**
- **Range**: Any integer value

### **Matching Order**
1. **Priority** (highest number wins)
2. **Order of definition** (first defined wins if priorities are equal)

## Use Cases

### **1. Specific vs General Matching**

Use priority to create specific handlers with general fallbacks:

```yaml
# High priority: Specific user
- service: UserService
  method: GetUser
  priority: 100
  input:
    equals:
      id: "user123"
  output:
    data:
      id: "user123"
      name: "John Doe"
      email: "john@example.com"

# Low priority: General fallback
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
      email: "unknown@example.com"
```

### **2. Error Scenarios**

Create specific error handlers with different priorities:

```yaml
# High priority: Specific error for invalid ID
- service: UserService
  method: GetUser
  priority: 100
  input:
    equals:
      id: "invalid"
  output:
    error: "Invalid user ID format"
    code: 3  # INVALID_ARGUMENT

# Medium priority: General validation error
- service: UserService
  method: GetUser
  priority: 50
  input:
    contains:
      id: ""
  output:
    error: "User ID cannot be empty"
    code: 3  # INVALID_ARGUMENT

# Low priority: Generic error fallback
- service: UserService
  method: GetUser
  priority: 1
  input:
    contains:
  output:
    error: "User not found"
    code: 5  # NOT_FOUND
```

### **3. Testing Different Scenarios**

Test various response scenarios with priority:

```yaml
# High priority: Success scenario
- service: PaymentService
  method: ProcessPayment
  priority: 100
  input:
    equals:
      amount: 100
      currency: "USD"
  output:
    data:
      transactionId: "txn_123"
      status: "success"
      amount: 100

# Medium priority: Partial success
- service: PaymentService
  method: ProcessPayment
  priority: 50
  input:
    contains:
      amount: 100
  output:
    data:
      transactionId: "txn_456"
      status: "pending"
      amount: 100

# Low priority: Error fallback
- service: PaymentService
  method: ProcessPayment
  priority: 1
  input:
    contains:
  output:
    error: "Payment processing failed"
    code: 13  # INTERNAL
```

## Examples

### **JSON Format**
```json
{
  "service": "AuthService",
  "method": "Authenticate",
  "priority": 100,
  "input": {
    "equals": {
      "username": "admin",
      "password": "secret"
    }
  },
  "output": {
    "data": {
      "token": "admin_token_123",
      "role": "admin"
    }
  }
}
```

### **YAML Format**
```yaml
service: AuthService
method: Authenticate
priority: 100
input:
  equals:
    username: "admin"
    password: "secret"
output:
  data:
    token: "admin_token_123"
    role: "admin"
```

## Advanced Patterns

### **1. Cascading Fallbacks**

Create multiple levels of fallback behavior:

```yaml
# Level 1: Exact match (highest priority)
- service: SearchService
  method: Search
  priority: 1000
  input:
    equals:
      query: "exact search term"
  output:
    data:
      results: ["exact match"]

# Level 2: Contains match (medium priority)
- service: SearchService
  method: Search
  priority: 100
  input:
    contains:
      query: "search"
  output:
    data:
      results: ["partial match 1", "partial match 2"]

# Level 3: Any match (lowest priority)
- service: SearchService
  method: Search
  priority: 1
  input:
    contains:
  output:
    data:
      results: ["default result"]
```

### **2. Environment-Specific Responses**

Use priority to simulate different environments:

```yaml
# Production-like responses (high priority)
- service: DataService
  method: GetData
  priority: 100
  input:
    contains:
      environment: "production"
  output:
    data:
      data: "production data"
      source: "production_db"

# Development responses (medium priority)
- service: DataService
  method: GetData
  priority: 50
  input:
    contains:
      environment: "development"
  output:
    data:
      data: "development data"
      source: "dev_db"

# Test responses (low priority)
- service: DataService
  method: GetData
  priority: 1
  input:
    contains:
  output:
    data:
      data: "test data"
      source: "mock_db"
```

### **3. Rate Limiting Simulation**

Simulate rate limiting with priority:

```yaml
# Rate limit exceeded (high priority)
- service: APIService
  method: CallAPI
  priority: 100
  input:
    contains:
      rateLimit: "exceeded"
  output:
    error: "Rate limit exceeded"
    code: 8  # RESOURCE_EXHAUSTED

# Normal response (low priority)
- service: APIService
  method: CallAPI
  priority: 1
  input:
    contains:
  output:
    data:
      result: "success"
      timestamp: "2024-01-01T12:00:00Z"
```

## Best Practices

### **1. Priority Ranges**
- **1000+**: Critical/specific scenarios
- **100-999**: Important business logic
- **10-99**: General fallbacks
- **1-9**: Default/error responses

### **2. Naming Convention**
Use descriptive comments to document priority levels:

```yaml
# Priority 1000: Exact match scenarios
- service: Service
  method: Method
  priority: 1000
  # ...

# Priority 100: Business logic scenarios  
- service: Service
  method: Method
  priority: 100
  # ...

# Priority 1: Fallback scenarios
- service: Service
  method: Method
  priority: 1
  # ...
```

### **3. Testing Strategy**
- Test with different priority combinations
- Verify fallback behavior works correctly
- Ensure no unintended stub matches occur

## Verification

### **Check Stub Priority**
```bash
# List all stubs with their priorities
curl http://localhost:4771/api/stubs
```

### **Test Priority Matching**
```bash
# Test specific high-priority stub
curl -X POST -d '{
  "service": "UserService",
  "method": "GetUser",
  "data": {"id": "user123"}
}' http://localhost:4771/api/stubs/search
```

## Related Documentation

- [Input Matching Rules](../matcher/input.md)
- [Header Matching Rules](../matcher/headers.md)
- [Stub Search API](../api/stubs/search.md)
- [JSON Schema](../schema/) 