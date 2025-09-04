# GripMock - gRPC Mock Server

GripMock is a Go-based gRPC mock server that creates mock services from .proto files or compiled .pb descriptors. It supports dynamic stubbing via YAML/JSON, streaming operations, and provides both gRPC and REST interfaces.

**Always reference these instructions first and fallback to search or bash commands only when you encounter unexpected information that does not match the info here.**

## Working Effectively

### Environment Setup
- **Go**: Requires Go 1.25+ (project uses latest Go features)
- **Node.js**: Requires Node.js 20+ and npm 10+ for documentation builds  
- **Tools**: grpcurl for testing gRPC functionality, jq for JSON processing

### Tool Installation
```bash
# Install grpcurl (if not present)
curl -sSL "https://github.com/fullstorydev/grpcurl/releases/download/v1.9.3/grpcurl_1.9.3_linux_x86_64.tar.gz" | tar -xz
chmod +x grpcurl && sudo mv grpcurl /usr/local/bin/

# Install jq (if not present)  
sudo apt-get update && sudo apt-get install -y jq

# Install gofumpt for formatting (if needed)
go install mvdan.cc/gofumpt@latest
```

### Core Build Commands
**NEVER CANCEL these commands - wait for completion:**

```bash
# Download dependencies - takes ~10 seconds
go mod download

# Build the application - takes ~35 seconds, NEVER CANCEL
go build -o gripmock .

# Run unit tests - takes ~15 seconds  
go test ./...

# Documentation build - takes ~10 seconds
npm install && npm run docs:build
```

### Make Targets
```bash
# Run tests with coverage - takes ~60 seconds, NEVER CANCEL
make test
# Note: May show "covdata" tool warnings but tests will pass

# Lint code - KNOWN ISSUE: golangci-lint v2.4.0 incompatible with Go 1.25
make lint
# Workaround: Install latest version or skip for now

# Docker build - takes 3-5 minutes in normal environments, NEVER CANCEL  
make build
# Note: May fail in sandboxed environments due to network restrictions
```

## Application Usage

### Basic Server Operation
```bash
# Start server with proto file
./gripmock examples/projects/greeter/service.proto

# Start server with static stubs  
./gripmock --stub examples/projects/greeter examples/projects/greeter/service.proto

# Server runs on:
# - gRPC: localhost:4770
# - REST API/Web UI: localhost:4771
```

### Health and Status Checks
```bash
# Check if server is ready
./gripmock check --timeout=60s

# Health endpoints
curl localhost:4771/api/health/liveness
curl localhost:4771/api/health/readiness

# List stubs via API  
curl localhost:4771/api/stubs | jq .
```

## Validation Scenarios

**ALWAYS validate changes by running through these scenarios:**

### 1. Basic gRPC Functionality
```bash
# Install grpcurl if needed
curl -sSL "https://github.com/fullstorydev/grpcurl/releases/download/v1.9.3/grpcurl_1.9.3_linux_x86_64.tar.gz" | tar -xz
chmod +x grpcurl && sudo mv grpcurl /usr/local/bin/

# Start gripmock with greeter example
./gripmock --stub examples/projects/greeter examples/projects/greeter/service.proto &

# Test service listing
grpcurl -plaintext localhost:4770 list

# Test method call (should return "Hello, Alex!" using dynamic template)
grpcurl -plaintext -d '{"name":"Alex"}' localhost:4770 helloworld.Greeter/SayHello

# Stop server
pkill gripmock
```

### 2. REST API Validation  
```bash
# Start server in background
./gripmock examples/projects/greeter/service.proto &

# Test health
curl -s localhost:4771/api/health/liveness

# Test stub management
curl -s localhost:4771/api/stubs | jq .

# Stop server
pkill gripmock
```

### 3. Documentation Build
```bash
# Test docs can be built successfully
npm install
npm run docs:build
# Should complete without errors in ~10 seconds
```

## Key Project Structure

### Important Directories
- `cmd/` - CLI command implementations (root.go, check.go)
- `internal/app/` - Core application logic
- `internal/domain/` - Business domain (proto, rest, protoset)
- `internal/infra/` - Infrastructure (storage, encoding, grpc)
- `examples/` - Test examples and proto files
- `docs/` - VitePress documentation source

### Key Files to Monitor
- `main.go` - Application entry point
- `Makefile` - Build and test targets
- `go.mod` - Go dependencies (uses Go 1.25)
- `package.json` - Node.js dependencies for docs
- `.golangci.yml` - Linting configuration
- `.github/workflows/` - CI/CD pipelines

### Common File Locations
```bash
# Repository root
ls -la
#  .
#  ..
#  .git/
#  .github/
#  .golangci.yml
#  Dockerfile  
#  LICENSE
#  Makefile
#  README.md
#  changelog.md
#  cmd/
#  docs/
#  examples/
#  go.mod
#  go.sum
#  internal/
#  main.go
#  package.json
#  setup.sh
```

## Known Issues and Limitations

### golangci-lint Compatibility
- **Issue**: golangci-lint v2.4.0 is incompatible with Go 1.25
- **Error**: "Go language version (go1.24) used to build golangci-lint is lower than the targeted Go version (1.25)"
- **Workaround**: Install latest golangci-lint or temporarily skip linting
- **Status**: Will be resolved when golangci-lint releases Go 1.25 compatible version

### Docker Build in Sandboxed Environments
- **Issue**: Docker builds may fail with network restrictions
- **Error**: "unable to select packages: binutils (no such package)"  
- **Workaround**: Test locally or use pre-built images
- **Status**: Expected limitation in restricted environments

### Test Coverage Tools
- **Issue**: `make test` shows "go: no such tool 'covdata'" warnings
- **Impact**: Tests still pass, only coverage reporting affected
- **Workaround**: Ignore warnings or use `go test ./...` directly

## CI/CD Integration

**Always run these before committing:**

```bash
# Format and basic validation
go fmt ./...
# Install gofumpt if needed: go install mvdan.cc/gofumpt@latest  
~/go/bin/gofumpt -w .

# Run tests (skip lint if golangci-lint issue persists)
go test ./...

# Build to ensure no compilation errors  
go build -o /tmp/gripmock .

# Test basic functionality
/tmp/gripmock --version
/tmp/gripmock check --help
```

**CI Pipeline Overview:**
- Unit tests run in ~15 seconds
- E2E tests with coverage run in ~60 seconds  
- golangci-lint runs (may fail due to Go 1.25 issue)
- Docker builds and releases (in normal environments)

## Performance Notes

- **Go build**: ~35 seconds - set timeout to 60+ seconds
- **Unit tests**: ~15 seconds - set timeout to 30+ seconds  
- **E2E tests**: ~60 seconds - set timeout to 120+ seconds
- **Documentation build**: ~10 seconds - set timeout to 30+ seconds
- **Docker build**: 3-5 minutes - set timeout to 10+ minutes

**CRITICAL**: Never cancel long-running commands. Use appropriate timeouts and wait for completion.