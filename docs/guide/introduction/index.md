---
title: Introduction
---

# Introduction

![GripMock](https://github.com/bavix/gripmock/assets/5111255/023aae40-5950-43ba-abd1-0803de6fd246)

GripMock is your go-to tool for testing gRPC services. It creates a mock server that responds exactly how you want it to, making testing faster and more reliable.

## What is GripMock?

GripMock is a **mock server** for **gRPC** services. Give it your `.proto` files, and it instantly creates a working server that responds with your predefined test data. Perfect for testing your applications without needing a real backend server.

## Why Use GripMock?

- **Fast Setup**: Get a working server in seconds
- **Easy Testing**: Define responses in simple YAML or JSON files
- **Real Scenarios**: Test file uploads, chat applications, and data streaming
- **No Dependencies**: Works with any programming language that supports gRPC
- **Production Ready**: Built-in health checks and Docker support

## Architecture Story

GripMock was initially inspired by [tokopedia/gripmock](https://github.com/tokopedia/gripmock), but the current project is fully rewritten.

The modern GripMock engine is fundamentally different:

- It does **not** generate gRPC server code at runtime
- It does **not** spawn generated servers via `cmd/exec`
- It uses a native in-process runtime for stubbing and transport handling

## Key Features

- **Quick Start**: Use your `.proto` files to start a mock server instantly
- **YAML & JSON**: Define test responses in the format you prefer
- **Header & Input Matching**: Fine-grained request matching with `equals`, `contains`, `matches`, `anyOf` <VersionTag version="v3.11.0" />
- **Streaming**: Server, client, and bidirectional streaming support
- **Error Simulation**: Test error handling with codes and details
- **Dynamic Templates**: Generate realistic data with `faker.*`, `{{uuid}}`, etc. <VersionTag version="v3.10.0" />
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

The **dashboard** provides a user-friendly way to:
- Create and edit test responses
- Monitor which responses are being used
- Manage your test scenarios visually

Access it at `http://localhost:4771/` when you start GripMock.

## Experimental Features

- **Embedded SDK** <VersionTag version="v3.7.0" />: Programmatic GripMock usage in Go tests with built-in verification helpers
- **Upstream Modes** <VersionTag version="v3.9.0" />: Reflection-based `proxy`/`replay`/`capture` modes for reverse proxy, local-first fallback, and automatic stub recording

## Runtime descriptor loading <VersionTag version="v3.7.0" />

Need to load `.pb` descriptors into a running server without restart? See [Descriptor API (`/api/descriptors`)](/guide/api/descriptors).

## Getting Started

1. **Install**: Download or use Docker
2. **Configure**: Point to your `.proto` files
3. **Define Responses**: Create YAML/JSON files with your test data
4. **Test**: Your mock server is ready!

## Need Help?

Stuck with something? Check out our [GitHub issues page](https://github.com/bavix/gripmock/issues) or join our community discussions.
