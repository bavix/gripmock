---
title: Matching Logic
---

# Matching Logic

GripMock uses a declarative matching model built on three strategies and one composition operator. This page defines the formal semantics that apply identically to [Input](./input) and [Header](./headers) matching.

## Strategies

| Strategy | Semantics |
|---|---|
| `equals` | Exact deep-equality. Every key-value pair must match the request verbatim. |
| `contains` | Substring/subset check. String values must contain the expected substring; arrays must contain all expected elements; objects are matched recursively. |
| `matches` | Regular expression. Each value is treated as a Go regex pattern applied to the corresponding request value. |

## Conjunction (AND)

Within one matcher block, all three strategies are AND-ed:

```
equals(request) AND contains(request) AND matches(request)
```

Empty/absent maps always pass (`len == 0 → true`), so in practice only the strategies you provide contribute to the result.

**Example:**

```yaml
input:
  equals:
    role: "admin"
  contains:
    name: "jo"
  matches:
    email: "^[a-z]+@example\\.com$"
```

Matches only when `role` is exactly `"admin"` **and** `name` contains `"jo"` **and** `email` matches the regex — all three must hold.

## Disjunction (`anyOf`) <VersionTag version="v3.11.0" />

`anyOf` adds an OR layer on top of the base conjunction:

```
base(equals, contains, matches) AND (anyOf[0] OR anyOf[1] OR ...)
```

Each `anyOf` element is itself a conjunction (`element.equals AND element.contains AND element.matches`). At least one element must pass for the whole `anyOf` to pass.

If `anyOf` is empty or absent, only the base conjunction is evaluated.

**`anyOf` is flat** — no nested `anyOf` inside `anyOf`. Depth is exactly 1.

### Example

```yaml
headers:
  equals:
    x-env: "prod"
  anyOf:
    - equals:
        x-user: "alice"
    - matches:
        authorization: "^Bearer admin_"
```

Formula:

```
x-env == "prod" AND (
  x-user == "alice" OR authorization ~= "^Bearer admin_"
)
```

| Request headers | Result | Reason |
|---|---|---|
| `x-env=prod`, `x-user=alice` | match | base ✓, alt[0] ✓ |
| `x-env=prod`, `authorization=Bearer admin_x` | match | base ✓, alt[1] ✓ |
| `x-env=staging`, `x-user=alice` | no match | base ✗ |
| `x-env=prod` | no match | base ✓, alt[0] ✗, alt[1] ✗ |

## `ignoreArrayOrder`

Controls whether arrays are compared as ordered sequences or as sets.

**Scoping rules:**

| Level | Affects |
|---|---|
| `input.ignoreArrayOrder` | Base `equals`/`contains`/`matches` only |
| `input.anyOf[i].ignoreArrayOrder` | That alternative only |
| **Not inherited** | Parent flag does not propagate into `anyOf` elements |

Each `anyOf` element must set its own `ignoreArrayOrder` if needed.

```yaml
input:
  ignoreArrayOrder: true        # affects base only
  equals:
    tags: ["grpc", "mock"]
  anyOf:
    - ignoreArrayOrder: true    # required; not inherited
      equals:
        ids: ["a", "b"]
    - matches:
        name: "^admin_"         # no arrays → flag irrelevant
```

## Empty matchers

A stub with all empty matchers (`equals: {}`, `contains: {}`, `matches: {}`, no `anyOf`) acts as a **catch-all** — it matches any request. This is useful for fallback stubs with low [priority](../stubs/priority).

## Formal grammar

```
Matcher      = Base AND AnyOf?
Base         = Equals(Request) AND Contains(Request) AND Matches(Request)
AnyOf        = Alt[0] OR Alt[1] OR ...
Alt[i]       = AltEquals(Request) AND AltContains(Request) AND AltMatches(Request)
```

Where each `*Equals`/`*Contains`/`*Matches` returns `true` when its map is empty.

## Summary

| Concept | Operator | Notes |
|---|---|---|
| Within one block | `AND` | `equals ∧ contains ∧ matches` |
| Between `anyOf` alternatives | `OR` | At least one must pass |
| Overall formula | `base AND (anyOf?)` | `anyOf` is optional |
| `ignoreArrayOrder` | Per-block | Not inherited into `anyOf` |

## Related

- [Input Matching](./input) — field-level details, arrays, nested objects
- [Header Matching](./headers) — multi-value headers, case sensitivity
- [Stub Priority](../stubs/priority) — how match score affects stub selection
