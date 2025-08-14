# Dynamic Templates

Dynamic templates allow you to use request data in your stub responses at runtime. This feature enables creating more realistic and flexible mock responses that adapt based on the incoming request.

## Overview

Dynamic templates use Go's `text/template` syntax to process request data and generate responses. Templates are processed at runtime, not at load time, allowing for real-time data substitution.

## Basic Syntax

### Request Data Access
Use `{{.Request.field}}` to access request data:

```yaml
- service: example.Service
  method: GetUser
  input:
    matches:
      id: "\\d+"
  output:
    data:
      id: "{{.Request.id}}"
      name: "User {{.Request.id}}"
      email: "user{{.Request.id}}@example.com"
```

### Header Access
Use `{{.Headers.field}}` to access request headers:

```yaml
- service: example.Service
  method: GetUser
  input:
    equals:
      id: "admin"
  output:
    headers:
      x-user-role: "{{.Headers.authorization | split \" \" | index 1 | upper}}"
    data:
      id: "{{.Request.id}}"
      role: "admin"
```

## Template Functions

GripMock provides several built-in template functions:

### String Functions
- `upper(s)`: Convert to uppercase
- `lower(s)`: Convert to lowercase
- `title(s)`: Convert to title case
- `split(s, sep)`: Split string by separator
- `join(slice, sep)`: Join string slice with separator
- `index(slice, i)`: Get element at index from slice

### Math Functions
- `len(slice)`: Get length of slice
- `mul(a, b)`: Multiply two numbers
- `add(a, b)`: Add two numbers
- `sub(a, b)`: Subtract two numbers
- `div(a, b)`: Divide two numbers (returns 0 for division by zero)

### Slice Functions
- `sum(slice)`: Sum all values in a slice
- `mul(slice)`: Multiply all values in a slice
- `avg(slice)`: Calculate average of all values in a slice
- `min(slice)`: Find minimum value in a slice
- `max(slice)`: Find maximum value in a slice

### Time Functions
- `now()`: Get current time (changes for each message)
- `requestTime()`: Get atomic request time (same for all templates in one request)
- `unix(t)`: Convert time to Unix timestamp
- `format(t, layout)`: Format time with layout

### Utility Functions
- `json(v)`: Convert value to JSON string

### State Management
You can access and modify request state directly using `.State`:
- `{{.State.key}}`: Get value from request state
- `{{setState "key" "value"}}`: Set value in request state (returns empty string)

State is isolated per request and can be used to track calculations across multiple template evaluations.

## Technical Parameters

Dynamic templates provide access to essential technical parameters:

### Core Parameters
- `{{.MessageIndex}}`: Current message index (0-based) for streaming
- `{{.RequestTime}}`: Atomic request time for consistent timestamps
- `{{.State}}`: Request-scoped state for tracking calculations across templates

### Streaming Context
- `{{.Requests}}`: Slice of all non-empty client messages for client streaming
- Use `{{len .Requests}}` to get the count of messages
- Use `{{(index .Requests 0).field}}` to access a specific message

## Streaming Support

### Unary Requests
Templates are processed once per request with full access to request data.

### Server Streaming
Templates are processed once before streaming starts. The same processed data is used for all stream messages.

### Client Streaming
Templates are processed after all client messages are received. You have access to:
- `{{.Requests}}`: All received non-empty messages
- `{{len .Requests}}`: Total number of messages
- `{{(index .Requests N)}}`: Access message by index, then field via `{{(index .Requests 0).value}}`
- The last message is used as primary `.Request`

### Bidirectional Streaming
Templates are processed for each message with:
- `{{.MessageIndex}}`: Current message index (0-based)
- Current message data as primary request data

## Examples

See the complete [ecommerce example](/examples/projects/ecommerce/) and [calculator example](/examples/projects/calculator/) for full demonstrations of dynamic templates with all streaming types.

### E-commerce Product Lookup
```yaml
- service: ecommerce.EcommerceService
  method: GetProduct
  input:
    matches:
      product_id: "PROD_\\d+"
      user_id: "USER_\\d+"
  output:
    data:
      product_id: "{{.Request.product_id}}"
      name: "Product {{.Request.product_id}}"
      description: "Dynamic product for user {{.Request.user_id}}"
      user_discount: "{{.Request.user_id | split \"_\" | index 1 | title}}"
```

### Order Creation with Dynamic ID
```yaml
- service: ecommerce.EcommerceService
  method: CreateOrder
  input:
    equals:
      user_id: "USER_123"
  output:
    data:
      order_id: "ORDER_{{.Request.user_id | split \"_\" | index 1}}_{{now | unix}}"
      user_id: "{{.Request.user_id}}"
      total_amount: "{{.Request.items | len | mul 25.50}}"
      status: "processing"
```

### Customer Support Chat
```yaml
- service: ecommerce.EcommerceService
  method: CustomerSupportChat
  input:
    equals:
      user_id: "USER_789"
  output:
    stream:
      - message_id: "MSG_{{.MessageIndex}}_SUPPORT"
        user_id: "SUPPORT_001"
        content: "Hello! I'm support agent for message {{.MessageIndex}}. How can I help you with: {{.Request.content}}"
        timestamp: "{{now | format \"2006-01-02T15:04:05Z\"}}"
        sender_type: "support"
```

### Mathematical Calculator with Real Calculations
```yaml
- service: calculator.CalculatorService
  method: CalculateAverage
  inputs:
    - matches:
        value: "\\d+(\\.\\d+)?"
    - matches:
        value: "\\d+(\\.\\d+)?"
    - matches:
        value: "\\d+(\\.\\d+)?"
  output:
    data:
      result: "{{avg (extract .Requests `value`)}}"
      count: "{{len .Requests}}"
      sum: "{{sum (extract .Requests `value`)}}"

- service: calculator.CalculatorService
  method: DivideNumbers
  inputs:
    - equals:
        value: 100.0
    - equals:
        value: 2.0
  output:
    data:
      result: "{{div (index (extract .Requests `value`) 0) (index (extract .Requests `value`) 1)}}"
      count: "{{len .Requests}}"
```

## Advanced Usage

### Conditional Responses
You can create different responses based on request data:

```yaml
# Different responses for different users
- service: example.Service
  method: GetUser
  input:
    equals:
      user_id: "USER_789"
  output:
    data:
      user_id: "SUPPORT_001"
      content: "Hello! I'm support agent for message {{.MessageIndex}}"

- service: example.Service
  method: GetUser
  input:
    equals:
      user_id: "USER_999"
  output:
    data:
      user_id: "SUPPORT_SPECIAL"
      content: "Special support for user 999, message {{.MessageIndex}}"
```

### Complex Calculations
```yaml
- service: example.Service
  method: CalculateTotal
  input:
    equals:
      user_id: "USER_123"
  output:
    data:
      total: "{{.Request.items | len | mul 25.50}}"
      discount: "{{.Request.user_tier | mul 0.1}}"
      final_total: "{{.Request.total | mul 0.9}}"
```

### Error Handling with Dynamic Messages
```yaml
- service: ecommerce.EcommerceService
  method: GetProduct
  input:
    equals:
      product_id: "INVALID_PROD"
      user_id: "USER_ERROR"
  output:
    error: "Product {{.Request.product_id}} not found for user {{.Request.user_id}}. Please check your request."
    code: 5
```

### State Management Example
```yaml
- service: example.Service
  method: ProcessOrder
  input:
    equals:
      user_id: "USER_123"
  output:
    data:
      order_id: "ORDER_{{.Request.user_id | split \"_\" | index 1}}_{{now | unix}}"
      processing_step: "{{if .State.step}}{{.State.step}}{{else}}1{{end}}"
      message: "{{setState \"step\" (add (.State.step | default 0) 1)}}Processing step {{.State.step}}"
```

## Implementation Details

### Template Processing Flow
1. **Detection**: Templates containing `{{.Request.}}`, `{{.Headers.}}`, `{{.MessageIndex}}`, `{{.Requests.}}`, or `{{.State}}` are identified as dynamic
2. **Processing**: Dynamic templates are processed at runtime, not at load time
3. **Execution**: Go's `text/template` engine processes templates with custom functions
4. **Integration**: Processed data is integrated into gRPC responses

### YAML Processing
- Dynamic templates are detected and preserved during YAML → JSON conversion
- Static templates (no `.Request/.Headers/.MessageIndex/.Requests/.State`) are processed at load time
- Dynamic evaluation happens only at runtime

## Backward Compatibility

Dynamic templates are fully backward compatible:
- Static templates (without `{{.Request.}}` or `{{.Headers.}}`) continue to work unchanged
- No migration required for existing stubs
- Dynamic templates are opt-in only

## Performance Considerations

- Template processing happens at runtime, so there's a small performance impact
- Complex template functions may impact performance
- Template errors return gRPC internal errors
- Consider caching for high-throughput scenarios

## Thread Safety and Atomicity

### Atomic Functions
All template functions are designed to be thread-safe and atomic:
- **Mathematical functions** (`add`, `mul`, `sub`, `div`, `len`): Pure functions, no side effects
- **String functions** (`upper`, `lower`, `split`, `join`): Pure functions, no side effects
- **Time functions**: `now()` returns current time for each message, `requestTime()` uses atomic time within a single request

### Race Condition Prevention
- Each request gets its own `TemplateData` instance
- Time functions use atomic timestamps within a single request
- No shared state between concurrent requests
- Template processing is isolated per request

## Error Handling

Template errors are handled gracefully:
- Invalid template syntax returns gRPC internal errors
- Missing request fields are treated as empty strings
- Template processing errors are logged for debugging
- Division by zero returns 0 instead of causing errors
- For server streaming with `output.stream` and `output.error`/`output.code` set: stream messages are sent first, then the error is returned. If `output.stream` is empty, the error is returned immediately

## Best Practices

1. **Use meaningful field names**: Make templates readable and maintainable
2. **Test thoroughly**: Verify templates work with different request data
3. **Keep templates simple**: Avoid overly complex template logic
4. **Use appropriate functions**: Choose the right template function for your use case
5. **Consider performance**: Be mindful of template complexity in high-throughput scenarios
6. **Handle edge cases**: Consider division by zero, missing fields, etc.
7. **Use state management**: Leverage state functions for complex calculations

## Limitations

- Template errors cause gRPC internal errors
- State is request-scoped and not persisted between requests

## Migration Guide

### From Static to Dynamic Templates

**Before (Static)**:
```yaml
output:
  data:
    id: "123"
    name: "User 123"
```

**After (Dynamic)**:
```yaml
output:
  data:
    id: "{{.Request.id}}"
    name: "User {{.Request.id}}"
```

### Testing Dynamic Templates

```bash
# Start server
go run main.go examples/projects/calculator/service.proto --stub examples/projects/calculator

# Run tests
grpctestify examples/projects/calculator
```

## Important Notes

- Do not use dynamic templates inside `input.equals`, `input.contains`, or `input.matches`. Matching expressions must be static (plain strings, numbers, or regex strings). Use dynamic templates only in the `output` section

## Additional Functions

Along with the functions listed above, the following helpers are available:

- `extract(messages, field)` → returns a slice with `field` extracted from each message (e.g., `extract .Requests "value"`)
- `sprintf`, `str`, `int`, `int64`, `float`, `round`, `floor`, `ceil`