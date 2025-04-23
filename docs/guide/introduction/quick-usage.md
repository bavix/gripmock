# Quick Usage

## Installation

GripMock can be installed using one of the following methods:

### 1. **Using Homebrew (Recommended)**
Homebrew provides an easy way to install GripMock on macOS and Linux.

#### Step 1: Tap the Repository
Tap the official Homebrew tap for GripMock:
```bash
brew tap gripmock/homebrew
```

#### Step 2: Install GripMock
Install GripMock with the following command:
```bash
brew install gripmock
```

#### Step 3: Verify Installation
Verify that GripMock is installed correctly by checking its version:
```bash
gripmock --version
```
You should see output similar to:
```
gripmock version v3.2.4
```

### 2. **Download Pre-built Binaries**
Pre-built binaries for various platforms are available on the [Releases](https://github.com/bavix/gripmock/releases) page. Download the appropriate binary for your system and add it to your `PATH`.

### 3. **Using Docker**
GripMock is packaged as a Docker image for ease of use. Ensure Docker is installed:  
[Install Docker](https://docs.docker.com/engine/install/).

Pull the latest GripMock Docker image:
```bash
docker pull bavix/gripmock
```

### 4. **Using Go**
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

### Binary Descriptor Support
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

### Headers Matching
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

### Array Order Flexibility
Use `ignoreArrayOrder: true` to disable array sorting checks:
```json
{
  "input": {
    "ignoreArrayOrder": true,
    "equals": { "ids": ["id2", "id1"] }
  }
}
```

### Healthchecks
Check service status:
```bash
curl http://127.0.0.1:4771/api/health/liveness
curl http://127.0.0.1:4771/api/health/readiness
```

## Cleanup
- **Delete all stubs**:  
  ```bash
  curl -X DELETE http://127.0.0.1:4771/api/stubs
  ```
- **Delete specific stub**:  
  ```bash
  curl -X DELETE http://127.0.0.1:4771/api/stubs/{uuid}
  ```
