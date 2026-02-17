# GripMock Embedded SDK <VersionTag version="v3.7.0" />

::: warning
⚠️ **EXPERIMENTAL FEATURE**: The GripMock Embedded SDK is currently experimental. The API is subject to change without notice, and functionality may be modified in future versions. Use at your own risk and avoid in production-critical systems without careful consideration.
:::

::: info
**Minimum Requirements**: Go 1.26 or later
:::

The GripMock Embedded SDK provides an integrated way to use GripMock directly within your Go tests without requiring external processes or containers. This approach offers faster test execution and better integration with your development workflow.

## Real-World Example

Looking for a full project that uses embedded mode in practice?

- [bavix/greeter-gripmock-embedded](https://github.com/bavix/greeter-gripmock-embedded) - End-to-end example of GripMock Embedded SDK usage

## Why Use the Embedded SDK?

The GripMock Embedded SDK offers several key advantages over traditional external mock server approaches:

### 1. **Faster Test Execution**
- No network overhead for gRPC calls to external mock servers
- Eliminates process startup/shutdown time
- Reduces test execution time significantly, especially for large test suites

### 2. **Better Test Isolation**
- Each test gets its own isolated GripMock instance
- No cross-test pollution from shared state
- Deterministic test results

### 3. **Simplified Test Setup**
- No need to manage external processes or Docker containers
- Single function call to start a fully configured mock server
- Automatic cleanup when using the recommended patterns

### 4. **Type Safety**
- Compile-time checking of your mock setup
- Leverage Go's type system for stub definitions
- Reduce runtime errors in test setup

### 5. **Enhanced Debugging**
- Everything runs in the same process as your test
- Easier to debug test failures
- Full access to mock internals if needed

### 6. **Seamless Integration**
- Natural Go API that feels like part of your test code
- Easy to integrate with existing test frameworks
- Works well with popular testing libraries like `testify`

## Core Capabilities

The Embedded SDK allows you to:

- **Start and manage GripMock instances** directly from your Go code
- **Define stubs programmatically** using Go functions with type safety
- **Access history and verification features** directly from your tests
- **Run tests without external dependencies** like Docker or external processes
- **Verify call counts and patterns** with built-in verification tools
- **Support all gRPC features** including unary, streaming, headers, and error responses

## How It Works

The Embedded SDK creates a GripMock server instance directly within your test process. This server:
- Listens on a local port (typically a random available port)
- Responds to gRPC requests according to your stub definitions
- Maintains its own state for the duration of the test
- Automatically cleans up when the test completes

## Navigation

Explore the documentation:

- [Installation](./installation.md) - How to install the SDK
- [Quick Start](./quick-start.md) - Basic usage examples with AAA pattern
- [Defining Stubs](./defining-stubs.md) - How to define stubs with various matching strategies
- [Advanced Features](./advanced-features.md) - Advanced features like delays, headers, and priority
- [Verification](./verification.md) - How to verify calls and check history
- [Remote Mode](./remote-mode.md) - Connecting to remote GripMock instances
- [Best Practices](./best-practices.md) - Recommended patterns and practices