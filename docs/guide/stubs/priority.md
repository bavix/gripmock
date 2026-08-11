---
title: Priority
---

# Stub Priority <VersionTag version="v3.3.0" />

`priority` decides which stub answers when several of them match one request.
It is any integer, and defaults to 0.

## How a stub is chosen

Among the stubs that match, GripMock compares, in this order:

1. **Specificity** — how narrowly the stub's matchers describe the request.
2. **Score** — the match rank plus `priority × 10`.
3. **Field count** — the stub declaring more matcher fields wins.
4. **Stub ID** — a deterministic tiebreak, so a fully tied pair always resolves
   the same way.

Priority therefore outranks the match rank, but not specificity: a `contains`
stub with `priority: 1000` still loses to an `equals` stub that matches the same
request. Position in the file has no effect.

## Specific stub with a fallback

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

## Error scenarios

Three levels: a named error, a validation error, and a catch-all.

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

## JSON and YAML

### JSON
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

### YAML
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

## Cascading fallbacks

Three levels, narrowest first. Note that the priorities here only reinforce an
order that specificity already produces — `equals` beats `contains` beats the
catch-all regardless of the numbers.

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

Priority earns its keep when two stubs are equally specific — two `contains`
stubs over the same field, say — and you need one of them to win.

## Verification

```bash
# List all stubs with their priorities
curl http://localhost:4771/api/stubs
```

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