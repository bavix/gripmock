# GripMock — CLAUDE.md

## What this repo is

gripmock is a gRPC mock server. It reads `.proto` files (or pre-compiled `.pb` descriptors),
loads stub YAML/JSON files, and responds to gRPC (and ConnectRPC) calls by matching incoming
requests against stubs stored in memory.

Module path: `github.com/bavix/gripmock/v3`

---

## Ports / servers

| Server | Default port | Env vars | Notes |
|---|---|---|---|
| gRPC | 4770 | `GRPC_HOST`, `GRPC_PORT`, `GRPC_NETWORK` | pure gRPC, HTTP/2 |
| REST stub API | 4771 | `HTTP_HOST`, `HTTP_PORT` | manages stubs, history, descriptors |
| ConnectRPC | 4772 | `CONNECT_HOST`, `CONNECT_PORT` | unary only; added in this fork |

TLS for each: `GRPC_TLS_*`, `HTTP_TLS_*` (no `CONNECT_TLS_*` yet).

---

## Build / test

```bash
# Build (use explicit paths — `./...` fails on an unrelated internal test binary)
go build ./internal/... ./cmd/...

# Test
go test ./internal/... ./cmd/... -count=1 -timeout 60s

# Lint
go vet ./internal/... ./cmd/...
```

---

## Key packages

| Package | Role |
|---|---|
| `cmd/` | Cobra CLI; starts all three servers |
| `internal/config/` | Env-based config (`config.Config`); all ports and flags here |
| `internal/app/` | Core server logic — gRPC handler, REST handler, ConnectRPC handler |
| `internal/deps/` | Dependency wiring (`Builder`); one `*Serve` method per server |
| `internal/infra/stuber/` | In-memory stub storage (`Budgerigar`) and request matching |
| `internal/domain/descriptors/` | Dynamic proto descriptor registry (runtime-registered protos) |
| `internal/domain/history/` | Call history recorder and reader |
| `internal/infra/template/` | Go-template engine for dynamic stub responses |
| `internal/infra/storage/` | Stub file loading/watching from disk |

---

## Request flow (gRPC and ConnectRPC)

```
Request arrives
  → find method descriptor (protoregistry.GlobalFiles OR descriptors.Registry)
  → decode request body → dynamicpb.Message → convertToMap()
  → stuber.Query{Service, Method, Input, Headers, Session}
  → budgerigar.FindByQuery(query) → *stuber.Result
  → template.Engine.ProcessMap(output.Data, templateData)
  → encode response → send
```

The stub matcher (`FindByQuery`) supports: `equals`, `contains`, `matches` (regex), `glob`,
`anyOf` — on both input fields and headers. Priority + specificity rank ties.

---

## Key files to know

| File | What it does |
|---|---|
| `internal/app/grpc_server.go` | gRPC handler; `handleUnknownService` is the entry point for all mock calls |
| `internal/app/connect_handler.go` | ConnectRPC HTTP handler (added in this session) |
| `internal/app/output_runtime_common.go` | `outputStatusBase()`, `delayResponse()` — shared by gRPC and Connect |
| `internal/app/utils_copy.go` | `deepCopyMapAny`, `deepCopyStringMap` — shared helpers |
| `internal/infra/stuber/stub.go` | `Stub` struct — the canonical stub schema |
| `internal/infra/stuber/query.go` | `Query` struct — what FindByQuery receives |
| `internal/deps/rest_server.go` | REST server setup — reference when adding middleware |
| `internal/deps/connect_server.go` | ConnectRPC server setup (added in this session) |

---

## ConnectRPC handler (added in this fork)

- **File**: `internal/app/connect_handler.go`
- **URL format**: `POST /{package.ServiceName}/{MethodName}`
- **Content-Type**: `application/proto` (binary) or `application/json`
- **Gzip**: transparent decompression via `Content-Encoding: gzip`
- **Session**: `X-Gripmock-Session` header (same key as gRPC metadata)
- **Streaming**: supported via Connect streaming protocol (`application/connect+proto/json`)
  - **Server streaming**: read one framed request → find stub → send `output.stream` items as
    data frames → send end-stream `{}`
  - **Client streaming**: read all framed requests until EOF → find stub with all inputs →
    send one framed response + end-stream
  - **Bidirectional**: uses `FindByQueryBidi`/`BidiResult.Next()` per request frame; works
    sequentially (read one request → send response(s) → repeat); requires HTTP/2 for true
    full-duplex but works sequentially over HTTP/1.1 too
  - Streaming errors: HTTP 200 + end-stream frame with `{"error":{"code":"…","message":"…"}}`
  - Descriptor required for streaming (no descriptor-less fallback)
  - Frame format: 5-byte header `[flags:1][length:4 big-endian]`; flag `0x01` = gzip compressed,
    flag `0x02` = end-stream; per-frame gzip decompression is transparently handled
- **Unary error format**: `{"code": "not_found", "message": "..."}` + appropriate HTTP status
- Shares `budgerigar`, `descriptors.Registry`, history recorder with gRPC server
- Reuses `jsonBufferPool`, `convertToMap`, `deepCopyMapAny`, `outputStatusBase`,
  `findMethodInGlobalFiles`, `template.Engine`, `delayResponse` from same `app` package

---

## Stub YAML format (quick reference)

```yaml
- service: package.ServiceName
  method: MethodName
  input:
    equals:
      field: value
  output:
    data:
      responseField: value
    headers:
      x-custom: header-value
    error: ""        # non-empty triggers error response
    code: null       # gRPC code override
    delay: 0         # nanoseconds
```

### Validation rules (XOR logic — exactly one side must be non-empty)

**Input**: must have EITHER `input` (with at least one non-nil matcher: `equals`, `contains`,
`matches`, or `glob`) OR `inputs` (array, for client-streaming). An empty `input:` or `input: {}`
without any matcher key fails validation.

**Output**: must have EITHER data-type output (`data` non-nil, OR `error` non-empty, OR `code`
set, OR `details` non-empty) OR `stream` (array). An empty `data:` (null) fails validation.

**To match any request and return an empty proto response (e.g., `message Foo {}`):**
```yaml
- service: package.ServiceName
  method: MethodName
  input:
    equals: {}       # empty map = match any request, {} makes Equals non-nil
  output:
    data: {}         # empty map = empty proto message, {} makes Data non-nil
```

---

## Adding a new server (pattern)

1. Add `FooHost/FooPort/FooAddr` fields to `internal/config/config.go`
2. Create `internal/app/foo_handler.go` with the handler
3. Create `internal/deps/foo_server.go` with `(b *Builder) FooServe(ctx) error`
4. Add goroutine in `cmd/root.go` matching the REST/Connect pattern

---

## Important env vars

```
LOG_LEVEL            info
GRPC_PORT            4770
HTTP_PORT            4771
CONNECT_PORT         4772
STUB_WATCHER_TYPE    fsnotify | timer
STUB_WATCHER_INTERVAL 1s
HISTORY_ENABLED      true
HISTORY_LIMIT        64M
OTEL_ENABLED         false
```
