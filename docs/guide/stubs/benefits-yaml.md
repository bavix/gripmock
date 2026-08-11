# Why YAML?

GripMock reads stubs from both JSON and YAML. YAML is the shorter of the two,
and it carries anchors, comments and multi-line strings that JSON does not.

## 1. Less punctuation

The same stub, first as JSON:

```json  
[
  {
    "service": "Gripmock",
    "method": "SayHello",
    "input": {
      "equals": {
        "name": "gripmock"
      }
    },
    "output": {
      "data": {
        "message": "Hello GripMock",
        "returnCode": 1
      }
    }
  }
]
```  

And as YAML:

```yaml  
- service: Gripmock
  method: SayHello
  input:
    equals:
      name: gripmock
  output:
    data:
      message: Hello GripMock
      returnCode: 1
```  

17 lines against 10, for the same stub.

---

## 2. Streaming

### Simple Request-Response
```yaml
- service: ChatService
  method: SendMessage
  input:
    equals:
      user: Alice
      text: "Hello!"
  output:
    data:
      reply: "Hello, Alice!"
      timestamp: "2024-01-01T12:00:00Z"
```

### File Upload in Chunks
```yaml
- service: UploadService
  method: UploadFile
  inputs:
    - equals:
        chunk_id: "file_001"
        sequence: 1
        total_chunks: 3
    - equals:
        chunk_id: "file_001"
        sequence: 2
        total_chunks: 3
    - equals:
        chunk_id: "file_001"
        sequence: 3
        total_chunks: 3
  output:
    data:
      success: true
      message: "File uploaded successfully!"
```

### Real-Time Chat
```yaml
- service: ChatService
  method: Chat
  inputs:
    - equals:
        user_id: "alice"
        message: "Hello!"
    - equals:
        user_id: "alice"
        message: "How are you?"
  output:
    stream:
      - user_id: "bot"
        message: "Hello, Alice!"
      - user_id: "bot"
        message: "I'm doing great!"
```

---

## 3. Matching options

### Ignore Array Order
```yaml
- service: UserService
  method: ProcessUsers
  input:
    ignoreArrayOrder: true  # Order doesn't matter
    equals:
      user_ids:
        - "user_001"
        - "user_002"
        - "user_003"
  output:
    data:
      processed: 3
      status: "Done!"
```

### Multiple Matching Strategies
```yaml
- service: SearchService
  method: Search
  input:
    equals:
      query: "gripmock"  # Exact match
    contains:
      tags: ["grpc", "mock"]  # Contains these tags
    matches:
      pattern: ".*test.*"  # Regular expression
  output:
    data:
      results: ["Found it!"]
```

---

## 4. Anchors

An anchor (`&name`) defines a value once; an alias (`*name`) reuses it. JSON has
no equivalent.

```yaml  
# Create a response template
- service: &service Gripmock
  method: &method SayHello
  input:
    equals:
      name: gripmock
      code: &code 0ad1348f1403169275002100356696
  output:
    data: &result
      message: Hello GripMock
      returnCode: 1

# Use the same template for another case
- service: *service
  method: *method
  input:
    equals:
      name: world
      code: *code
  output:
    data: *result
```  

---

## 5. Template functions

Stub values pass through the template engine, in YAML and JSON alike:

### UUID Handling
```yaml
# For bytes fields (Base64 encoding)
base64: {{ uuid2base64 "77465064-a0ce-48a3-b7e4-d50f88e55093" }}

# For int64 representations
highLow: {{ uuid2int64 "e351220b-4847-42f5-8abb-c052b87ff2d4" }}

# String to Base64 conversion
string: {{ string2base64 "hello world" }}
```

### Results
```json  
{
  "base64": "d0ZQZKDOSKO35NUPiOVQkw==",
  "highLow": {
    "high": -773977811204288029,
    "low": -3102276763665777782
  },
  "string": "aGVsbG8gd29ybGQ="
}
```  

---

## 6. Both stub formats

### V1 — `input`
```yaml
- service: ChatService
  method: SendMessage
  input:
    equals:
      user: Alice
      text: "Hello"
  output:
    data:
      reply: "Hello, Alice!"
```

### V2 — `inputs`
```yaml
- service: ChatService
  method: SendMessage
  inputs:
    - equals:
        user: Alice
        text: "Hello"
  output:
    data:
      reply: "Hello, Alice!"
```

GripMock accepts both, and picks the format from whichever field the stub carries.

---

## 7. Priority

When several stubs match, the higher `priority` wins:

```yaml
# High priority for important cases
- service: UserService
  method: GetUser
  priority: 100
  input:
    equals:
      user_id: "12345"
      exact_match: true
  output:
    data:
      name: "John Doe"
      priority: "high"

# Normal priority for others
- service: UserService
  method: GetUser
  priority: 50
  input:
    equals:
      user_id: "12345"
  output:
    data:
      name: "John Doe"
      priority: "normal"
```

---

Both `.yaml` and `.yml` extensions are read.

For the UUID conversion functions, see the
[UUID Utilities documentation](https://bavix.github.io/uuid-ui/).
