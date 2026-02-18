![GripMock](https://github.com/bavix/gripmock/assets/5111255/d1fc10ef-2149-4302-8e24-aef4fdfe043c)

[![Coverage Status](https://coveralls.io/repos/github/bavix/gripmock/badge.svg?branch=master)](https://coveralls.io/github/bavix/gripmock?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/bavix/gripmock/v3)](https://goreportcard.com/report/github.com/bavix/gripmock/v3)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

# GripMock 🚀

**The fastest and most reliable gRPC mock server** for testing and development.

GripMock creates a mock server from your `.proto` files or compiled `.pb` descriptors, making gRPC testing simple and efficient. Perfect for end-to-end testing, development environments, and CI/CD pipelines.

## ✨ Features

- 🚀 **Instant Setup** - Create a working gRPC server in seconds
- 📝 **YAML & JSON Support** - Define stubs in your preferred format
- 🔄 **All Streaming Types** - Unary, server, client, and bidirectional streaming
- ⚡ **20-35% Faster** - Enhanced performance for quicker tests
- 🔒 **100% Backward Compatible** - All existing tests continue to work
- 🐳 **Docker Ready** - Lightweight container for CI/CD
- 🧱 **Builder Image** - Build Go plugins with `bavix/gripmock:<tag>-builder` for runtime compatibility
- 🖥️ **Web Interface** - Manage stubs through a friendly dashboard
- 📋 **JSON Schema** - Full IDE support with validation
- ❤️ **Health Checks** - Production-ready monitoring endpoints
- 🔌 **Plugin System** - Extend template functions with custom plugins
- 🧪 **Embedded SDK (Experimental)** - Run GripMock inside Go tests and services
- 🧪 **MCP API (Experimental)** - Use Model Context Protocol endpoints for tool integration

## 📚 Documentation

📖 **[Full Documentation](https://bavix.github.io/gripmock/)** - Complete guide with examples

- **Descriptor API (`/api/descriptors`)**: runtime loading of compiled proto descriptors (`.pb`) with validated curl workflow: [docs](https://bavix.github.io/gripmock/guide/api/descriptors)

## 🆚 Why Choose Our Fork?

This service is a fork of [tokopedia/gripmock](https://github.com/tokopedia/gripmock), but you should choose our fork. Here's why:

### 🆕 New Features
- ✅ **YAML support** as JSON alternative for static stubs
- ✅ **Health check endpoints** (`/api/health/liveness`, `/api/health/readiness`)
- ✅ **Header matching** support for authentication testing
- ✅ **gRPC error codes** for realistic error simulation
- ✅ **Priority system** for controlling stub matching order
- ✅ **Binary descriptor support** (`.pb` files) for faster startup
- ✅ **Array streaming** for server streaming methods
- ✅ **JSON Schema validation** with IDE support
- ✅ **Enhanced performance** with 20-35% speed improvements
- ✅ **Plugin system** for extending template functions

### 🔧 Improvements
- ✅ **Updated dependencies** - All deprecated packages fixed
- ✅ **Reduced image size** - Optimized Docker containers
- ✅ **Better error handling** - 404 errors for missing stubs
- ✅ **Active maintenance** - Regular updates and bug fixes
- ✅ **Comprehensive documentation** - Complete guides and examples

## 🖥️ Web Interface

![gripmock-ui](https://github.com/bavix/gripmock/assets/5111255/3d9ebb46-7810-4225-9a30-3e058fa5fe16)

Access the web dashboard at `http://localhost:4771/` to manage your stubs visually.

## 🚀 Quick Start

### Installation

Choose your preferred installation method:

#### 🍺 Homebrew (Recommended)
```bash
brew tap gripmock/tap
brew install gripmock
```

#### 📦 Shell Script
```bash
curl -s https://raw.githubusercontent.com/bavix/gripmock/refs/heads/master/setup.sh | sh -s
```

#### 🪟 PowerShell (Windows)
```powershell
irm https://raw.githubusercontent.com/bavix/gripmock/refs/heads/master/setup.ps1 | iex
```

#### 🐳 Docker
```bash
docker pull bavix/gripmock
```

For plugin builds, use the paired builder image:

```bash
docker pull bavix/gripmock:v3.7.1-builder
```

#### 🔧 Go Install
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

**Using Docker:**
```bash
docker run -p 4770:4770 -p 4771:4771 \
  -v $(pwd)/stubs:/stubs \
  -v $(pwd)/proto:/proto \
  bavix/gripmock --stub=/stubs /proto/service.proto
```

- **Port 4770**: gRPC server
- **Port 4771**: Web UI and REST API

## 📖 Examples

Check out our comprehensive examples in the [`examples`](https://github.com/bavix/gripmock/tree/master/examples) folder:

- 🔄 **Streaming** - Server, client, and bidirectional streaming
- 📁 **File Uploads** - Test chunked file uploads
- 💬 **Real-time Chat** - Bidirectional communication
- 📊 **Data Feeds** - Continuous data streaming
- 🔐 **Authentication** - Header-based auth testing
- ⚡ **Performance** - High-throughput scenarios

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

GripMock supports three powerful matching strategies:

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

## 🔗 Useful Resources

- 📖 **[Documentation](https://bavix.github.io/gripmock/)** - Complete guides and examples
- 🧪 **[Testing gRPC with Testcontainers](https://medium.com/skyro-tech/testing-grpc-client-with-mock-server-and-testcontainers-f51cb8a6be9a)** - Article by [@AndrewIISM](https://github.com/AndrewIISM)
- 📋 **[JSON Schema](https://bavix.github.io/gripmock/schema/stub.json)** - Stub validation schema
- 🔗 **[OpenAPI](https://bavix.github.io/gripmock-openapi/)** - REST API documentation

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## 📄 License

This project is licensed under the **MIT License** - see the [LICENSE](LICENSE) file for details.

---

**Made with ❤️ by the GripMock community**
