---
title: Performance Comparison
---

# Performance Comparison

`bavix/gripmock` against the original `tokopedia/gripmock`. Every number comes
from [`bench/`](https://github.com/bavix/gripmock/tree/master/bench).
Full run: 29 minutes.

## Runtime architecture

| | `tokopedia/gripmock` | `bavix/gripmock` |
| --- | --- | --- |
| Protobuf handling | `protoc` runs at startup | in-process, no code generation |
| Server process | generated Go server compiled and spawned | single application |
| Stub lookup | internal gRPC to HTTP hop | in-process |
| Runtime dependencies in image | `protobuf`, `protoc-gen-go`, `protoc-gen-go-grpc` | none |

## Results

### Throughput

Peak requests per second.

| Stubs | Scenario | bavix | bavix native | tokopedia |
| --- | --- | --- | --- | --- |
| 500 | equals | 29 804 | 36 751 | 6 221 |
| 500 | contains | 29 579 | 36 069 | 6 191 |
| 500 | matches | 28 955 | 35 924 | 986 |
| 500 | miss | 15 914 | 18 685 | 4 091 |
| 10 000 | equals | 29 386 | 36 174 | 383 |
| 10 000 | contains | 29 419 | 37 407 | 330 |
| 10 000 | matches | 28 621 | 34 591 | 32 |
| 10 000 | miss | 11 220 | 12 637 | 135 |
| 500 000 | equals | 30 527 | 38 609 | 32 |
| 500 000 | contains | 30 265 | 37 624 | 31 |
| 500 000 | matches | 27 626 | 33 534 | 1 |
| 500 000 | miss | 11 536 | 12 879 | 4 |

`bavix/gripmock` resolves a request through an index covering all three
matcher kinds, including regular expressions that reduce to an anchored
literal. `tokopedia/gripmock` scans the stub list.

![Throughput, equals matcher](/bench/throughput-equals.svg)

![Throughput, no matching stub](/bench/throughput-miss.svg)

Matcher kinds at 500 stubs. `tokopedia/gripmock` drops from 6 221 to 986 req/s
on regular expressions; `bavix/gripmock` drops from 29 804 to 28 955:

![Throughput by matcher kind](/bench/matcher-kinds.svg)

### Latency

At 500 stubs, concurrency 100:

| | avg | p50 | p95 | p99 |
| --- | --- | --- | --- | --- |
| bavix | 3.33 ms | 2.91 ms | 6.75 ms | 9.44 ms |
| bavix native | 2.69 ms | 2.30 ms | 5.69 ms | 8.18 ms |
| tokopedia | 16.06 ms | 16.10 ms | 22.90 ms | 24.08 ms |

![Latency distribution](/bench/latency-percentiles.svg)

p99 against stub count, and the same call without queueing at concurrency 1:

| Stubs | bavix p99 | bavix native p99 | tokopedia p99 | tokopedia at c=1 |
| --- | --- | --- | --- | --- |
| 500 | 9 ms | 8 ms | 24 ms | 0.80 ms |
| 10 000 | 10 ms | 8 ms | 552 ms | 8.72 ms |
| 500 000 | 9 ms | 8 ms | 3485 ms | 85.98 ms |

`bavix/gripmock` answers in 0.20 ms at concurrency 1, at every stub count.

### CPU and memory

| Stubs | | CPU avg | Requests/s per core | At rest | Peak under load |
| --- | --- | --- | --- | --- | --- |
| 500 | bavix | 348 % | 8 564 | 153 MB | 795 MB |
| 500 | bavix native | 355 % | 10 352 | 184 MB | 730 MB |
| 500 | tokopedia | 338 % | 1 841 | 71 MB | 108 MB |
| 10 000 | bavix | 348 % | 8 439 | 166 MB | 936 MB |
| 10 000 | bavix native | 356 % | 10 167 | 201 MB | 794 MB |
| 10 000 | tokopedia | 155 % | 247 | 84 MB | 135 MB |
| 500 000 | bavix | 336 % | 9 088 | 989 MB | 2522 MB |
| 500 000 | bavix native | 350 % | 11 028 | 1025 MB | 2520 MB |
| 500 000 | tokopedia | 230 % | 14 | 800 MB | 1057 MB |

All three saturate the four CPUs granted to them. Per core `bavix/gripmock`
delivers 4.7 times more requests at 500 stubs and 649 times more at 500 000.

Memory per 1000 requests/s, `bavix/gripmock` against `tokopedia/gripmock`:
26.7 MB against 17.4 MB at 500 stubs, 31.9 MB against 352.5 MB at 10 000,
82.6 MB against 33 031 MB at 500 000.

![CPU efficiency](/bench/efficiency-cpu.svg)

![Memory at rest](/bench/memory-usage.svg)

### Startup

Until the service answers gRPC reflection, one stub loaded, 50 container starts
each.

| | min | avg | max |
| --- | --- | --- | --- |
| bavix | 0.347 s | 0.399 s | 0.453 s |
| bavix native | 0.169 s | 0.174 s | 0.228 s |
| tokopedia | 1.577 s | 2.205 s | 3.418 s |

![Startup readiness](/bench/startup-ready.svg)

### Image size

Compressed layers from the registry manifest, read from published tags.

| | linux/amd64 | linux/arm64 |
| --- | --- | --- |
| `bavix/gripmock:3.18.4` | 19.17 MB | 18.49 MB |
| `tkpd/gripmock:v1.14` | 226.29 MB | 219.90 MB |

![Image size](/bench/image-size.svg)

## Method

Both servers get the same stub file byte for byte, the same proto, the same
limits, the same client invocation, one at a time.

| Scenario | Stub set | Outcome |
| --- | --- | --- |
| `equals` | `input.equals` | every request matches |
| `contains` | `input.contains` | every request matches |
| `matches` | `input.matches`, anchored regex per stub | every request matches |
| `miss` | `input.equals` | no request can match |

- Requests come from `names.csv`, one row per stub, shuffled with a fixed seed.
  In insertion order a scanning engine answers request N after N comparisons.
- `miss` asserts `ERROR partial {}`, because the two engines return different
  status codes for an unmatched request.
- A level stops at `BENCH_REQUESTS` or its time cap, whichever comes first.
- A run is rejected if the container stops mid-measurement, if any request
  fails at the transport level, if the outcome disagrees with the scenario, or
  if the load generator reports that it was itself the bottleneck. All 72
  measurements passed. Throughput derived from measured latency and
  concurrency agrees with the reported figure within 1 % in every row.
- Every report carries the generator's own CPU accounting, so a figure can be
  audited rather than taken on trust. The client held 2.3 of the host's 12
  cores at the fastest row and never raised `generator_limited`.
- Startup is measured from `docker run` until `helloworld.Greeter` answers
  reflection. docker-proxy binds the published port at container creation, and
  the health service is registered before the mocked one, so neither a TCP
  probe nor bare reflection marks readiness.
- CPU comes from `docker stats` for containers and from `/proc` for the native
  process, since `ps -o %cpu` averages over process lifetime. Averages only:
  `docker stats` peaks read above the container's own CPU limit, because its
  window does not align with the 100 ms CFS quota periods.
- Charts use a linear axis; bar length is proportional to value. Bars below
  half a percent of the axis are drawn at that floor. Metrics spanning more
  than about fifty times — p99 against stub count, memory per request — are
  tables.

### Calibration

`make bench-calibrate` measures the settings the run holds fixed. On this host:

| | | |
| --- | --- | --- |
| Client floor | 78 904 req/s | the same document against a no-op target inside the generator |
| 2 CPUs | 19 218 req/s | p99 14.8 ms |
| 4 CPUs | 29 907 req/s | p99 9.4 ms — chosen |
| 6 CPUs | 38 535 req/s | p99 7.8 ms |
| Concurrency 100 | 30 170 req/s | p99 10 ms — chosen |
| Concurrency 200 | 29 575 req/s | p99 20 ms |
| Concurrency 400 | 29 909 req/s | p99 39 ms |
| Concurrency 800 | 29 553 req/s | p99 72 ms |
| 50 000 req/level | 1.8 s window | 3.6 % spread over repeats |
| 150 000 req/level | 5.3 s window | 1.2 % spread |
| 300 000 req/level | 10.6 s window | 0.7 % spread — chosen |
| 1 connection | 29 480 req/s | 7.1 % client CPU per 1000 req/s |
| 8 connections | 28 308 req/s | 7.9 % client CPU per 1000 req/s — chosen |

The client floor is not a ceiling for a real run: the no-op target lives inside
the generator, so both ends share the same cores and the same CPU counter. Read
it as what the document costs before an engine is involved — 2.6 times the
fastest measured engine row, which is why no row here is generator-bound.

Throughput is flat from concurrency 100 to 800 while p99 grows sevenfold, so
concurrency 100 is the last level that buys anything. CPU scales close to
linearly across 2, 4 and 6 CPUs; 4 keeps the engine inside half the host and
leaves the rest to the client.

The run refuses to start unless the host has at least twice `BENCH_CPUS` cores
and is idle, because an unrelated job competing for CPU moves the figures by
tens of percent.

## Reproducing

Requires Docker, Go and [grpctestify](https://github.com/gripmock/grpctestify-rust)
v1.10.0 or newer.

```bash
make bench            # measure, then regenerate every chart
make bench-run        # measure only, writes bench/results-<count>/
make bench-chart      # redraw charts from the last measurement
make bench-calibrate  # re-measure the settings, writes bench/results-calibration/
```

[`bench/run.sh`](https://github.com/bavix/gripmock/blob/master/bench/run.sh)
drives both the measurement and the calibration,
[`bench/tests/`](https://github.com/bavix/gripmock/tree/master/bench/tests)
holds the scenarios, [`bench/chart.go`](https://github.com/bavix/gripmock/blob/master/bench/chart.go)
renders the SVGs.

| Variable | Default |
| --- | --- |
| `BAVIX_IMAGE` | `bavix/gripmock:3.18.4` |
| `TKPD_IMAGE` | `tkpd/gripmock:v1.14` |
| `BENCH_COUNTS` | `500 10000 500000` |
| `BENCH_SCENARIOS` | `equals contains matches miss` |
| `BENCH_CONCURRENCY_LEVELS` | `1,100` |
| `BENCH_REQUESTS` | `300000` |
| `BENCH_DURATION` | `30s` |
| `BENCH_SWEEP_DURATION` | `10s` |
| `BENCH_WARMUP` | `1s` |
| `BENCH_STARTUP_RUNS` | `50` |
| `BENCH_CONNECTIONS` | `8` |
| `BENCH_REQUEST_TIMEOUT` | `180s` |
| `BENCH_CPUS` | `4` |
| `BENCH_MEMORY` | `8g` |
| `BENCH_MAX_LOAD` | `1.5` |

### Environment

| | |
| --- | --- |
| Machine | AMD Ryzen 5 5600G, 6 cores / 12 threads, 30 GiB RAM |
| OS | Debian GNU/Linux 12 (bookworm), kernel 6.1.0-44-amd64 |
| Architecture | x86_64; both images linux/amd64, nothing emulated |
| Docker | 29.4.1 |
| Go | go1.26.2 |
| jq | jq-1.6 |
| grpctestify | v1.10.0 |
| bavix | `bavix/gripmock:3.18.4`, built from this tree |
| bavix native | same tree as a host binary, `GOMAXPROCS=4` |
| tokopedia | `tkpd/gripmock:v1.14` |
| Container limits | 4 CPUs, 8 GiB |
| Run time | 29 min |

## Known deviations

- `bavix/gripmock` reports 29 804 req/s at 500 stubs and 30 527 at 500 000 —
  faster with more stubs, by about 2 %, reproduced across three sweeps.
  Unexplained. At 500 stubs each stub is matched thousands of times per run
  against a handful at 500 000.
- The Docker build runs 19 to 23 % below the native process in every throughput
  row. Under grpctestify v1.9.5 the same gap read 9 to 11 %: the client was
  then the slower half of the pair, and it compressed the difference.
  `tokopedia/gripmock` has no native mode; it compiles a gRPC server from the
  proto at startup.
- A single client connection reaches 4 % more throughput than eight, at lower
  client CPU per request. The run still uses eight, because one HTTP/2
  connection multiplexes every stream through one flow-control window and that
  is a property of the client, not of the engine under test.
- Image pull times are not measured.
