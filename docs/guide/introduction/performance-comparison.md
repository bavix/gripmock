---
title: Performance Comparison
---

# Performance Comparison

This page contains architecture and benchmark comparison details.

## Runtime Architecture

### Original runtime

- Runtime protobuf code generation through `protoc`
- Process orchestration to run generated gRPC server code
- Extra runtime toolchain dependencies in container (`protobuf`, `protoc-gen-go`, `protoc-gen-go-grpc`, scripts)

### Bavix runtime

- Native in-process runtime engine (single application)
- No runtime gRPC code generation
- No generated child gRPC server process
- No internal gRPC->HTTP hop for stub lookup in request path

## Benchmark Source

All metrics are generated from the internal benchmark pipeline and published here as summarized results and charts.

The benchmark suite source will be published separately after final cleanup and stabilization.

Latest measured highlights (example run):

- Stub dataset loaded into both runtimes before GHZ: **5000 stubs** (same dataset, same matching key format)

- Image size (compressed layers):
  - `bavix/gripmock` amd64: **16.55 MB**, `tkpd/gripmock` amd64: **226.29 MB** (**92.69% smaller**)
  - `bavix/gripmock` arm64: **16.02 MB**, `tkpd/gripmock` arm64: **219.90 MB** (**92.71% smaller**)
- Startup readiness (both gRPC + HTTP ready):
  - simple proto: **0.398s vs 1.265s** (**68.54% faster**)
  - wkt proto: **0.477s vs 1.398s** (**65.88% faster**)
  - average: **0.438s vs 1.331s** (**67.14% faster**)
- GHZ unary (`Greeter/SayHello`, 30s, concurrency 20):
  - RPS: **2349.19 vs 1256.09** (**87.02% higher**)
  - avg latency: **8.394ms vs 15.859ms** (**47.07% lower**)
  - p75 latency: **10.119ms vs 16.899ms** (**40.12% lower**)
  - p95 latency: **17.259ms vs 19.047ms** (**9.38% lower**)

Pull-time metric is environment-sensitive (registry/CDN/cache conditions) and should be interpreted with caution.

## Charts

### Image size benchmark

Shows compressed Docker image size on amd64 and arm64.

![Image size benchmark](/bench/image-size.svg)

### Startup readiness benchmark

Shows time to full readiness (both gRPC and HTTP endpoints available).

![Startup readiness benchmark](/bench/startup-ready.svg)

### Latency percentiles benchmark

Shows request latency distribution across avg, p75, p95, and p99.

![Latency percentiles benchmark](/bench/latency-percentiles.svg)

### Throughput benchmark

Shows achieved requests per second under the benchmark load profile.

![Throughput benchmark](/bench/throughput-rps.svg)
