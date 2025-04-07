# Quick Usage

## Installation

For ease of use, GripMock is packaged as a Docker image. Ensure Docker is installed:  
[Install Docker](https://docs.docker.com/engine/install/).

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

### Single Service
```bash
docker run \
  -p 4770:4770 \
  -p 4771:4771 \
  -v ./api/proto:/proto:ro \
  bavix/gripmock /proto/simple.proto
```

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
Mount a directory containing **multiple `.proto` files**:
```bash
docker run \
  -p 4770:4770 \
  -p 4771:4771 \
  -v ./protos:/proto:ro \
  bavix/gripmock /proto
```
This will **automatically load all `.proto` files** in the `/proto` directory.

> **Note**:  
> - All `.proto` files in the specified directory will be processed  
> - Ensure there are no conflicting service/message definitions across files  
> - Subdirectories are scanned recursively  

## Web UI (v3.0+)
Access the admin panel at:  
**http://localhost:4771/** (default port).  
Features:
- Create, edit, and delete stubs.
- View lists of used/unused stubs.
- Monitor real-time activity logs. (much later)
- Access healthcheck status. (much later)

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
