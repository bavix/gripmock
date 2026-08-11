---
title: Overview
---

# Matcher Overview

GripMock's matcher determines which stub responds to an incoming gRPC request based on input data and headers.

## Core Concepts

| Concept | Description |
|---|---|
| [Matching Logic](./logic) | AND/OR composition, `anyOf` semantics, `ignoreArrayOrder` |
| [Input](./input) | Match against request body fields |
| [Headers](./headers) | Match against gRPC metadata/headers |

## Quick Reference

```yaml
input:
  equals:       # exact match (AND)
  contains:     # partial match (AND)
  matches:      # regex match (AND)
  glob:         # glob match (AND)
  anyOf:        # OR alternatives
  ignoreArrayOrder: true

headers:
  # same structure
```

## Matching Flow

1. **Fast Path**: Exact `equals` matches are checked first
2. **Full Match**: `contains`, `matches`, `glob` and `anyOf` are evaluated
3. **Ranking**: Matching stubs are ordered by specificity, then by score
   (rank plus `priority × 10`), then by declared field count

## Examples

**Exact match:**
```yaml
input:
  equals:
    user_id: "123"
```

**Flexible match:**
```yaml
input:
  contains:
    name: "john"
  matches:
    email: "@company\\.com$"
```

**OR conditions:**
```yaml
input:
  equals:
    role: "admin"
  anyOf:
    - equals:
        department: "engineering"
    - matches:
        team: "^platform-"
```

## Related

- [Stub Priority](../stubs/priority)
- [Schema Reference](../schema/)