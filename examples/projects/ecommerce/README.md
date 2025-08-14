# E-commerce Dynamic Stubs Example

This example demonstrates dynamic template functionality in GripMock using a realistic e-commerce scenario with all streaming types.

## Features Demonstrated

- **Dynamic Templates**: Use `{{.Request.field}}`, `{{.Requests}}`, and `{{.MessageIndex}}`
- **Template Functions**: Advanced functions like `len`, `sum`, `avg`, `mul`, `add`, `now`, `unix`, `format`, `extract`
- **All Streaming Types**: Complete coverage of unary, server streaming, client streaming, and bidirectional streaming
- **Real-world Scenarios**: E-commerce use cases like product lookup, order creation, chat support

## Project Structure

The ecommerce example is organized by method with separate directories:

- `get_product/`: Product lookup with personalized discounts and dynamic error handling (`output.error` + `output.code`)
- `create_order/`: Order creation with dynamic ID generation and total calculation
- `get_order_history/`: Server streaming order history with dynamic user-specific data
- `submit_product_reviews/`: Client streaming reviews with aggregation and validation
- `customer_support_chat/`: Bidirectional streaming chat with message index support

## Service Methods

### Unary Methods
- **GetProduct**: Retrieve product information with personalized discounts
- **CreateOrder**: Create orders with dynamic ID generation and total calculation

### Server Streaming
- **GetOrderHistory**: Stream order history with dynamic user-specific data

### Client Streaming
- **SubmitProductReviews**: Process multiple reviews with aggregation
- **SubmitProductReviews (multiple inputs)**: Process batch reviews with validation

### Bidirectional Streaming
- **CustomerSupportChat**: Real-time chat with message index support

## Dynamic Template Examples

### Basic Request Data Access
```yaml
product_id: "{{.Request.product_id}}"
name: "Product {{.Request.product_id}}"
description: "Dynamic product for user {{.Request.user_id}}"
```

### Template Functions
```yaml
order_id: "ORDER_{{.Request.user_id | split \"_\" | index 1}}_{{now | unix}}"
total_amount: "{{.Request.items | len | mul 25.50}}"
user_discount: "{{.Request.user_id | split \"_\" | index 1 | title}}"
```

### Time Functions
```yaml
timestamp: "{{now | format \"2006-01-02T15:04:05Z\"}}"
submission_id: "REVIEW_{{len .Requests}}_{{now | unix}}"
```

### Streaming Context
```yaml
message_id: "MSG_{{.MessageIndex}}_SUPPORT"
content: "Hello! I'm support agent for message {{.MessageIndex}}"
```

## Available Template Functions

- `json(v)`: Convert value to JSON string
- `join(slice, sep)`: Join string slice with separator
- `split(s, sep)`: Split string by separator
- `upper(s)`: Convert to uppercase
- `lower(s)`: Convert to lowercase
- `title(s)`: Convert to title case
- `index(slice, i)`: Get element at index from slice
- `len(slice)`: Get length of slice
- `mul(...)`: Multiply numbers (variadic)
- `add(a, b)`: Add two numbers
- `now()`: Get current time
- `unix(t)`: Convert time to Unix timestamp
- `format(t, layout)`: Format time with layout

## Running the Example

1. Start GripMock with the example:
   ```bash
    go run main.go examples/projects/ecommerce/service.proto --stub examples/projects/ecommerce
   ```

2. Run individual tests:
   ```bash
    grpctestify examples/projects/ecommerce/get_product/case_unary_get_product.gctf
    grpctestify examples/projects/ecommerce/create_order/case_unary_create_order.gctf
    grpctestify examples/projects/ecommerce/get_order_history/case_server_streaming.gctf
    grpctestify examples/projects/ecommerce/submit_product_reviews/case_client_streaming_simple.gctf
    grpctestify examples/projects/ecommerce/customer_support_chat/case_bidi_chat_simple.gctf
    grpctestify examples/projects/ecommerce/customer_support_chat/case_bidi_special_user.gctf
   ```

3. Run all tests:
   ```bash
   grpctestify examples/projects/ecommerce
   ```

## Test Scenarios

### Unary Tests
- **GetProduct**: Tests dynamic product lookup with user-specific data
- **CreateOrder**: Tests order creation with dynamic ID generation and total calculation

### Server Streaming Test
- **GetOrderHistory**: Tests streaming order history with dynamic user-specific order IDs

### Client Streaming Test
- **SubmitProductReviews**: Tests processing multiple reviews with message count aggregation

### Bidirectional Streaming Test
- **CustomerSupportChat**: Tests real-time chat with message index support
- **Special User Test**: Verifies that different users get different responses

## Key Features

### Message Index Support
For bidirectional streaming, each message gets a unique index:
```yaml
message_id: "MSG_{{.MessageIndex}}_SUPPORT"
content: "Hello! I'm support agent for message {{.MessageIndex}}"
```

### User-Specific Responses
Different users get different responses based on their ID:
```yaml
# For USER_789
user_id: "SUPPORT_001"
content: "Hello! I'm support agent for message 0..."

# For USER_999
user_id: "SUPPORT_SPECIAL"
content: "Special support for user 999, message 0..."
```

### Dynamic Calculations
Real-time calculations based on request data:
```yaml
total_amount: "{{.Request.items | len | mul 25.50}}"
reviews_count: "{{len (extract .Requests `rating`)}}"
```

## Error Handling in Streams

If both `output.stream` and `output.error`/`output.code` are specified for a server streaming method:
- All stream messages are sent first, then the RPC is terminated with the specified error
- If `output.stream` is empty, the error is returned immediately

## Verification Points

1. **Template Processing**: All dynamic templates are processed correctly
2. **User Isolation**: Different users get appropriate responses
3. **Message Index**: Bidirectional streaming correctly uses message indices
4. **Streaming Types**: All streaming types work with dynamic templates
5. **Function Support**: All template functions work as expected
6. **Backward Compatibility**: Static templates continue to work

## Limitations

- Template processing happens at runtime
- Complex functions may impact performance
- Template errors return gRPC internal errors
- Limited to string-based template processing
