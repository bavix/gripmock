# Why YAML?

## 1. Concise Syntax
YAML eliminates unnecessary punctuation while maintaining readability:

**JSON Equivalent**  
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
  },
  {
    "service": "Gripmock",
    "method": "SayHello",
    "input": {
      "equals": {
        "name": "world"
      }
    },
    "output": {
      "data": {
        "message": "Hello World",
        "returnCode": 1
      }
    }
  }
]
```  

**YAML Simplification**  
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
- service: Gripmock
  method: SayHello
  input:
    equals:
      name: world
  output:
    data:
      message: Hello World
      returnCode: 1
```  

---

## 2. Reusable Components
Leverage anchors (`&`) and aliases (`*`) for DRY configurations:

```yaml  
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

## 3. Data Transformation
Built-in template functions handle complex conversions:

**UUID Handling**  
```yaml
# For bytes fields (Base64 encoding)
base64: {{ uuid2base64 "77465064-a0ce-48a3-b7e4-d50f88e55093" }}

# For int64 high/low representations
highLow: {{ uuid2int64 "e351220b-4847-42f5-8abb-c052b87ff2d4" }}

# String to Base64 conversion
string: {{ string2base64 "hello world" }}
bytes: {{ bytes "hello world" | bytes2base64 }}
```  

**Output Results**  
```json  
{
  "base64": "d0ZQZKDOSKO35NUPiOVQkw==",
  "highLow": {
    "high": -773977811204288029,
    "low": -3102276763665777782
  },
  "string": "aGVsbG8gd29ybGQ=",
  "bytes": "aGVsbG8gd29ybGQ="
}
```  

---

## 4. Key Notes
- 🔄 **Readability**: No braces/commas reduces visual noise  
- ♻️ **Reusability**: Shared components via anchors prevent duplication  
- 🛠 **Flexibility**: Template functions handle:  
  - UUID format conversions  
  - Base64 encoding/decoding  
  - Byte manipulation  
- 🔧 **Compatibility**: Works with both `.yaml` and `.yml` extensions  

For advanced template functions, see [UUID Utilities Documentation](https://bavix.github.io/uuid-ui/).  
