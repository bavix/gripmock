# History API

Every gRPC call GripMock serves is recorded: the request and response messages, the status
code, the elapsed time and the session it belonged to. The History page of the admin UI, the
`verify` endpoint and the MCP history tools all read this store.

Recording can be turned off with `HISTORY_ENABLED=false`, in which case the endpoints below
answer with an empty list and `verify` always reports `0` actual calls.

## List calls

- **Method**: `GET`
- **URL**: `/api/history`

| Parameter | Meaning |
|---|---|
| `limit` | Return at most N most-recent records |
| `offset` | Skip the N newest records — page backwards through older calls |
| `service` | Keep only calls to this fully qualified service |
| `method` | Keep only calls to this method |
| `error` | `true` returns only calls that ended with a gRPC error |

The response carries `X-Total-Count`: the number of matches **before** pagination, so a client
can tell how far back it can page.

```bash
# The 20 most recent calls
curl 'http://127.0.0.1:4771/api/history?limit=20'

# Only the failures of one endpoint
curl 'http://127.0.0.1:4771/api/history?service=helloworld.Greeter&method=SayHello&error=true'
```

Each record looks like this:

```json
{
  "service": "helloworld.Greeter",
  "method": "SayHello",
  "session": "team-a",
  "stubId": "3ba04b6d-49e7-480e-a08e-e504977a1c07",
  "code": 0,
  "requests": [{"name": "Alex"}],
  "responses": [{"message": "Hello Alex"}],
  "elapsedMs": 1,
  "timestamp": "2026-08-19T09:20:55Z"
}
```

`requests` and `responses` are arrays: a unary call carries exactly one of each, a streaming
call carries one entry per message. `stubId` is absent when no stub matched — that record is
the evidence of a miss, and `code` tells you it failed.

## Session scope

`X-Gripmock-Session` narrows the list to that session's calls **plus the global ones** — the
same scope its stubs match in.

```bash
curl -H 'X-Gripmock-Session: team-a' http://127.0.0.1:4771/api/history
```

Without the header you get the operator view: every call, including those made under a
session. Note the asymmetry with stubs, where an unscoped request sees only global stubs. It is
deliberate — an operator watching a shared server wants to see all traffic — but it means an
unscoped `POST /api/verify` counts other sessions' calls too.

## Purge calls

- **Method**: `DELETE`
- **URL**: `/api/history`

```bash
# Everything
curl -X DELETE http://127.0.0.1:4771/api/history

# Only one session's calls
curl -X DELETE -H 'X-Gripmock-Session: team-a' http://127.0.0.1:4771/api/history
```

```json
{"deletedCount": 2, "session": "team-a"}
```

With `X-Gripmock-Session` only that session's records are removed, so a parallel session keeps
its evidence; without it the whole history is cleared. `session` is absent from the response
when the purge was unscoped.

The admin UI exposes both forms: **Clear** on the History page purges the current scope, and
the trash button next to a session on the Session page drops that session's stubs and calls
together.

## Limits

History is bounded by bytes, not by record count:

| Variable | Default | Meaning |
|---|---|---|
| `HISTORY_ENABLED` | `true` | Turns recording off entirely |
| `HISTORY_LIMIT` | `64M` | Total budget; the oldest records are evicted first |
| `HISTORY_MESSAGE_MAX_BYTES` | `262144` | A larger message is replaced with `{"_truncated": true}` |
| `HISTORY_REDACT_KEYS` | — | Comma-separated field names whose values become `[REDACTED]` |

Redaction matches by key name at any depth and applies on write, so a secret never reaches the
store — but only for the keys you name, and a value copied into a differently named field is
not covered.

## Related

- [Verify API](./verify) — assert a call count over this same store
- [MCP API](./mcp/) — `history_list`, `history_errors` and `history_purge` do the same over MCP
