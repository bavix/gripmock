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

## Key Features

- **Quick Start**: Use your `.proto` files to generate a server instantly
- **YAML & JSON**: Define your test responses in the format you prefer
- **Health Monitoring**: Built-in health checks for production deployment
- **Header Testing**: Test different authentication tokens and headers
- **Error Simulation**: Test how your app handles real-world errors
- **File Upload Testing**: Test chunked file uploads and batch processing
- **Real-time Chat**: Test bidirectional streaming for chat applications
- **Web Interface**: Manage your test scenarios through a friendly web UI
- **Docker Ready**: Lightweight container perfect for CI/CD pipelines
- **20-35% Faster**: Latest improvements make your tests run quicker
- **Zero Breaking Changes**: All your existing tests continue to work

## Streaming Support

GripMock supports all types of gRPC communication:

### Simple Requests (1:1)
Traditional request-response - you send one message, get one response back.

### Data Feeds (1:N)
Send one request, receive multiple responses over time - perfect for real-time data.

### File Uploads (N:1)
Send multiple messages (like file chunks), receive one summary response.

### Real-time Chat (N:N)
Send and receive messages continuously - ideal for chat apps and live collaboration.

## Web Interface

The **dashboard** provides a user-friendly way to:
- Create and edit test responses
- Monitor which responses are being used
- Manage your test scenarios visually

Access it at `http://localhost:4771/` when you start GripMock.

## Getting Started

1. **Install**: Download or use Docker
2. **Configure**: Point to your `.proto` files
3. **Define Responses**: Create YAML/JSON files with your test data
4. **Test**: Your mock server is ready!

## Need Help?

Stuck with something? Check out our [GitHub issues page](https://github.com/bavix/gripmock/issues) or join our community discussions.
