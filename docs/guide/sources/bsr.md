# Buf Schema Registry (BSR) <VersionTag version="v3.8.4" />

GripMock can load gRPC API definitions directly from Buf Schema Registry modules.

## Basic usage

Start GripMock from a public BSR module:

```bash
gripmock --stub ./stubs buf.build/connectrpc/eliza
```

This uses `main` by default if no ref is provided.

## Pin module ref

You can pin by label or commit ID:

```bash
gripmock --stub ./stubs buf.build/connectrpc/eliza:main
gripmock --stub ./stubs buf.build/connectrpc/eliza@233fca715f49425581ec0a1b660be886
```

## Private modules

Use default BSR profile env vars:

```bash
BSR_BUF_TOKEN=<token> gripmock --stub ./stubs buf.build/acme/private-api
```

## Configuration

GripMock uses simple BSR profile configuration with these environment variables:

- `BSR_BUF_BASE_URL` - Base URL for public BSR (default: `https://buf.build`)
- `BSR_BUF_TOKEN` - Token for public BSR access
- `BSR_BUF_TIMEOUT` - Request timeout (default: `5s`)
- `BSR_SELF_BASE_URL` - Base URL for self-hosted BSR
- `BSR_SELF_TOKEN` - Token for self-hosted BSR
- `BSR_SELF_TIMEOUT` - Request timeout (default: `5s`)

## Host matching

GripMock routes modules based on host matching:

GripMock uses the host from `BSR_SELF_BASE_URL` for routing.

### Examples

```bash
# 1. Public buf.build (default)
BSR_BUF_BASE_URL=https://buf.build
BSR_BUF_TOKEN=<token>
# Modules: buf.build/owner/repo → uses Buf profile

# 2. Self-hosted BSR
BSR_SELF_BASE_URL=https://bsr.company.local
BSR_SELF_TOKEN=<token>
# Host extracted from BaseURL: bsr.company.local → modules with this host use Self profile
```

## On-premise BSR

For self-hosted BSR installations:

```bash
# Self-hosted BSR
BSR_SELF_BASE_URL=https://bsr.company.local \
BSR_SELF_TOKEN=<token> \
gripmock --stub ./stubs bsr.company.local/team/payments
```

With explicit ref:

```bash
BSR_SELF_BASE_URL=https://bsr.company.local \
BSR_SELF_TOKEN=<token> \
gripmock --stub ./stubs bsr.company.local/team/payments:main
```

Mixed setup (public + on-prem):

```bash
# Public BSR
BSR_BUF_BASE_URL=https://buf.build
BSR_BUF_TOKEN=<public-token>

# On-premise BSR
BSR_SELF_BASE_URL=https://bsr.company.local
BSR_SELF_TOKEN=<on-prem-token>

# Routes:
# - buf.build/owner/repo → uses Buf profile
# - bsr.company.local/owner/repo → uses Self profile
```

## How routing works

1. When module is requested (e.g., `bsr.company.local/owner/repo`)
2. GripMock extracts the host part (`bsr.company.local`)
3. If Self profile is configured and host matches → use Self client
4. Otherwise → use Buf client

This simple approach makes it easy to work with multiple BSR instances without complex configuration.

## Project fixture example

This repository includes a real BSR fixture project:

```bash
cd third_party/bsr/eliza
make up
make test
make down
```
