# Stub API. Get Stubs List

Stubs List — endpoint returns a list of all registered stub files. It can be helpful to debbug your integration tests.

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

Enough to knock on the handle `GET /api/stubs`:
```bash
curl http://127.0.0.1:4771/api/stubs
```

Response:
```json
[
  {
    "id": "6c85b0fa-caaf-4640-a672-f56b7dd8074d",
    "service": "Gripmock",
    "method": "SayHello",
    "input": {
      "equals": {
        "name": "gripmock"
      },
      "contains": null,
      "matches": null
    },
    "output": {
      "data": {
        "message": "Hello GripMock",
        "return_code": 42
      },
      "error": ""
    }
  }
]
```
