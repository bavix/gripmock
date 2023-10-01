## Rest API. Stubs Search

Stubs Search — endpoint helps to flexibly search for stubs in the stub storage.

Let's imagine that our contract `simple.proto` looks something like this:
```protobuf
syntax = "proto3";
option go_package = "github.com/bavix/gripmock/protogen/example/simple";

package simple;

service Gripmock {
  rpc SayHello (Request) returns (Reply);
}

message Request {
  string name = 1;
}

message Reply {
  string message = 1;
  int32 return_code = 2;
}
```

Enough to knock on the handle POST /api/stubs/search:

```json
{
  "service": "Greeter",
  "method": "SayHello",
  "data": {
    "name": "gripmock"
  }
}
```

Response:
```json
{
  "data":{
    "message": "World",
    "return_code": 0
  },
  "error": ""
}
```

## Find by ID

Enough to knock on the handle `POST /api/stubs/search`:
```bash
curl -X POST -d '{ \
  "id": "6c85b0fa-caaf-4640-a672-f56b7dd8074d", \
  "service": "Gripmock", \
  "method": "SayHello", \
  "data":{} \
}' http://127.0.0.1:4771/api/stubs/search
```

Response:
```json
{
  "data":{
    "message": "World",
    "return_code": 0
  },
  "error": ""
}
```

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

## Headers Matching
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

### Headers Matching Rule
Headers matching has 3 rules to match an input: **equals**,**contains** and **matches**
<br>
Headers can consist of a key and a value. If there are several values, then you need to list them separated by ";". Data type string.
<br>
**Gripmock** recursively goes over the fields and tries to match with given input.
<br>
**equals** will match the exact field name and value of input into expected stub. example stub JSON:
```json
{
  .
  .
  "headers":{
    "equals":{
      "authorization": "mytoken",
      "system": "ec071904-93bf-4ded-b49c-d06097ddc6d5"
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
  "headers":{
    "contains":{
      "field2":"hello"
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
  "headers":{
    "matches":{
      "name":"^grip.*$"
    }
  }
  .
  .
}
```
