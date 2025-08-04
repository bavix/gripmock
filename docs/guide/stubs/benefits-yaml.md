# Why YAML?

YAML makes working with GripMock simple and enjoyable. Let's explore why this format is the perfect choice for your gRPC mocking needs.

## 1. Clean and Readable Syntax

YAML removes unnecessary punctuation, making your configuration crystal clear at first glance:

**Here's how it would look in JSON:**
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

**And here's the same thing in YAML:**
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

See the difference? No more brackets and commas cluttering your code - just clean, readable configuration!

---

## 2. Streaming Support Made Simple

YAML handles all types of gRPC streaming scenarios with ease:

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
  stream:
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
  stream:
    - equals:
        user_id: "alice"
        message: "Hello!"
    - equals:
        user_id: "alice"
        message: "How are you?"
  output:
    stream:
      - data:
          user_id: "bot"
          message: "Hello, Alice!"
      - data:
          user_id: "bot"
          message: "I'm doing great!"
```

---

## 3. Flexible Matching Options

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

## 4. Reusable Components

YAML lets you create templates and reuse them throughout your configuration:

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

## 5. Built-in Data Transformations

YAML supports built-in functions for handling complex data conversions:

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

## 6. Version Compatibility

GripMock automatically supports both old and new formats:

### Old Format (V1) - Still Works
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

### New Format (V2) - Recommended
```yaml
- service: ChatService
  method: SendMessage
  stream:
    - equals:
        user: Alice
        text: "Hello"
  output:
    data:
      reply: "Hello, Alice!"
```

**Important**: GripMock automatically detects the format and works with both!

---

## 7. Priority Control

Control which stub should be selected first:

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

## What Makes YAML Special?

- 🔄 **Readability**: No more cluttered brackets and commas
- ♻️ **Reusability**: Create templates and use them over and over
- 🛠 **Flexibility**: Built-in functions for UUID, Base64, and other formats
- 🔧 **Compatibility**: Works with both `.yaml` and `.yml` files
- 📡 **Streaming**: Excellent support for all gRPC streaming types
- 🎯 **Matching**: Flexible settings for exact, partial, and regex matching
- 🔄 **Backward Compatibility**: Automatic support for old and new formats
- ⚡ **Performance**: Efficient ranking and matching algorithms

For advanced transformation functions, check out the [UUID Utilities Documentation](https://bavix.github.io/uuid-ui/).  
