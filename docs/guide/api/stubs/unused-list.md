# Stub API. Get Stubs Unused List

Stubs Unused List — endpoint returns a list of unused stubs (all stubs that were not accessed through search).
A very useful method that helps find dead stubs in the code.

Let's imagine that our contract `simple.proto` looks something like this:
```proto
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

## Search Query

Enough to knock on the handle `GET /api/stubs/unused`:
```bash
curl http://127.0.0.1:4771/api/stubs/unused
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

## Checking

Find stub by ID. Enough to knock on the handle `POST /api/stubs/search`:
```bash
curl -X POST -d '{ \
  "id": "6c85b0fa-caaf-4640-a672-f56b7dd8074d", \
  "service": "Gripmock", \
  "method": "SayHello", \
  "data":{} \
}' http://127.0.0.1:4771/api/stubs/search
```

Now the stub is marked as used. Let's try to get a list of unused stubs.
```bash
curl http://127.0.0.1:4771/api/stubs/unused
```

Response:
```json
[]
```
