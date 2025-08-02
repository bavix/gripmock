# Circle Converter

Simple microservice example for geometric calculations.

## What it does

- Converts radius to diameter
- Shows "one function - one task" principle
- Demonstrates simple mathematical operations

## Run

```bash
gripmock --stub examples/projects/circle-converter examples/projects/circle-converter/service.proto
```

## Tests

```bash
grpctestify examples/projects/circle-converter/
```

## Structure

- `service.proto` - gRPC service definition
- `stub.yaml` - mock responses for testing
- `test.gctf` - test scenario

## Features

- **Single Responsibility**: One method, one operation
- **Performance**: Optimized for frequent calculations
- **Precision**: Accurate mathematical calculations
- **Simplicity**: Minimal and clear API 