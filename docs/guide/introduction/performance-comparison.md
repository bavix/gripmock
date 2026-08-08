---
title: Performance Comparison
---

# Performance Comparison

`bavix/gripmock` against the original `tokopedia/gripmock`. Every number comes
from [`bench/`](https://github.com/bavix/gripmock/tree/master/bench).
Full run: 26 minutes.

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
| 500 | equals | 24 284 | 26 777 | 5 600 |
| 500 | contains | 24 242 | 26 518 | 5 594 |
| 500 | matches | 24 023 | 26 409 | 936 |
| 500 | miss | 13 867 | 15 237 | 3 762 |
| 10 000 | equals | 23 947 | 26 627 | 393 |
| 10 000 | contains | 24 272 | 26 623 | 368 |
| 10 000 | matches | 23 375 | 25 968 | 34 |
| 10 000 | miss | 10 032 | 10 681 | 159 |
| 500 000 | equals | 25 032 | 27 177 | 31 |
| 500 000 | contains | 24 391 | 27 523 | 31 |
| 500 000 | matches | 22 619 | 24 838 | 1 |
| 500 000 | miss | 10 472 | 11 260 | 4 |

`bavix/gripmock` resolves a request through an index covering all three
matcher kinds, including regular expressions that reduce to an anchored
literal. `tokopedia/gripmock` scans the stub list.

![Throughput, equals matcher](/bench/throughput-equals.svg)

![Throughput, no matching stub](/bench/throughput-miss.svg)

Matcher kinds at 500 stubs. `tokopedia/gripmock` drops from 5 600 to 936 req/s
on regular expressions; `bavix/gripmock` drops from 24 284 to 24 023:

![Throughput by matcher kind](/bench/matcher-kinds.svg)

### Latency

At 500 stubs, concurrency 100:

| | avg | p50 | p95 | p99 |
| --- | --- | --- | --- | --- |
| bavix | 4.08 ms | 3.64 ms | 8.01 ms | 11.01 ms |
| bavix native | 3.70 ms | 3.24 ms | 7.54 ms | 10.86 ms |
| tokopedia | 17.84 ms | 17.84 ms | 24.82 ms | 26.10 ms |

![Latency distribution](/bench/latency-percentiles.svg)

p99 against stub count, and the same call without queueing at concurrency 1:

| Stubs | bavix p99 | bavix native p99 | tokopedia p99 | tokopedia at c=1 |
| --- | --- | --- | --- | --- |
| 500 | 11 ms | 10 ms | 26 ms | 0.86 ms |
| 10 000 | 11 ms | 10 ms | 576 ms | 8.58 ms |
| 500 000 | 10 ms | 10 ms | 3518 ms | 93.32 ms |

`bavix/gripmock` answers in 0.37 ms at concurrency 1, at every stub count.

### CPU and memory

| Stubs | | CPU avg | Requests/s per core | At rest | Peak under load |
| --- | --- | --- | --- | --- | --- |
| 500 | bavix | 352 % | 6 891 | 154 MB | 776 MB |
| 500 | bavix native | 366 % | 7 310 | 183 MB | 718 MB |
| 500 | tokopedia | 380 % | 1 476 | 71 MB | 113 MB |
| 10 000 | bavix | 350 % | 6 842 | 167 MB | 787 MB |
| 10 000 | bavix native | 366 % | 7 267 | 202 MB | 749 MB |
| 10 000 | tokopedia | 158 % | 249 | 89 MB | 145 MB |
| 500 000 | bavix | 342 % | 7 317 | 992 MB | 2414 MB |
| 500 000 | bavix native | 366 % | 7 434 | 1021 MB | 2416 MB |
| 500 000 | tokopedia | 228 % | 14 | 802 MB | 1064 MB |

All three saturate the four CPUs granted to them. Per core `bavix/gripmock`
delivers 4.7 times more requests at 500 stubs and 520 times more at 500 000.

Memory per 1000 requests/s, `bavix/gripmock` against `tokopedia/gripmock`:
32.0 MB against 20.2 MB at 500 stubs, 32.9 MB against 368.9 MB at 10 000,
96.4 MB against 33 671 MB at 500 000.

![CPU efficiency](/bench/efficiency-cpu.svg)

![Memory at rest](/bench/memory-usage.svg)

### Startup

Until the service answers gRPC reflection, one stub loaded, 50 container starts
each.

| | min | avg | max |
| --- | --- | --- | --- |
| bavix | 0.369 s | 0.430 s | 0.507 s |
| bavix native | 0.172 s | 0.177 s | 0.214 s |
| tokopedia | 1.743 s | 2.285 s | 3.337 s |

![Startup readiness](/bench/startup-ready.svg)

### Image size

Compressed layers from the registry manifest, read from published tags.

| | linux/amd64 | linux/arm64 |
| --- | --- | --- |
| `bavix/gripmock:3.18.2` | 19.13 MB | 18.45 MB |
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
  fails at the transport level, or if the outcome disagrees with the scenario.
  All 72 measurements passed. Throughput derived from measured latency and
  concurrency agrees with the reported figure within 1 % in every row.
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

Measured before choosing the settings, on this host:

| | | |
| --- | --- | --- |
| Environment ceiling | 31 000 req/s | unconstrained server, one stub, flat from concurrency 50 to 400 |
| 2 CPUs | 13 915 req/s | 45 % of ceiling |
| 4 CPUs | 22 003 req/s | 71 % of ceiling — chosen |
| 6 CPUs | 25 492 req/s | 82 % of ceiling |
| Concurrency 50 | 18 209 req/s | p99 4 ms |
| Concurrency 100 | 18 925 req/s | p99 9 ms — chosen |
| Concurrency 200 | 19 413 req/s | p99 20 ms |
| Concurrency 800 | 19 757 req/s | p99 70 ms |
| 50 000 req/level | 2.0 s window | 9.8 % spread over repeats |
| 150 000 req/level | 6.0 s window | 5.3 % spread |
| 300 000 req/level | 12.2 s window | 2.0 % spread — chosen |
| 1 connection | | 14.4 % client CPU per 1000 req/s |
| 8 connections | | 8.8 % client CPU per 1000 req/s — chosen |

The run refuses to start unless the host has at least twice `BENCH_CPUS` cores
and is idle. The same measurement returns 22 078 req/s on an idle machine and
8 023 while an unrelated job holds five of twelve threads.

## Reproducing

Requires Docker, Go and [grpctestify](https://github.com/gripmock/grpctestify-rust)
v1.9.5 or newer.

```bash
make bench        # measure, then regenerate every chart
make bench-run    # measure only, writes bench/results-<count>/
make bench-chart  # redraw charts from the last measurement
```

[`bench/run.sh`](https://github.com/bavix/gripmock/blob/master/bench/run.sh)
drives the measurement, [`bench/tests/`](https://github.com/bavix/gripmock/tree/master/bench/tests)
holds the scenarios, [`bench/chart.go`](https://github.com/bavix/gripmock/blob/master/bench/chart.go)
renders the SVGs.

| Variable | Default |
| --- | --- |
| `BAVIX_IMAGE` | `bavix/gripmock:3.18.3` |
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
| Go | go1.26.5 |
| jq | jq-1.6 |
| grpctestify | v1.9.5 |
| bavix | `bavix/gripmock:3.18.3`, built from this tree |
| bavix native | same tree as a host binary, `GOMAXPROCS=4` |
| tokopedia | `tkpd/gripmock:v1.14` |
| Container limits | 4 CPUs, 8 GiB |
| Run time | 26.4 min |

## Known deviations

- `bavix/gripmock` reports 24 284 req/s at 500 stubs and 25 032 at 500 000 —
  faster with more stubs, by about 3 %, reproduced across three sweeps.
  Unexplained. At 500 stubs each stub is matched thousands of times per run
  against a handful at 500 000.
- The Docker build runs 9 to 11 % below the native process in every throughput
  row. `tokopedia/gripmock` has no native mode; it compiles a gRPC server from
  the proto at startup.
- Image pull times are not measured.
