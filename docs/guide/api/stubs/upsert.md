# Stub API. Stubs Upsert

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

Stubs Upsert — endpoint adds or updates a stub via http api using its ID..

Enough to knock on the handle `POST /api/stubs`. One stub:
```bash
curl -X POST -d '{"service":"Gripmock","method":"SayHello","input":{"equals":{"name":"gripmock"},"contains":null,"matches":null},"output":{"data":{"message":"Hello GripMock","return_code":42},"error":""}}' http://127.0.0.1:4771/api/stubs
```

Response:
```json
["6c85b0fa-caaf-4640-a672-f56b7dd8074d"]
```

Stack of stubs:
```bash
curl -X POST -d '[{"service":"Gripmock","method":"SayHello","input":{"equals":{"name":"gripmock1"},"contains":null,"matches":null},"output":{"data":{"message":"Hello GripMock. stab1","return_code":42},"error":""}},{"service":"Gripmock","method":"SayHello","input":{"equals":{"name":"gripmock2"},"contains":null,"matches":null},"output":{"data":{"message":"Hello GripMock. stab2","return_code":42},"error":""}}]' http://127.0.0.1:4771/api/stubs
```

Response:
```json
["2378ccb8-f36e-48b0-a257-4309876bed47", "0ee02a07-4cae-4a0b-b0c1-5e7c379bc858"]
```
