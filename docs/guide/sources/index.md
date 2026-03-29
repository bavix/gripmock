# Sources <VersionTag version="v3.8.4" />

GripMock can load API definitions from different source types.

## Supported source types

### 1) `.proto` files

Load one or multiple proto files directly:

```bash
gripmock service.proto
gripmock api/service1.proto api/service2.proto
```

### 2) Compiled descriptors (`.pb`, `.protoset`)

Load precompiled `FileDescriptorSet` files:

```bash
gripmock service.pb
gripmock api.protoset
```

This is useful for reproducible CI runs and dependency-free runtime startup.

### 3) Directory with mixed sources

Load all supported files recursively from a directory:

```bash
gripmock ./proto
```

GripMock will process `.proto`, `.pb`, and `.protoset` files found under the directory.

### 4) Buf Schema Registry (BSR)

Load API definitions directly from a BSR module:

```bash
gripmock --stub ./stubs buf.build/connectrpc/eliza
```

For private/on-prem usage and refs, see [BSR](/guide/sources/bsr).

### 5) gRPC Reflection (`grpc://`, `grpcs://`) <VersionTag version="v3.8.5" />

Load API definitions from a live gRPC server via reflection:

```bash
gripmock grpc://localhost:50051
gripmock grpcs://api.company.local:443
```

For query parameters and TLS notes, see [gRPC Reflection Source](/guide/sources/grpc-reflection).

### 6) Reflection Proxy Modes (`grpc+proxy://`, `grpc+replay://`, `grpc+capture://`) <VersionTag version="v3.9.0" />

Use reflection sources with forwarding modes:

```bash
gripmock grpc+proxy://localhost:1111
gripmock grpc+replay://localhost:1111
gripmock grpc+capture://localhost:1111
```

For behavior details and overlap rules, see [Upstream Modes](/guide/modes/index).
