## Static Stubs. How is yaml better than json?

Let's talk about the pros of yaml, shall we?
- Short syntax that is difficult to get confused;
- Support for Anchor and Alias;
- Additional features to be added to the syntax;

Let's talk about each item separately.

### Short Syntax

I think there is nothing to discuss here. Let's look at the json format:
```json
[
  {
    "service": "Gripmock",
    "method": "SayHello",
    "input": {
      "equals": {
        "name": "tokopedia"
      }
    },
    "output": {
      "data": {
        "message": "Hello Tokopedia",
        "return_code": 1
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
        "return_code": 1
      }
    }
  }
]
```

And now the same thing, but in yaml:
```yaml
- service: Gripmock
  method: SayHello
  input:
    equals:
      name: tokopedia
  output:
    data:
      message: Hello Tokopedia
      return_code: 1
- service: Gripmock
  method: SayHello
  input:
    equals:
      name: world
  output:
    data:
      message: Hello World
      return_code: 1
```

### Anchor and Alias

You can read more details here: https://github.com/goccy/go-yaml#2-reference-elements-declared-in-another-file

```yaml
- service: &service Gripmock
  method: &method SayHello
  input:
    equals:
      name: tokopedia
      code: &code 0ad1348f1403169275002100356696
  output:
    data: &result
      message: Hello Tokopedia
      return_code: 1
- service: *service
  method: *method
  input:
    equals:
      name: world
      code: *code
  output:
    data: *result
```

### Additional functions

You know these standards, right? Each developer starts creating their own guide standard, and you have to mock it all.

Yes, Yes. I even had to do something like this: https://bavix.github.io/uuid-ui/

Option 1:
```protobuf
message .... {
  bytes uuid = 1;
}
```

Option 2:
```protobuf
message UUID {
  int64 high = 1;
  int64 low = 2;
}
```

Option 3 (there is such an implementation, but so far it is not in the functions):
```protobuf
message UUID {
  uint64 high = 1;
  uint64 low = 2;
}
```

I find it more convenient to read guides than the internal representation of a guide in a service.

For the first option, it is enough to pack in base64:
```yaml
base64: {{ uuid2base64 "77465064-a0ce-48a3-b7e4-d50f88e55093" }}
```

Result:
```json
{"base64": "d0ZQZKDOSKO35NUPiOVQkw=="}
```

For the second option, it is enough to pack in high-low:
```yaml
highLow: {{ uuid2int64 "e351220b-4847-42f5-8abb-c052b87ff2d4" }}
```

Result:
```json
{"highLow": {"high":-773977811204288029,"low":-3102276763665777782}}
```

But what if you need to pass a string in bytes (base64)? Let's transform.
```yaml
string: {{ string2base64 "hello world" }}
bytes: {{ bytes "hello world" | bytes2base64 }}
```

Result:
```json
{
  "string": "aGVsbG8gd29ybGQ=",
  "bytes": "aGVsbG8gd29ybGQ="
}
```

New features will be added as needed.

It worked! 
