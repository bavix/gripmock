---
title: Delay Configuration
---

# Delay Configuration <VersionTag version="v3.2.16" />

`output.delay` makes the server wait before sending a response — long enough to
exercise a client timeout, a retry path, or a slow-network assumption.

## Basic Usage

### JSON Format
```json
{
  "service": "TrackService",
  "method": "StreamTrack",
  "input": {
    "equals": {
      "stn": "MS#00001"
    }
  },
  "output": {
    "delay": "100ms",
    "data": {
      "stn": "MS#00001",
      "identity": "00",
      "latitude": 0.1,
      "longitude": 0.005,
      "speed": 45,
      "updatedAt": "2024-01-01T12:00:00.000Z"
    }
  }
}
```

### YAML Format
```yaml
service: TrackService
method: StreamTrack
input:
  equals:
    stn: "MS#00001"
output:
  delay: 100ms
  data:
    stn: "MS#00001"
    identity: "00"
    latitude: 0.1
    longitude: 0.005
    speed: 45
    updatedAt: "2024-01-01T12:00:00.000Z"
```

## Delay Formats

GripMock supports various time duration formats:

### Supported Units
- **Milliseconds**: `100ms`, `500ms`, `1.5ms`
- **Seconds**: `1s`, `2.5s`, `30s`
- **Minutes**: `1m`, `5m`
- **Hours**: `1h`, `2h`

### Examples
```yaml
output:
  delay: 100ms    # 100 milliseconds
  delay: 2.5s     # 2.5 seconds
  delay: 1m       # 1 minute
  delay: 1h30m    # 1 hour 30 minutes
```

## Streaming Responses

For streaming responses, delay is applied **before** every message in the stream.

### Uniform Delay

```yaml
service: TrackService
method: StreamTrack
input:
  equals:
    stn: "MS#00005"
output:
  delay: 200ms
  stream:
    - stn: "MS#00005"
      identity: "00"
      latitude: 0.11
      longitude: 0.006
      speed: 50
      updatedAt: "2024-01-01T13:00:00.000Z"
    - stn: "MS#00005"
      identity: "01"
      latitude: 0.11001
      longitude: 0.00601
      speed: 51
      updatedAt: "2024-01-01T13:00:01.000Z"
    - stn: "MS#00005"
      identity: "02"
      latitude: 0.11002
      longitude: 0.00602
      speed: 52
      updatedAt: "2024-01-01T13:00:02.000Z"
```

The 200ms delay is applied **before** every message:
- 200ms → Message 1 → 200ms → Message 2 → 200ms → Message 3

### Per-Element Delay <VersionTag version="v3.15.1" />

Use the reserved `_gripmock` key inside a stream element to set a per-message delay:

```yaml
output:
  stream:
    - _gripmock:
        delay: 50ms
      status: NOT_SERVING
    - _gripmock:
        delay: 150ms
      status: SERVING
    - _gripmock:
        delay: 100ms
      status: DONE
```

Each message gets its own delay instead of the global `output.delay`:

- 50ms → `{status: NOT_SERVING}`
- 150ms → `{status: SERVING}`
- 100ms → `{status: DONE}`

The `_gripmock` key is **reserved** — it does not conflict with protobuf field names
and is stripped before the message is sent to the client. The `_gripmock.delay` value
takes a duration (`100ms`, `1s`) or a [template](#computed-delay).

When a stream element contains `_gripmock.delay`, the per-element delay takes
**priority** over the global `output.delay`. Elements without `_gripmock` still
use the global delay.

This is especially useful for captured streams where `recordDelay` records
the actual inter-message timing from the upstream service. See
[Capture Mode](../modes/capture) for details.

## Computed Delay <VersionTag version="v3.21.0" />

`delay` also takes a [template](./dynamic-templates), rendered per request into a duration.
Three helpers cover the usual cases:

::: v-pre
```yaml
output:
  delay: '{{ regressive .AttemptNumber "3s" "500ms" }}'   # 3s, 2.5s, 2s ... 0
  delay: '{{ backoff .AttemptNumber "100ms" "5s" }}'      # 100ms, 200ms, 400ms ... 5s
  delay: '{{ jitter "50ms" "250ms" }}'                    # random in the range
```
:::

`.AttemptNumber` counts matches of this stub per session, from 1, and is capped by `options.times`.
`regressive` never drops below zero, `backoff` doubles and stops at the cap (the cap is optional).
Single-quote the YAML value — then the durations inside need no escaping.

Anything else is plain template math over milliseconds, wrapped in `duration`:

::: v-pre
```yaml
delay: '{{ duration (mul 100 .MessageIndex) }}'   # ramp over an output.stream array
delay: '{{ index .Headers "x-delay" }}'          # client dictates the pause
```
:::

Empty or negative result means no pause. A broken template is rejected on registration with `400`;
a result that is not a duration fails the call with `Internal`.

### Custom Curves

A [plugin](../plugins/) can add its own curve — any function returning a duration string works:

```go
func Register(reg plugins.Registry) {
	reg.AddPlugin(
		plugins.PluginInfo{Name: "delay", Kind: "external", Capabilities: []string{"template-funcs"}},
		[]plugins.SpecProvider{plugins.Specs(
			plugins.FuncSpec{Name: "fibonacci", Fn: fibonacci},
		)},
	)
}
```

::: v-pre
```yaml
output:
  delay: '{{ fibonacci .AttemptNumber "60ms" }}'   # 60ms, 60ms, 120ms, 180ms, 300ms
```
:::

Full example: `examples/plugins/delay`. A plugin function reusing a built-in name is ignored unless it
declares `Decorates: "@gripmock/<name>"`, which wraps the original instead of replacing it.

## Use Cases

### 1. Timeout Testing
```yaml
output:
  delay: 5s
  data:
    message: "Slow response"
```
Use long delays to test client timeout handling.

### 2. Realistic Network Simulation
```yaml
output:
  delay: 150ms
  data:
    message: "Typical network latency"
```
Simulate realistic network conditions for performance testing.

### 3. Rate Limiting Simulation
```yaml
output:
  delay: 1s
  data:
    message: "Rate limited response"
```
Test client behavior under rate limiting scenarios.

## Error Responses with Delay

Delay can be combined with error responses:

```yaml
output:
  delay: 500ms
  error: "Service temporarily unavailable"
  code: 14  # UNAVAILABLE
```

## Delay Logic

### Unary Calls
- Delay is applied **before** sending the single response
- Total delay = configured delay value

### Streaming Calls  
- Delay is applied **before** every message in the stream
- Total delay = number of messages × configured delay

### Examples

#### Unary Response (1 response)
```yaml
output:
  delay: 200ms
  data:
    message: "Hello"
```
**Timing**: 200ms delay → response sent

#### Streaming Response (3 messages)
```yaml
output:
  delay: 200ms
  stream:
    - message: "First"
    - message: "Second" 
    - message: "Third"
```
**Timing**:
- 200ms delay
- Message 1 sent
- 200ms delay
- Message 2 sent
- 200ms delay
- Message 3 sent

**Total delay**: 600ms (3 × 200ms)

## Limitations

- Maximum delay is limited by client timeout settings
- Delay affects all response types (data, error, stream)
- Delay is applied consistently across all gRPC call types

## Verification

You can verify delay behavior using gRPC clients or tools like `grpcurl`:

```bash
# Test with delay
grpcurl -plaintext -d '{"stn":"MS#00001"}' localhost:4770 TrackService/StreamTrack
```

The response time should include the configured delay plus processing time. 