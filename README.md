![GripMock](https://github.com/bavix/gripmock/assets/5111255/d1fc10ef-2149-4302-8e24-aef4fdfe043c)

[![Coverage Status](https://coveralls.io/repos/github/bavix/gripmock/badge.svg?branch=master)](https://coveralls.io/github/bavix/gripmock?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/bavix/gripmock/v3)](https://goreportcard.com/report/github.com/bavix/gripmock/v3)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

# GripMock 🚀

**Languages:** English | [简体中文](README.zh-CN.md)

**The fastest and most reliable gRPC mock server** for testing and development.

GripMock creates a mock server from your `.proto` files or compiled `.pb` descriptors, making gRPC testing simple and efficient. Perfect for end-to-end testing, development environments, and CI/CD pipelines.

![greeter](https://raw.githubusercontent.com/bavix/.github/master/svgs/gripmock-greeter.gif)

## ✨ Features

- **Native Runtime** - Single in-process engine without runtime gRPC code generation
- **Descriptor Sources** - Load API from `.proto`, compiled `.pb`, BSR modules, or gRPC reflection
- **Dynamic `.pb` Service Loading** - Load compiled protobuf descriptors at runtime via API without restarts
- **Hot Stub Management** - Create, update, and remove stubs via API/UI without server restarts
- **Flexible Matching** - `equals`, `contains`, `matches`, `glob`, headers, priority, and match limits
- **Array-Aware Matching** - Optional array-order flexibility to reduce brittle test assertions
- **Dynamic Templates** - Build responses from request payload, headers, and stream context
- **Complete gRPC Coverage** - Unary, server streaming, client streaming, and bidirectional streaming
- **Error, Details, and Delay Simulation** - Return realistic gRPC status codes, details (`Any`), and response timing
- **TLS and mTLS Support** - Run secure gRPC/HTTP test environments with native TLS options
- **Advanced Protobuf Type Support** - Handle well-known and extended protobuf types (`google.protobuf.*`, `google.type.*`)
- **YAML/JSON + Schema** - Author stubs in either format with JSON Schema IDE validation
- **Plugin Ecosystem** - Extend functions with Go plugins and matching builder image tags
- **Built-in Faker Templates** - Generate realistic fake person/contact/geo/network data directly in templates (`faker.*`)
- **OpenTelemetry Tracing** - OTLP tracing for gRPC and HTTP paths (`otelgrpc` + `otelhttp`)
- **Prometheus Metrics (`/metrics`)** - Runtime/process metrics (`go_*`, `process_*`) plus GripMock metrics
- **Operational APIs** - Health endpoints, descriptors API, stubs API, and web dashboard
- **Embedded SDK (Experimental)** - Run GripMock inside Go tests/services with verification helpers
- **MCP API (Experimental)** - Streamable MCP endpoint for agent and tool integrations
- **Upstream Modes (Experimental)** - `proxy`, `replay`, `capture` modes for gradual migration from live upstream services to local mocks

## 📚 Documentation

**[Full Documentation](https://bavix.github.io/gripmock)** - Complete guide with examples

- **Descriptor API (`/api/descriptors`)**: runtime loading of compiled proto descriptors (`.pb`) with validated curl workflow: [docs](https://bavix.github.io/gripmock/guide/api/descriptors)
- **Upstream Modes (Experimental)**: `proxy`, `replay`, `capture` with practical rollout guidance: [docs](https://bavix.github.io/gripmock/guide/modes)
- **Embedded SDK (Experimental)**: in-process testing with stubs, verification, `sdk.By(fullMethod)` helpers, and context-aware remote checks: [docs](https://bavix.github.io/gripmock/guide/embedded-sdk)
- **Faker Reference**: built-in faker key-by-key catalog with examples: [docs](https://bavix.github.io/gripmock/guide/stubs/faker)
- **OpenTelemetry + Metrics**: tracing env vars and `/metrics` behavior: [docs](https://bavix.github.io/gripmock/guide/introduction/advanced-usage)
- **GitHub Actions (CI/CD)**: official workflow action to download, start, wait for readiness, and stop GripMock automatically: [docs](https://bavix.github.io/gripmock/guide/ci-cd/github-actions)

## 🧬 Project Evolution

GripMock started as a fork of [tokopedia/gripmock](https://github.com/tokopedia/gripmock), and then evolved into an independent, fully rewritten project.

Today GripMock is an independent runtime focused on practical testing workflows:

- Native in-process architecture (no runtime code generation)
- Flexible descriptor sources and runtime operations (hot stubs + descriptors API)
- Production-style testing features (streaming, templates, upstream modes, plugins, SDK, MCP)

For architecture details and benchmark methodology, see: [Performance Comparison](https://bavix.github.io/gripmock/guide/introduction/performance-comparison)

## 🖥️ Web Interface

![gripmock-ui](https://raw.githubusercontent.com/bavix/.github/master/svgs/gripmock-ui.gif)

Access the web dashboard at `http://localhost:4771/` to manage your stubs visually.

## 🚀 Quick Start

### Installation

Choose your preferred installation method:

#### Homebrew (Recommended)
```bash
brew tap gripmock/tap
brew install --cask gripmock
```

#### Shell Script
```bash
curl -s https://raw.githubusercontent.com/bavix/gripmock/refs/heads/master/setup.sh | sh -s
```

#### PowerShell (Windows)
```powershell
irm https://raw.githubusercontent.com/bavix/gripmock/refs/heads/master/setup.ps1 | iex
```

#### Docker
```bash
docker pull bavix/gripmock
```

For plugin builds, use the paired builder image:

```bash
docker pull bavix/gripmock:v3.7.1-builder
```

#### Go Install
```bash
go install github.com/bavix/gripmock/v3@latest
```

### Basic Usage

**Start with a `.proto` file:**
```bash
gripmock service.proto
```

**Add static stubs:**
```bash
gripmock --stub stubs/ service.proto
```

**Load API directly from Buf Schema Registry (BSR):**
```bash
gripmock --stub third_party/bsr/eliza buf.build/connectrpc/eliza
```

**Load API from live gRPC server reflection:**
```bash
gripmock grpc://localhost:50051
gripmock grpcs://api.company.local:443
```

With options:
```bash
gripmock grpc://localhost:50051?timeout=10s
gripmock grpcs://10.0.0.5:8443?serverName=api.company.local
gripmock grpc://localhost:50051?bearer=<token>
```

**Use upstream modes over reflection (Experimental):**
```bash
# Pure reverse proxy through GripMock
gripmock grpc+proxy://localhost:50051

# Local stubs first, then upstream fallback on matcher miss
gripmock grpc+replay://localhost:50051

# Replay + record upstream misses into GripMock stubs
gripmock grpc+capture://localhost:50051
```

For private BSR modules:
```bash
BSR_BUF_TOKEN=<token> gripmock --stub stubs/ buf.build/acme/private-api
```

For self-hosted BSR:
```bash
BSR_SELF_BASE_URL=https://bsr.company.local \
BSR_SELF_TOKEN=<token> \
gripmock --stub stubs/ bsr.company.local/team/payments
```

**Using Docker:**
```bash
docker run -p 4770:4770 -p 4771:4771 \
  -v $(pwd)/stubs:/stubs \
  -v $(pwd)/proto:/proto \
  bavix/gripmock --stub=/stubs /proto/service.proto
```

- **Port 4770**: gRPC server
- **Port 4771**: Web UI and REST API

### Observability (v3.10.0)

```bash
OTEL_ENABLED=true \
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
OTEL_EXPORTER_OTLP_INSECURE=true \
gripmock --stub stubs/ service.proto
```

- `GET /metrics` is always available
- Tracing export is enabled only when `OTEL_ENABLED=true`

## 🤖 GitHub Actions (CI/CD)

Use the official action [`bavix/gripmock-action`](https://github.com/bavix/gripmock-action) to run GripMock in CI pipelines.

```yaml
name: test

on: [push, pull_request]

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5

      - name: Start GripMock
        uses: bavix/gripmock-action@v1
        with:
          source: proto/service.proto
          stub: stubs

      - name: Run tests
        run: go test ./...
```

What the action does:

- Downloads GripMock from GitHub Releases (`latest` or pinned `version`)
- Starts GripMock in background and waits for readiness (`/api/health/readiness`)
- Exposes addresses via outputs (`grpc-addr`, `http-addr`) for test steps
- Stops GripMock automatically in the post step

More examples and full inputs/outputs: [GitHub Actions guide](https://bavix.github.io/gripmock/guide/ci-cd/github-actions).

## 📖 Examples

Check out our comprehensive examples in the [`examples`](https://github.com/bavix/gripmock/tree/master/examples) folder:

- **Streaming** - Server, client, and bidirectional streaming
- **File Uploads** - Test chunked file uploads
- **Real-time Chat** - Bidirectional communication
- **Data Feeds** - Continuous data streaming
- **Authentication** - Header-based auth testing
- **Performance** - High-throughput scenarios

### Greeter: dynamic stub demo

Stub (universal):

```yaml
# yaml-language-server: $schema=https://bavix.github.io/gripmock/schema/stub.json
# examples/projects/greeter/stub_say_hello.yaml
- service: helloworld.Greeter
  method: SayHello
  input:
    matches:
      name: ".+"
  output:
    data:
      message: "Hello, {{.Request.name}}!"  # dynamic template lives in output
```

Notes:
- Put dynamic templates only in `output` (e.g., `data`, `headers`, `stream`).
- Keep `input` matching static (no `{{ ... }}` in `equals`/`contains`/`matches`).

```bash
# Start server
go run main.go examples/projects/greeter/service.proto --stub examples/projects/greeter

# Call via grpcurl
grpcurl -plaintext -d '{"name":"Alex"}' localhost:4770 helloworld.Greeter/SayHello
```

Expected response:

```json
{
  "message": "Hello, Alex!"
}
```

## 🔧 Stubbing

### Basic Stub Example

```yaml
service: Greeter
method: SayHello
input:
  equals:
    name: "gripmock"
output:
  data:
    message: "Hello GripMock!"
```

### Advanced Features

**Priority System:**
```yaml
- service: UserService
  method: GetUser
  priority: 100  # Higher priority
  input:
    equals:
      id: "admin"
  output:
    data:
      role: "administrator"

- service: UserService
  method: GetUser
  priority: 1    # Lower priority (fallback)
  input:
    contains:
      id: "user"
  output:
    data:
      role: "user"
```

**Streaming Support:**
```yaml
service: TrackService
method: StreamData
input:
  equals:
    sensor_id: "GPS001"
output:
  stream:
    - position: {"lat": 40.7128, "lng": -74.0060}
      timestamp: "2024-01-01T12:00:00Z"
    - position: {"lat": 40.7130, "lng": -74.0062}
      timestamp: "2024-01-01T12:00:05Z"
```

### Dynamic Templates

GripMock supports dynamic templates in the `output` section using Go's `text/template` syntax.

- Access request fields: `{{.Request.field}}`
- Access headers: `{{.Headers.header_name}}`
- Client streaming context: `{{.Requests}}` (slice of received messages), `{{len .Requests}}`, `{{(index .Requests 0).field}}`
- Bidirectional streaming: `{{.MessageIndex}}` gives the current message index (0-based)
- Math helpers: `sum`, `avg`, `mul`, `min`, `max`, `add`, `sub`, `div`
- Utility: `json`, `split`, `join`, `upper`, `lower`, `title`, `sprintf`, `int`, `int64`, `float`, `round`, `floor`, `ceil`
- Built-in faker: `faker.Person.*`, `faker.Contact.*`, `faker.Geo.*`, `faker.Network.*`, `faker.Identity.*`

Important rules:
- Do not use dynamic templates inside `input.equals`, `input.contains`, or `input.matches` (matching must be static)
- For server streaming, if both `output.stream` and `output.error`/`output.code` are set, messages are sent first and then the error is returned. If `output.stream` is empty, the error is returned immediately

**Header Matching:**
```yaml
service: AuthService
method: ValidateToken
headers:
  equals:
    authorization: "Bearer valid-token"
input:
  equals:
    token: "abc123"
output:
  data:
    valid: true
    user_id: "user123"
```

## 🔍 Input Matching

GripMock supports four powerful matching strategies:

### 1. Exact Match (`equals`)
```yaml
input:
  equals:
    name: "gripmock"
    age: 25
    active: true
```

### 2. Partial Match (`contains`)
```yaml
input:
  contains:
    name: "grip"  # Matches "gripmock", "gripster", etc.
```

### 3. Regex Match (`matches`)
```yaml
input:
  matches:
    email: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
    phone: "^\\+?[1-9]\\d{1,14}$"
```

### 4. Glob Match (`glob`)
```yaml
input:
  glob:
    filename: "*.txt"
    path: "/usr/local/*"
```

## 🛠️ API

### REST API Endpoints

- `GET /api/stubs` - List all stubs
- `POST /api/descriptors` - Load protobuf descriptor set (`FileDescriptorSet`) at runtime
- `POST /api/stubs` - Add new stub
- `POST /api/stubs/search` - Find matching stub
- `DELETE /api/stubs` - Clear all stubs
- `GET /api/health/liveness` - Health check
- `GET /api/health/readiness` - Readiness check

### Example API Usage

```bash
# Add a stub
curl -X POST http://localhost:4771/api/stubs \
  -H "Content-Type: application/json" \
  -d '{
    "service": "Greeter",
    "method": "SayHello",
    "input": {"equals": {"name": "world"}},
    "output": {"data": {"message": "Hello World!"}}
  }'

# Search for matching stub
curl -X POST http://localhost:4771/api/stubs/search \
  -H "Content-Type: application/json" \
  -d '{
    "service": "Greeter",
    "method": "SayHello",
    "data": {"name": "world"}
  }'
```

## 📋 JSON Schema Support

Add schema validation to your stub files for IDE support:

**JSON files:**
```json
{
  "$schema": "https://bavix.github.io/gripmock/schema/stub.json",
  "service": "MyService",
  "method": "MyMethod"
}
```

**YAML files:**
```yaml
# yaml-language-server: $schema=https://bavix.github.io/gripmock/schema/stub.json
service: MyService
method: MyMethod
```

## 🌐 BSR Integration

GripMock supports simplified integration with Buf Schema Registry:

### Configuration

```bash
# Public BSR (default)
BSR_BUF_BASE_URL=https://buf.build
BSR_BUF_TOKEN=<token>

# Self-hosted BSR
BSR_SELF_BASE_URL=https://bsr.company.local
BSR_SELF_TOKEN=<token>
```

### Usage

```bash
# Public module
gripmock buf.build/connectrpc/eliza

# Self-hosted module  
gripmock bsr.company.local/team/payments:main

# With stubs
gripmock --stub stubs/ bsr.company.local/team/payments
```

### Routing

GripMock automatically routes modules:
- `buf.build/owner/repo` → uses Buf profile
- `bsr.company.local/owner/repo` → uses Self profile

For details see [BSR Documentation](https://bavix.github.io/gripmock/guide/sources/bsr).

## 🔎 gRPC Reflection Source

GripMock supports descriptor loading from gRPC reflection using endpoint schemes:

- `grpc://host:port` (insecure)
- `grpcs://host:port` (TLS)

Supported query parameters:

- `timeout` (default `5s`)
- `bearer` (Authorization token)
- `serverName` (TLS SNI override)

Examples:

```bash
gripmock grpc://localhost:50051
gripmock grpcs://api.company.local:443
gripmock grpcs://10.0.0.5:8443?serverName=api.company.local
```

Full guide: [gRPC Reflection Source](https://bavix.github.io/gripmock/guide/sources/grpc-reflection).

## 🔁 Upstream Modes (Experimental)

⚠️ **EXPERIMENTAL FEATURE**: Upstream modes may change without notice.

Upstream modes work on top of reflection sources and define runtime behavior:

- `proxy` - pure reverse proxy
- `replay` - local-first + upstream fallback
- `capture` - replay + automatic stub recording from upstream

Mode guides:

- [Upstream Modes Overview](https://bavix.github.io/gripmock/guide/modes)
- [Proxy Mode](https://bavix.github.io/gripmock/guide/modes/proxy)
- [Replay Mode](https://bavix.github.io/gripmock/guide/modes/replay)
- [Capture Mode](https://bavix.github.io/gripmock/guide/modes/capture)

## 📊 Benchmark Charts

![Image size benchmark](docs/public/bench/image-size.svg)
![Startup readiness benchmark](docs/public/bench/startup-ready.svg)
![Latency percentiles benchmark](docs/public/bench/latency-percentiles.svg)
![Throughput benchmark](docs/public/bench/throughput-rps.svg)

## 🔗 Useful Resources

- 📖 **[Documentation](https://bavix.github.io/gripmock)** - Complete guides and examples
- 🧪 **[Testing gRPC with Testcontainers](https://medium.com/skyro-tech/testing-grpc-client-with-mock-server-and-testcontainers-f51cb8a6be9a)** - Article by [@AndrewIISM](https://github.com/AndrewIISM)
- 📋 **[JSON Schema](https://bavix.github.io/gripmock/schema/stub.json)** - Stub validation schema
- 🔗 **[OpenAPI](https://bavix.github.io/gripmock-openapi/)** - REST API documentation

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## 📄 License

This project is licensed under the **MIT License** - see the [LICENSE](LICENSE) file for details.

---

**Made with ❤️ by the GripMock community**
