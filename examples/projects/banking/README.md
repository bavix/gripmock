# Banking PaymentsService Example

This example demonstrates realistic banking payment flows using the universal v4 stub format with all streaming types and dynamic templates.

## Features Demonstrated

- **Universal v4 Format**: Complete v4 stub implementation with `outputs` for different RPC types
- **All Streaming Types**: Unary, server streaming, client streaming, and bidirectional streaming
- **Dynamic Templates**: Use `{{.Request.field}}`, `{{.Requests}}`, and `{{.MessageIndex}}`
- **Real-world Banking Scenarios**: Payment processing, statement retrieval, receipt upload, fraud detection
- **Sequence Rules**: Bidirectional streaming with sequence-based responses

## Project Structure

The banking example demonstrates a complete payments service:

- `service.proto` - gRPC service definition for PaymentsService
- `stubs.yaml` - v4 format stubs with dynamic templates
- `stubs_legacy.yaml` - Legacy format stubs for backward compatibility
- `*.gctf` - Test scenarios for each method

## Service Methods

### Unary Methods
- **ProcessPayment**: Process single payment transactions with approval logic

### Server Streaming
- **GetStatement**: Stream transaction statements with dynamic transaction IDs

### Client Streaming
- **UploadReceipts**: Process multiple receipt uploads with batch processing

### Bidirectional Streaming
- **FraudAlerts**: Real-time fraud detection with sequence-based responses

## Dynamic Template Examples

### Basic Request Data Access
```yaml
user_id: "{{.Request.user_id}}"
merchant_id: "{{.Request.merchant_id}}"
amount: "{{.Request.amount}}"
```

### Template Functions
```yaml
transaction_id: "TX-{{.Request.user_id | split \"-\" | index 1}}"
upload_id: "upload_{{.Request.receipt_type}}_{{now | unix}}"
```

### Streaming Context
```yaml
event_id: "e{{.MessageIndex}}"
action: "{{if eq .MessageIndex 0}}ALLOW{{else}}REVIEW{{end}}"
```

### Sequence Rules
```yaml
sequence:
  - send:
      action: "ALLOW"
  - send:
      action: "REVIEW"
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
- `now()`: Get current time
- `unix(t)`: Convert time to Unix timestamp
- `format(t, layout)`: Format time with layout

## Running the Example

1. Start GripMock with the example:
   ```bash
   go run main.go examples/projects/banking/service.proto --stub examples/projects/banking
   ```

2. Run individual tests:
   ```bash
   grpctestify examples/projects/banking/case_unary_payment.gctf
   grpctestify examples/projects/banking/case_server_stream_statement.gctf
   grpctestify examples/projects/banking/case_client_stream_upload.gctf
   grpctestify examples/projects/banking/case_bidi_fraud_sequence.gctf
   ```

3. Run all tests:
   ```bash
   grpctestify examples/projects/banking
   ```

## Test Scenarios

### Unary Test
- **ProcessPayment**: Tests payment processing with approval logic

### Server Streaming Test
- **GetStatement**: Tests streaming transaction statements with dynamic transaction IDs

### Client Streaming Test
- **UploadReceipts**: Tests processing multiple receipt uploads

### Bidirectional Streaming Test
- **FraudAlerts**: Tests real-time fraud detection with sequence-based responses

## Key Features

### Sequence Rules
For bidirectional streaming, responses follow a specific sequence:
```yaml
sequence:
  - send:
      action: "ALLOW"
  - send:
      action: "REVIEW"
```

### Dynamic Transaction IDs
Transaction IDs are generated dynamically based on request data:
```yaml
transactionId: "TX-{{.Request.user_id | split \"-\" | index 1}}"
```

### Batch Processing
Client streaming supports batch processing of multiple receipts:
```yaml
uploaded: true
status: "RECEIVED"
```

## Verification Points

1. **Template Processing**: All dynamic templates are processed correctly
2. **Streaming Types**: All streaming types work with v4 format
3. **Sequence Rules**: Bidirectional streaming correctly follows sequence rules
4. **Backward Compatibility**: Legacy stubs continue to work
5. **Real-world Scenarios**: Banking-specific use cases are properly handled

## Limitations

- Template processing happens at runtime
- Complex functions may impact performance
- Template errors return gRPC internal errors
- Limited to string-based template processing
