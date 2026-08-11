# JSON Schema for Stubs

A JSON Schema for stub files, so an editor can validate them and complete field
names as you type: **https://bavix.github.io/gripmock/schema/stub.json**

It covers a single stub object or an array of them, every input matching
strategy (`equals`, `contains`, `matches`, `glob`, `anyOf`), header matching,
streaming responses, delays, errors and `priority`.

## Usage

### In YAML Files

Add this line to the top of your YAML stub files:

```yaml
# yaml-language-server: $schema=https://bavix.github.io/gripmock/schema/stub.json

service: MyService
method: MyMethod
output:
  data:
    result: success
```

### In JSON Files

Add this to your JSON stub files:

```json
{
  "$schema": "https://bavix.github.io/gripmock/schema/stub.json",
  "service": "MyService",
  "method": "MyMethod",
  "output": {
    "data": {
      "result": "success"
    }
  }
}
```

## IDE Support

### VS Code

1. Install the "YAML" extension
2. Add the schema reference to your files, as shown above

Validation and completion then work as you type.

## Validation

### Command Line

```bash
# Validate JSON files
python -m json.tool your-stubs.json

# Validate with schema (requires jsonschema package)
pip install jsonschema
jsonschema -i your-stubs.json https://bavix.github.io/gripmock/schema/stub.json
```

### CI/CD Integration

```yaml
# GitHub Actions example
- name: Validate Stubs
  run: |
    pip install jsonschema
    jsonschema -i stubs.json https://bavix.github.io/gripmock/schema/stub.json
```

## Schema Features

### Priority

```yaml
priority: 100  # Higher numbers = higher priority (default: 0)
```

### Input Matching

```yaml
input:
  ignoreArrayOrder: true  # Optional: disable array order checks
  equals:                 # Exact match
    name: "test"
    id: 123
  contains:               # Partial match
    description: "test"
  matches:                # Regex match
    email: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
```

### Header Matching

```yaml
headers:
  equals:                 # Exact match
    authorization: "Bearer token123"
  contains:               # Partial match
    user-agent: "Mozilla"
  matches:                # Regex match
    x-version: "v\\d+"
```

### Output Configuration

```yaml
output:
  delay: "100ms"          # Optional delay
  data:                   # Response data
    message: "Hello"
    code: 200
  error: "Error message"  # Error response
  code: 14               # gRPC status code
  headers:               # Response headers
    x-request-id: "123"
  stream:                # Streaming response
    - message: "First"
    - message: "Second"
```

## Examples

### Simple Stub

```yaml
# yaml-language-server: $schema=https://bavix.github.io/gripmock/schema/stub.json

service: Greeter
method: SayHello
priority: 100
input:
  equals:
    name: "world"
output:
  data:
    message: "Hello, world!"
```

### Multiple Stubs

```yaml
# yaml-language-server: $schema=https://bavix.github.io/gripmock/schema/stub.json

- service: Greeter
  method: SayHello
  input:
    equals:
      name: "Alice"
  output:
    data:
      message: "Hello, Alice!"

- service: Greeter
  method: SayHello
  input:
    equals:
      name: "Bob"
  output:
    data:
      message: "Hello, Bob!"
```

### Streaming Response

```yaml
# yaml-language-server: $schema=https://bavix.github.io/gripmock/schema/stub.json

service: StreamService
method: StreamData
output:
  delay: "200ms"
  stream:
    - id: 1
      data: "First message"
    - id: 2
      data: "Second message"
```

### Error Response

```yaml
# yaml-language-server: $schema=https://bavix.github.io/gripmock/schema/stub.json

service: ErrorService
method: SimulateError
output:
  delay: "500ms"
  error: "Service temporarily unavailable"
  code: 14  # UNAVAILABLE
```

## Related Documentation

- [JSON Stubs Guide](../stubs/json.md)
- [YAML Stubs Guide](../stubs/yaml.md)
- [Input Matching Rules](../matcher/input.md)
- [Header Matching Rules](../matcher/headers.md)
