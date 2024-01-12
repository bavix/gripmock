## Input Matching
Stub will respond with the expected response only if the request matches any rule. Stub service will serve `/api/stubs/search` endpoint with format:
```json
{
  "service":"<service name>",
  "method":"<method name>",
  "data":{
    // input that suppose to match with stored stubs
  }
}
```
So if you do a `curl -X POST -d '{"service":"Greeter","method":"SayHello","data":{"name":"gripmock"}}' localhost:4771/api/stubs/search` stub service will find a match from listed stubs stored there.

### Input Matching Rule
Input matching has 3 rules to match an input: **equals**,**contains** and **matches**
<br>
Nested fields are allowed for input matching too for all JSON data types. (`string`, `bool`, `array`, etc.)
<br>
**Gripmock** recursively goes over the fields and tries to match with given input.
<br>

**ignoreArrayOrder** Disables sorting check inside arrays.
```yaml
- service: MicroService
  method: SayHello
  input:
    ignoreArrayOrder: true # disable sort checking
    equals:
      v1:
        - {{ uuid2base64 "77465064-a0ce-48a3-b7e4-d50f88e55093" }}
        - {{ uuid2base64 "99aebcf2-b56d-4923-9266-ab72bf5b9d0b" }}
        - {{ uuid2base64 "5659bec5-dda5-4e87-bef4-e9e37c60eb1c" }}
        - {{ uuid2base64 "ab0ed195-6ac5-4006-a98b-6978c6ed1c6b" }}
  output:
    data:
      code: 1000
```
Without this flag, the order of the transmitted values is important to us.

**equals** will match the exact field name and value of input into expected stub. example stub JSON:
```json
{
  .
  .
  "input":{
    "equals":{
      "name":"gripmock",
      "greetings": {
            "english": "Hello World!",
            "indonesian": "Halo Dunia!",
            "turkish": "Merhaba Dünya!"
      },
      "ok": true,
      "numbers": [4, 8, 15, 16, 23, 42]
      "null": null
    }
  }
  .
  .
}
```

**contains** will match input that has the value declared expected fields. example stub JSON:
```json
{
  .
  .
  "input":{
    "contains":{
      "field2":"hello",
      "field4":{
        "field5": "value5"
      } 
    }
  }
  .
  .
}
```

**matches** using regex for matching fields expectation. example:

```json
{
  .
  .
  "input":{
    "matches":{
      "name":"^grip.*$",
      "cities": ["Jakarta", "Istanbul", ".*grad$"]
    }
  }
  .
  .
}
```
