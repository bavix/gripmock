# Quick Usage

## Installation

### 1. Homebrew (macOS and Linux) <VersionTag version="v3.2.5" />

```bash
brew tap gripmock/tap
brew install --cask gripmock
gripmock --version
```

The `gripmock` cask is built with cgo, so [plugins](/guide/plugins/) load. The
`gripmock-slim` cask is the same server without plugin support; the two casks
conflict, install one.

### 2. Shell script (curl)

Linux and macOS, arm64 and amd64:

```bash
curl -s https://raw.githubusercontent.com/bavix/gripmock/refs/heads/master/setup.sh | sh -s
```

Append `-- --slim` for the build without plugin support. On musl (Alpine) the
slim build is installed automatically, since the default one links glibc.

This script automatically:
1. Detects your system (Linux/macOS) and architecture (arm64/amd64)
2. Checks system dependencies
3. Downloads the latest release securely
4. Validates checksums
5. Installs to your system PATH

### 3. **Download Pre-built Binaries**
Ready-to-use binaries for various platforms are available on the [Releases](https://github.com/bavix/gripmock/releases) page. Download the right binary for your system and add it to your `PATH`.

### 4. **Using Docker**
GripMock comes as a Docker image for easy use. Make sure Docker is installed:  
[Install Docker](https://docs.docker.com/engine/install/).

Pull the latest GripMock Docker image:
```bash
docker pull bavix/gripmock
```

### 5. **Using Go**
If you have Go installed, you can install GripMock directly:
```bash
go install github.com/bavix/gripmock/v3@latest
```

## Preparation

Assume we have a gRPC service defined in `simple.proto`:
```proto
syntax = "proto3";

package simple;

service Gripmock {
  rpc SayHello (Request) returns (Reply);
}

message Request {
  string name = 1;
}

message Reply {
  string message = 1;
  int32 returnCode = 2;
}
```

## Run GripMock

### Single Service (Traditional .proto)
```bash
docker run \
  -p 4770:4770 \
  -p 4771:4771 \
  -v ./api/proto:/proto:ro \
  bavix/gripmock /proto/simple.proto
```

### Using Proto Descriptors (New)
GripMock now supports compiled proto descriptors (`.pb` files) for better dependency management:

1. **Generate descriptor file**:
   Using Protocol Buffers Compiler (`protoc`):
   ```bash
   protoc --proto_path=. --descriptor_set_out=service.pb service.proto
   ```
   
   Or using Buf (modern build tool):
   ```bash
   buf build -o service.pb
   ```

2. **Run with descriptor**:
   ```bash
   docker run \
     -p 4770:4770 \
     -p 4771:4771 \
     -v ./api/proto:/proto:ro \
     bavix/gripmock /proto/service.pb
   ```

> **Note**:  
> - When using `protoc`, add `--include_imports` for multi-file dependencies  
> - Buf automatically handles dependencies and requires no extra flags  
> - Descriptors package services/dependencies into a single binary file  
> - Buf requires a valid `buf.yaml` configuration in your project

### Multiple Services
#### Option 1: Specify Multiple Files
```bash
docker run \
  -p 4770:4770 \
  -p 4771:4771 \
  -v ./protos:/proto:ro \
  bavix/gripmock /proto/service1.proto /proto/service2.proto
```

#### Option 2: Auto-Load Folder
Mount a directory containing **multiple `.proto` and `.pb` files**:
```bash
docker run \
  -p 4770:4770 \
  -p 4771:4771 \
  -v ./protos:/proto:ro \
  bavix/gripmock /proto
```
This will **automatically load all `.proto` and `.pb` files** in the `/proto` directory.

> **Note**:  
> - All `.proto` and `.pb` files in the specified directory will be processed  
> - Ensure there are no conflicting service/message definitions across files  
> - Subdirectories are scanned recursively  
> - **Important**: If duplicate services are found in both `.proto` and `.pb` files, GripMock will fail to start. In such cases, specify files manually instead of using folder auto-load.

### Load from Buf Schema Registry (BSR) <VersionTag version="v3.8.4" />

You can start GripMock from a BSR module reference directly:

```bash
gripmock --stub ./stubs buf.build/connectrpc/eliza
```

See the dedicated BSR page for refs, private modules, and env vars: [BSR](/guide/sources/bsr).

## Web UI (v3.0+)
Access the admin panel at:  
**http://localhost:4771/** (default port).  
Features:
- Create, edit, and delete stubs.
- View lists of used/unused stubs.

## Stubbing

### Dynamic Stubs (API)
Add a stub via `curl`:
```bash
curl -X POST -d '{
  "service": "Gripmock",
  "method": "SayHello",
  "input": { "equals": { "name": "gripmock" } },
  "output": { "data": { "message": "Hello GripMock" } }
}' http://127.0.0.1:4771/api/stubs
```

Response (stub ID):
```json
["6c85b0fa-caaf-4640-a672-f56b7dd8074d"]
```

### Static Stubs (YAML/JSON)
Mount a stubs directory and use `--stub`:
```bash
docker run ... -v ./stubs:/stubs bavix/gripmock --stub=/stubs /proto/simple.proto
```

## Verification

### Check Stubs
- **API**:  
  ```bash
  curl http://127.0.0.1:4771/api/stubs
  ```
- **UI**: Visit **http://localhost:4771/** and navigate to the stubs section.

## Advanced Features

### Binary Descriptor Support <VersionTag version="v3.1.0" />
When using `.pb` descriptors:
- No need for original `.proto` files in the container
- Faster startup with pre-compiled definitions
- Better handling of complex proto dependencies
- Supports all features available with regular `.proto` files

> **Tip**: Use Buf for modern proto workflows:
> ```bash
> buf build --exclude-source-info -o service.pb
> ```
> This creates leaner descriptors optimized for runtime use

### Headers Matching <VersionTag version="v2.1.0" />
Add headers to stubs for fine-grained control:
```json
{
  "headers": {
    "equals": { "authorization": "Bearer token123" }
  },
  "input": { ... },
  "output": { ... }
}
```

### Array Order Flexibility <VersionTag version="v2.6.0" />
Use `ignoreArrayOrder: true` to disable array sorting checks:
```json
{
  "input": {
    "ignoreArrayOrder": true,
    "equals": { "ids": ["id2", "id1"] }
  }
}
```

### Health checks <VersionTag version="v2.0.2" />
Check service status:
```bash
curl http://127.0.0.1:4771/api/health/liveness
curl http://127.0.0.1:4771/api/health/readiness
```

### Plugins <VersionTag version="v3.5.0" />
Extend template functions with custom plugins:
```bash
# Build plugins
make plugins

# Load plugins
gripmock --plugins=./plugins/hash.so --plugins=./plugins/math.so service.proto
```
Use plugin functions in dynamic templates:
```yaml
output:
  data:
    hash: "{{.Request.data | sha256}}"
    result: "{{pow .Request.base .Request.exponent}}"
```
See [Plugins](../plugins/) for detailed documentation.

## Cleanup
- **Delete all stubs**:  
  ```bash
  curl -X DELETE http://127.0.0.1:4771/api/stubs
  ```
- **Delete specific stub**:  
  ```bash
  curl -X DELETE http://127.0.0.1:4771/api/stubs/{uuid}
  ```
