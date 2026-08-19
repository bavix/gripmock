# Verify API

Asserts that an endpoint was called an exact number of times. It reads the same store as the
[History API](./history), so it needs `HISTORY_ENABLED` (the default); with recording off every
check reports `0` actual calls.

## Request

- **Method**: `POST`
- **URL**: `/api/verify`

```bash
curl -X POST http://127.0.0.1:4771/api/verify \
  -H 'Content-Type: application/json' \
  -d '{"service":"helloworld.Greeter","method":"SayHello","expectedCount":3}'
```

## Response

`200` when the count matches:

```json
{"message": "ok", "time": "2026-08-19T09:20:55Z"}
```

`400` when it does not — the body carries both numbers, so a script can report the difference
without re-reading the history:

```json
{
  "expected": 3,
  "actual": 7,
  "message": "expected helloworld.Greeter/SayHello to be called 3 times, got 7"
}
```

The comparison is **exact**: `expectedCount: 3` fails on two calls and on four. Use
`GET /api/history?service=…&method=…` when you need a range or just the evidence.

## Session scope

`X-Gripmock-Session` counts that session's calls plus the global ones — the same scope its
stubs match in:

```bash
curl -X POST http://127.0.0.1:4771/api/verify \
  -H 'Content-Type: application/json' \
  -H 'X-Gripmock-Session: team-a' \
  -d '{"service":"helloworld.Greeter","method":"SayHello","expectedCount":2}'
```

Without the header every call is counted, **including calls made under a session**. On a shared
server that makes an unscoped assertion sensitive to a colleague's traffic: give each test its
own session and the count becomes stable.

## Counting model

Verify counts per **service and method**. The embedded SDK's `ExpectationsWereMet` counts per
**stub**, which is why two stubs on one method (say `Once()` and `Twice()`) are checked
independently there but add up here.

## Related

- [History API](./history) — the records behind the count
- [MCP API](./mcp/) — `verify_calls` is the same check over MCP
- [Embedded SDK verification](../embedded-sdk/verification) — per-stub expectations in Go tests
