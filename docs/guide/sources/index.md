# Sources

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
