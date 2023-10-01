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

[Input Matching](matching-rule-input.md ':include')

[Headers Matching](matching-rule-headers.md ':include')
