# Introduction

![GripMock](https://github.com/bavix/gripmock/assets/5111255/023aae40-5950-43ba-abd1-0803de6fd246)

`GripMock` is a mock server for gRPC services.

## Overview

GripMock is a **mock server** for **gRPC** services. It's using a `.proto` file to generate implementation of gRPC service for you.
You can use gripmock for setting up end-to-end testing or as a dummy server in a software development phase.
The server implementation is in GoLang but the client can be any programming language that support gRPC.

This service is a fork of the service [tokopedia/gripmock](https://github.com/tokopedia/gripmock).

## Key features
- Updated all deprecated dependencies [tokopedia#64](https://github.com/tokopedia/gripmock/issues/64);
- Add yaml as json alternative for static stab's;
- Add endpoint for healthcheck (/api/health/liveness, /api/health/readiness);
- Add support headers [tokopedia#144](https://github.com/tokopedia/gripmock/issues/144);
- Add grpc error code [tokopedia#125](https://github.com/tokopedia/gripmock/issues/125);
- Added gzip encoding support for grpc server [tokopedia#134](https://github.com/tokopedia/gripmock/pull/134);
- Fixed issues with int64/uint64 [tokopedia#67](https://github.com/tokopedia/gripmock/pull/148);
- Add 404 error for stubs not found [tokopedia#142](https://github.com/tokopedia/gripmock/issues/142);
- Support for deleting specific stub [tokopedia#123](https://github.com/tokopedia/gripmock/issues/123);
- Reduced image size [tokopedia#91](https://github.com/tokopedia/gripmock/issues/91);
- Active support [tokopedia#82](https://github.com/tokopedia/gripmock/issues/82);
- Added [documentation](https://bavix.github.io/gripmock/);

## Support

Please ask questions on the [Github issues page](https://github.com/bavix/gripmock/issues).
