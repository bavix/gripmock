---
title: Dynamic Templates
---

# Dynamic Templates <VersionTag version="v3.4.0" />

A stub response can read the request that triggered it. Templates use Go's
`text/template` syntax and are evaluated per request, not at load time.

## Basic Syntax

### Request Data Access
Use <code v-pre>`{{.Request.field}}`</code> to access request data:

::: v-pre
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
:::

## Structural Response Templates

Regular dynamic templates replace scalar values in an already defined response structure. Use `dataTemplate` when the request must also determine the structure itself, such as the number of elements in a response array.

The complete template is rendered at request time and then decoded as YAML or JSON. All regular template context and functions remain available.

::: v-pre
```yaml
- service: example.ItemService
  method: ProcessItems
  input:
    matches:
      items: ".+"
  output:
    dataTemplate: |
      items:
      {{ range .Request.items }}
        - id: {{ .id | json }}
          generated_id: {{ faker.Identity.UUID | json }}
      {{ else }}
        []
      {{ end }}
```
:::

An empty request array produces `items: []`; otherwise the response contains one element for every request item. `dataTemplate` is available for unary and client-streaming responses and cannot be combined with `data`.

For server-streaming and bidirectional-streaming methods, use `streamTemplate`. Its rendered result must be an array of response messages:

::: v-pre
```yaml
output:
  streamTemplate: |
    {{ range .Request.items }}
    - id: {{ .id | json }}
      status: processed
    {{ else }}
    []
    {{ end }}
```
:::

`streamTemplate` cannot be combined with `stream`. Dynamically generated stream elements support the same `_gripmock` delay and error directives as static stream elements.

If template execution fails, the rendered document is invalid YAML/JSON, or `streamTemplate` does not produce an array, GripMock returns an `INTERNAL` error for that call.

### Header Access
Use <code v-pre>`{{.Headers.field}}`</code> to access request headers:

::: v-pre
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
:::

## Template Functions

Go's own template builtins (`len`, `index`, `if`, `range`, …) are available.
GripMock adds these:

### String Functions
- `upper(s)`, `lower(s)`, `title(s)`: change case
- `split(s, sep)`: split a string into a slice
- `join(slice, sep)`: join a slice into a string
- `sprintf(format, args...)`, `str(v)`: format a value

### Math Functions
- `add`, `sub`, `div`, `mod`: two numbers. `div` returns 0 for division by zero
- `sum`, `mul`, `avg`, `min`, `max`: variadic — either spread numbers
  (<code v-pre>{{sum 1 2 3}}</code>) or one slice (<code v-pre>{{sum .Request.items}}</code>)
- `int`, `int64`, `float`, `round`, `floor`, `ceil`, `decimal`: numeric conversion
- `eq`, `gt`, `gte`, `lt`, `lte`: comparison

### Time Functions
- `now()`: current time, re-evaluated per message
- `unix(t)`: time to Unix timestamp
- `format(t, layout)`: format a time

For a timestamp that stays identical across every template in one request, use
the `.RequestTime` field rather than `now()`.

### Utility Functions
- `json(v)`: value to JSON string
- `extract(messages, field)`: pull `field` out of each message —
  <code v-pre>{{extract .Requests "value"}}</code>
- `uuid`, `uuid2base64`, `uuid2bytes`, `uuid2int64`, `string2base64`,
  `bytes2base64`, `bytes`: identifier and encoding conversion
- `faker.*`: see the [Faker reference](./faker)

### Plugin Functions <VersionTag version="v3.5.0" />
Custom functions provided by plugins are also available in templates. Load plugins using the `--plugins` flag and use their functions just like built-in functions.

**Example with hash plugin:**
::: v-pre
```yaml
output:
  data:
    hash: "{{.Request.data | sha256}}"
    checksum: "{{.Request.data | crc32}}"
```
:::

**Example with math plugin:**
::: v-pre
```yaml
output:
  data:
    result: "{{pow .Request.base .Request.exponent}}"
    sqrt: "{{sqrt .Request.value}}"
```
:::

See [Plugins](../plugins/) for more information on creating and using custom plugin functions.

### Built-in Faker Object <VersionTag version="v3.10.0" />

GripMock ships with a built-in `faker` object for realistic dynamic values.

Available semantic groups include:

- `faker.Person` (name, age, gender, ...)
- `faker.Contact` (email, phone, username, url)
- `faker.Geo` (country, city, latitude, longitude, ...)
- `faker.Network` (ip, domain, user-agent, http status/method)
- `faker.Company`, `faker.Commerce`, `faker.Text`, `faker.DateTime`, `faker.Identity`

See full key-by-key reference in [Faker Reference](/guide/stubs/faker).

Example:

::: v-pre
```yaml
- service: example.UserService
  method: GetProfile
  input:
    matches:
      id: "\\d+"
  output:
    data:
      id: "{{.Request.id}}"
      first_name: "{{faker.Person.FirstName}}"
      last_name: "{{faker.Person.LastName}}"
      email: "{{faker.Contact.Email}}"
      city: "{{faker.Geo.City}}"
      lat: "{{faker.Geo.Latitude}}"
      lon: "{{faker.Geo.Longitude}}"
      ip: "{{faker.Network.IPv4}}"
      user_agent: "{{faker.Network.UserAgent}}"
      account_id: "{{faker.Identity.UUID}}"
```
:::

## Technical Parameters

### Core Parameters
- <code v-pre>`{{.MessageIndex}}`</code>: Current message index (0-based) for streaming
- <code v-pre>`{{.RequestTime}}`</code>: Request time, identical for every template in one request (alias: <code v-pre>`{{.Timestamp}}`</code>)
- <code v-pre>`{{.StubID}}`</code>: UUID of the stub that matched (alias: <code v-pre>`{{.RequestID}}`</code>)

### Streaming Context
- <code v-pre>`{{.Requests}}`</code>: Slice of all non-empty client messages for client streaming
- Use <code v-pre>`{{len .Requests}}`</code> to get the count of messages
- Use <code v-pre>`{{(index .Requests 0).field}}`</code> to access a specific message

## Streaming Support

### Unary Requests
Templates are processed once per request with full access to request data.

### Server Streaming
Templates are processed once before streaming starts. The same processed data is used for all stream messages.

### Client Streaming
Templates are processed after all client messages are received. You have access to:
- <code v-pre>`{{.Requests}}`</code>: All received non-empty messages
- <code v-pre>`{{len .Requests}}`</code>: Total number of messages
- <code v-pre>`{{(index .Requests N)}}`</code>: Access message by index, then field via <code v-pre>`{{(index .Requests 0).value}}`</code>
- The last message is used as primary `.Request`

### Bidirectional Streaming
Templates are processed for each message with:
- <code v-pre>`{{.MessageIndex}}`</code>: Current message index (0-based)
- Current message data as primary request data

## Examples

See the complete ecommerce and calculator examples in the `examples/projects/` directory for full demonstrations of dynamic templates with all streaming types.

### E-commerce Product Lookup
::: v-pre
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
:::

### Order Creation with Dynamic ID
::: v-pre
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
:::

### Customer Support Chat
::: v-pre
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
:::

### Mathematical Calculator with Real Calculations
::: v-pre
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
:::

## Advanced Usage

### Conditional Responses
You can create different responses based on request data:

::: v-pre
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
:::

### Complex Calculations
::: v-pre
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
:::

### Error Handling with Dynamic Messages
::: v-pre
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
:::

## Implementation Details

### Template Processing Flow
1. **Detection**: Templates containing <code v-pre>`{{.Request.}}`</code>, <code v-pre>`{{.Headers.}}`</code>, <code v-pre>`{{.MessageIndex}}`</code>, <code v-pre>`{{.Requests.}}`</code>, or <code v-pre>`{{.State}}`</code> are identified as dynamic
2. **Processing**: Dynamic templates are processed at runtime, not at load time
3. **Execution**: Go's `text/template` engine processes templates with custom functions
4. **Integration**: Processed data is integrated into gRPC responses

### YAML Processing
- Dynamic templates are detected and preserved during YAML → JSON conversion
- Static templates (no `.Request/.Headers/.MessageIndex/.Requests/.State`) are processed at load time
- Dynamic evaluation happens only at runtime

## Backward Compatibility

Dynamic templates are fully backward compatible:
- Static templates (without <code v-pre>`{{.Request.}}`</code> or <code v-pre>`{{.Headers.}}`</code>) continue to work unchanged
- No migration required for existing stubs
- Dynamic templates are opt-in only

## Thread Safety

Every request builds its own template data. Nothing is shared between concurrent
requests, and the functions themselves are pure. `now()` re-reads the clock per
message; `.RequestTime` is fixed for the whole request.

## Error Handling

Template errors are handled gracefully:
- Invalid template syntax returns gRPC internal errors
- Missing request fields are treated as empty strings
- Template processing errors are logged for debugging
- Division by zero returns 0 instead of causing errors
- For server streaming with `output.stream` and `output.error`/`output.code` set: stream messages are sent first, then the error is returned. If `output.stream` is empty, the error is returned immediately

## Migration Guide

### From Static to Dynamic Templates

**Before (Static)**:
::: v-pre
```yaml
output:
  data:
    id: "123"
    name: "User 123"
```
:::

**After (Dynamic)**:
::: v-pre
```yaml
output:
  data:
    id: "{{.Request.id}}"
    name: "User {{.Request.id}}"
```
:::

### Testing Dynamic Templates

```bash
# Start server
go run main.go examples/projects/calculator/service.proto --stub examples/projects/calculator

# Run tests
grpctestify examples/projects/calculator
```

## Important Notes

Templates belong in `output` only. Every matcher — `input.equals`,
`input.contains`, `input.matches`, `input.glob`, `input.anyOf`, and their
`headers` counterparts — must be static: plain strings, numbers, regex or glob
patterns.
