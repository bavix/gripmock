---
title: Match Limit (times)
---

# Match Limit (options.times)

The `options.times` setting limits how many times a stub can be matched. After the limit is reached, the stub is exhausted and no longer used. This is useful for testing error scenarios, retries, and exact call counts.

## Overview

When a stub has `options.times` set:

- **times: 0** (default) — unlimited matches; the stub never exhausts
- **times: 1** — exactly one match, then the stub is exhausted
- **times: N** — up to N matches, then exhausted
- **Negative values** — invalid; rejected at validation

When a stub is exhausted, subsequent matching requests will not use it. If it was the only matching stub, the call will fail with `StubNotFound`.

## Basic Usage

### YAML Format

```yaml
# Stub that matches exactly once
- service: UserService
  method: GetUser
  input:
    equals:
      id: "one-time-user"
  output:
    data:
      id: "one-time-user"
      name: "Temporary"
  options:
    times: 1

# Stub that matches up to 3 times
- service: PaymentService
  method: ProcessPayment
  input:
    equals:
      orderId: "order-123"
  output:
    data:
      status: "success"
  options:
    times: 3

# Default: unlimited (omit options or set times: 0)
- service: HealthService
  method: Check
  input:
    contains: {}
  output:
    data:
      status: "ok"
```

### JSON Format

```json
{
  "service": "UserService",
  "method": "GetUser",
  "input": {"equals": {"id": "one-time-user"}},
  "output": {"data": {"id": "one-time-user", "name": "Temporary"}},
  "options": {"times": 1}
}
```

## Use Cases

### 1. One-Shot Stubs (Error After First Call)

Simulate a service that fails after the first successful call:

```yaml
# Succeeds once, then subsequent calls get StubNotFound
- service: CacheService
  method: Get
  input:
    equals:
      key: "expiring-key"
  output:
    data:
      value: "cached"
  options:
    times: 1
```

### 2. Retry Testing

Allow exactly N retries before failing:

```yaml
# Client retries 3 times; stub matches 3 times, then fails
- service: ExternalService
  method: Call
  input:
    equals:
      attempt: 1
  output:
    data:
      result: "success"
  options:
    times: 3
```

### 3. Different Limits per Input

Different inputs can have different match limits. Example: Ben matches once, Alice twice.

```yaml
# Ben: exactly 1 call, then exhausted
- service: helloworld.Greeter
  method: SayHello
  input:
    equals:
      name: "Ben"
  output:
    data:
      message: "Hello Ben"
  options:
    times: 1

# Alice: up to 2 calls, then exhausted
- service: helloworld.Greeter
  method: SayHello
  input:
    equals:
      name: "Alice"
  output:
    data:
      message: "Hello Alice"
  options:
    times: 2
```

**Flow:** SayHello("Ben") → ok; SayHello("Ben") again → StubNotFound. SayHello("Alice") × 2 → ok; third call → StubNotFound.

### 4. Fallback After Exhaustion

Use priority so a second stub handles calls after the first is exhausted:

```yaml
# First 2 calls: specific response
- service: QuoteService
  method: GetQuote
  priority: 10
  input:
    equals:
      symbol: "AAPL"
  output:
    data:
      price: 150.0
  options:
    times: 2

# After exhaustion: fallback
- service: QuoteService
  method: GetQuote
  priority: 1
  input:
    equals:
      symbol: "AAPL"
  output:
    error: "Service unavailable"
    code: 14
```

## Concurrency

The match limit is safe under concurrent access: `stubCallCount` is protected by a mutex. Under heavy concurrency, exactly `times` matches succeed; extra calls receive `StubNotFound`.

## Notes

- Exhausted stubs remain in storage; they are only excluded from matching
- `stubs/unused` and `stubs/used` reflect whether a stub has been matched (used count > 0)
- Clearing the storage (`DELETE /stubs` or restart) resets all match counts
