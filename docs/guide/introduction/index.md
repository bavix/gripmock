---
title: Introduction
---

# Introduction

![GripMock](https://github.com/bavix/gripmock/assets/5111255/023aae40-5950-43ba-abd1-0803de6fd246)

GripMock is a mock server for gRPC services. Point it at your `.proto` files or a
compiled `.pb` descriptor, describe the responses you want in YAML or JSON, and
it serves them — no backend, and no client library beyond the gRPC one your
language already has.

## Architecture

The engine does not generate gRPC server code at runtime, and does not spawn a
generated server through `cmd/exec`. Stubbing and transport are handled in
process, which is where the startup time and the image size come from — see the
[Performance Comparison](./performance-comparison).

## Key Features

- **Quick Start**: Use your `.proto` files to start a mock server instantly
- **YAML & JSON**: Define test responses in the format you prefer
- **Header & Input Matching**: `equals`, `contains`, `matches`, `anyOf` <VersionTag version="v3.11.0" />, `glob` <VersionTag version="v3.12.0" />
- **Streaming**: Server, client, and bidirectional streaming support
- **Error Simulation**: Test error handling with codes and details
- **Dynamic Templates**: Build responses from the request <VersionTag version="v3.4.0" />, with `faker.*` <VersionTag version="v3.10.0" />
- **Effects**: Stateful stubs with automatic upsert/delete after match <VersionTag version="v3.11.0" />
- **Health Checks**: Built-in health endpoints + mockable health service <VersionTag version="v3.9.3" />
- **TLS/mTLS**: Native TLS support for secure gRPC <VersionTag version="v3.8.1" />
- **Plugins**: Extend template functions with Go plugins <VersionTag version="v3.5.0" />
- **Match Limit (`options.times`)**: Limit how many times a stub can be matched <VersionTag version="v3.7.0" />
- **Embedded SDK**: Run GripMock inside Go tests <VersionTag version="v3.7.0" />
- **MCP API**: AI/agent tooling integration <VersionTag version="v3.7.0" />
- **Upstream Modes**: `proxy`, `replay`, `capture` for gradual migration <VersionTag version="v3.9.0" />
- **OpenTelemetry**: Export traces via OTLP <VersionTag version="v3.10.0" />
- **Prometheus**: Metrics at `/metrics` <VersionTag version="v3.10.0" />
- **Docker**: Lightweight container for CI/CD
- **GitHub Actions**: Official action for CI workflows

## Streaming Support

GripMock supports all gRPC streaming patterns:

- **Request-Response** — single request, single response
- **Server Streaming** — single request, multiple responses
- **Client Streaming** — multiple messages, single response
- **Bidirectional** — continuous two-way messaging

See [Streaming](../stubs/streaming) for details.

## Web Interface <VersionTag version="v3.0.0" />

The dashboard at `http://localhost:4771/` lists the loaded stubs, shows which of
them have been matched, and lets you edit them without restarting the server.

## Experimental Features

- **Embedded SDK** <VersionTag version="v3.7.0" />: Programmatic GripMock usage in Go tests with built-in verification helpers
- **Upstream Modes** <VersionTag version="v3.9.0" />: Reflection-based `proxy`/`replay`/`capture` modes for reverse proxy, local-first fallback, and automatic stub recording

## Runtime descriptor loading <VersionTag version="v3.7.0" />

`.pb` descriptors can be pushed into a running server without a restart — see the
[Descriptor API](/guide/api/descriptors).

## Getting Started

Install from Docker or a release binary, point GripMock at your `.proto` files,
and write the responses as YAML or JSON. [Quick Usage](./quick-usage) walks
through the first run.

Bugs and questions go to [GitHub issues](https://github.com/bavix/gripmock/issues).
