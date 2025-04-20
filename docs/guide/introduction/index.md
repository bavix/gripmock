# Introduction

![GripMock](https://github.com/bavix/gripmock/assets/5111255/023aae40-5950-43ba-abd1-0803de6fd246)

`GripMock` is a mock server for gRPC services.

## Overview

GripMock is a **mock server** for **gRPC** services. It uses `.proto` files **or compiled .pb descriptors** to generate gRPC service implementations automatically. It is ideal for end-to-end testing or as a dummy server during development. The server is implemented in Go, but clients can use any language that supports gRPC.

This service is a fork of [tokopedia/gripmock](https://github.com/tokopedia/gripmock) with significant improvements and new features.

## Key Features

- **Binary Descriptor Support**: Use compiled `.pb` files for faster startup and simplified dependency management
- **Updated Dependencies**: All deprecated dependencies resolved ([tokopedia#64](https://github.com/tokopedia/gripmock/issues/64))
- **YAML Support**: Define static stubs using YAML as an alternative to JSON
- **Healthcheck Endpoints**: Built-in liveness and readiness checks at `/api/health/liveness` and `/api/health/readiness`
- **Header Matching**: Support for matching gRPC request headers ([tokopedia#144](https://github.com/tokopedia/gripmock/issues/144))
- **gRPC Error Codes**: Specify custom gRPC error codes in stub responses ([tokopedia#125](https://github.com/tokopedia/gripmock/issues/125))
- **Gzip Compression**: Enable gzip encoding for gRPC server responses ([tokopedia#134](https://github.com/tokopedia/gripmock/pull/134))
- **404 Errors**: Return `NOT_FOUND` errors for unmatched stubs ([tokopedia#142](https://github.com/tokopedia/gripmock/issues/142))
- **Stub Management**: Delete specific stubs by ID ([tokopedia#123](https://github.com/tokopedia/gripmock/issues/123))
- **Reduced Image Size**: Optimized Docker image for faster deployment ([tokopedia#91](https://github.com/tokopedia/gripmock/issues/91))
- **Active Maintenance**: Ongoing support and updates ([tokopedia#82](https://github.com/tokopedia/gripmock/issues/82))
- **Array Order Flexibility**: Disable array order checks with `ignoreArrayOrder` flag ([bavix#108](https://github.com/bavix/gripmock/issues/108))
- **Pre-Alpha Slim Version**: Experimental lightweight build (may contain bugs) ([bavix#512](https://github.com/bavix/gripmock/issues/512))
- **Web-based UI (v3.0+)**: A graphical interface for managing stubs and monitoring activity (preview below).  
  ![GripMock UI Preview](https://github.com/bavix/gripmock/assets/5111255/3d9ebb46-7810-4225-9a30-3e058fa5fe16)

## Web Interface (v3.0+)
The **dashboard** is now available in version 3.x, providing a user-friendly way to:
- Create, edit, and delete stubs
- View lists of used/unused stubs

Access the UI at `http://localhost:4771/` (default port).

## Support

For questions or issues, visit the [GitHub issues page](https://github.com/bavix/gripmock/issues).
