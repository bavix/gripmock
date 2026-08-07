---
title: Performance Comparison
---

# Performance Comparison

GripMock compared against the original `tokopedia/gripmock`. Every number on
this page is produced by the benchmark in
[`bench/`](https://github.com/bavix/gripmock/tree/master/bench) and can be
regenerated from a checkout.

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

## Reproducing

Requires Docker and [grpctestify](https://github.com/gripmock/grpctestify-rust):

```bash
brew install gripmock/tap/grpctestify   # or: cargo install grpctestify
```

From a checkout:

```bash
make bench        # measure, then regenerate every chart below
make bench-run    # measure only, writes bench/results/
make bench-chart  # redraw charts from the last measurement
```

A different stub count or a different pair of versions:

```bash
BENCH_COUNT=10000 make bench-run
BAVIX_IMAGE=bavix/gripmock:3.18.3 make bench-run
```

Sources: [`bench/run.sh`](https://github.com/bavix/gripmock/blob/master/bench/run.sh)
drives the measurement, [`bench/tests/`](https://github.com/bavix/gripmock/tree/master/bench/tests)
holds the scenarios, [`bench/chart.go`](https://github.com/bavix/gripmock/blob/master/bench/chart.go)
renders the SVGs.

Everything is configurable through the environment:

| Variable | Default | Meaning |
| --- | --- | --- |
| `BAVIX_IMAGE` | `bavix/gripmock:3.18.3` | image under test |
| `TKPD_IMAGE` | `tkpd/gripmock:v1.14` | image under test |
| `BENCH_COUNT` | `500000` | stubs generated |
| `BENCH_STARTUP_RUNS` | `1000` | start/stop cycles for the startup measurement |
| `BENCH_CONCURRENCY_LEVELS` | `1,10,20,50` | concurrency sweep |
| `BENCH_DURATION` | `30s` | load duration at the highest level |
| `BENCH_SWEEP_DURATION` | `10s` | load duration at the other levels |
| `BENCH_SCENARIOS` | `hit miss` | scenarios from `bench/tests/` to run |
| `BENCH_RESULTS_DIR` | `results` | where JSON output is written |

## Method

Both servers receive identical treatment: the same generated `stubs.json`
(byte for byte, both accept the schema), the same proto, the same container
limits, and the same grpctestify invocation. They run one at a time, never
concurrently.

Two scenarios, both defined in `bench/tests/`:

- `hit.gctf` — the request matches a stub. Requests are driven from
  `names.csv`, one row per stub, in shuffled order with a fixed seed. Shuffling
  matters: an engine that scans stubs and stops at the first match answers
  request N after N comparisons, so asking in insertion order would measure
  request ordering rather than lookup cost.
- `miss.gctf` — no stub can match, so both implementations examine every stub
  before returning an error. The expectation is `ERROR partial {}`: the two
  engines return different status codes and messages for this case, and
  asserting the failure itself keeps one scenario file valid for both.

### Docker and native

Both implementations are benchmarked in Docker, under the same CPU and memory
limits, so the head-to-head numbers compare engines rather than packaging.

GripMock is additionally benchmarked as a native process, shown as
`bavix/gripmock (native)`. It ships as a single static binary: the container
image is `alpine` plus that binary, and nothing else is required at runtime.
`tokopedia/gripmock` has no equivalent mode — its image is built from
`golang:1.23-alpine` and installs `protoc`, `protoc-gen-go` and
`protoc-gen-go-grpc`, because it generates and compiles a gRPC server from the
proto when it starts. Running it outside a container means installing that
toolchain first.

The native process is given `GOMAXPROCS=2` to match the 2 CPUs granted to each
container. It still avoids the container network path, which is where most of
the remaining difference comes from.

Startup is measured separately, with a single stub loaded, as the time from
`docker run` until `helloworld.Greeter` answers gRPC reflection. It therefore
reports how quickly the server begins serving, not how quickly it parses a
dataset. A TCP probe would not work here — docker-proxy binds the published
port when the container is created, before anything inside listens — and
reflection merely responding is also too early, because GripMock registers its
health service before the mocked one.

Latency is reported as avg, p50, p95 and p99. p75 and p90 fall between
neighbours that already bracket them. p99.9 needs roughly 1000 requests before
it stops describing a single slow outlier, which a fixed-duration run does not
always reach; it stays in the JSON output.

A run is rejected rather than charted if the container stops mid-measurement,
if any request fails at the transport level, or if the outcome disagrees with
the scenario (every request must match in `hit`, none in `miss`).

## Environment

The published charts were measured on:

| | |
| --- | --- |
| Machine | Apple M1 Pro, 8 cores, 16 GB RAM |
| OS | macOS 26.5.2 |
| Docker | 29.4.0 |
| Container limits | 2 CPUs, 4 GiB per container |
| grpctestify | v1.9.4 |
| Go | go1.26.5 |
| jq | jq-1.8.2 |
| GripMock | `bavix/gripmock:3.18.3`, built from this working tree |
| GripMock (native) | same tree built as a host binary, `GOMAXPROCS=2` |
| tokopedia/gripmock | `tkpd/gripmock:v1.14` |
| Stub counts | 500, 1000, 10 000, 100 000, 500 000 |
| Concurrency levels | 1, 10, 20, 50 |
| Load duration | 10 s per level, 30 s at the highest |
| Startup samples | 20 container starts per engine |

Absolute values depend on the machine. The shape of the curves does not.
Image pull times are not measured; they depend on registry and CDN conditions.

Compressed image size is read from the registry manifest, so it is charted
only when both tags are published. For the last released pair it is 19.13 MB
(amd64) and 18.45 MB (arm64) for `bavix/gripmock:3.18.2`, against 226.29 MB
and 219.90 MB for `tkpd/gripmock:v1.14`.

## Charts

### Throughput, matching request

Peak requests per second across the concurrency sweep, by stub count.

![Throughput benchmark](/bench/throughput-rps.svg)

### Throughput, no matching stub

The same sweep for requests no stub can satisfy, where both implementations
examine every stub before giving up.

![Throughput on miss](/bench/throughput-miss.svg)

### p99 latency

Tail latency at the highest concurrency level, by stub count.

![p99 latency benchmark](/bench/latency-p99.svg)

### Latency distribution

Drawn at a small stub count, where both implementations are still within the
same range and the shape of the distribution stays visible.

![Latency percentiles benchmark](/bench/latency-percentiles.svg)

### Memory

Container resident memory with the stub set loaded, read before any request.

![Memory benchmark](/bench/memory-usage.svg)

### Startup

Time until the service answers gRPC reflection with one stub loaded, as
min/avg/max over repeated container starts.

![Startup readiness benchmark](/bench/startup-ready.svg)
