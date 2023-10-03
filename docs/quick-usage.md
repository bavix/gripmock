## Quick Usage

I suspect if you have reached this page, then you already have a grpc server and a proto contract. Do not delay the contract far, now you will need it.

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

At the moment, there is no standalone version of gripmock, only a docker image.

I will skip the details of installing docker and using it. Read documentation: https://docs.docker.com/engine/install/.

Let's start the GripMock server:
```bash
docker run -p 4770:4770 -p 4771:4771 -v ./simple.proto:/proto/simple.proto:ro bavix/gripmock /proto/simple.proto
```

After launch, you will see something like this: 
```bash
➜  simple git:(docs) ✗ docker run -p 4770:4770 -p 4771:4771 -v ./api:/proto:ro bavix/gripmock /proto/simple.proto
Starting GripMock
Serving stub admin on http://:4771
grpc server pid: 38
Serving gRPC on tcp://:4770
```

What is important to understand? 
1. gRPC Mock server started on port 4770;
2. HTTP server for working with the stub server is running on port 4771;

This means that everything went well. Now let's add the first stub:
```bash
curl -X POST -d '{"service":"Gripmock","method":"SayHello","input":{"equals":{"name":"gripmock"}},"output":{"data":{"message":"Hello GripMock"}}}' 127.0.0.1:4771/api/stubs
```

The stub has been successfully added, you have received a stub ID:
```bash
["6c85b0fa-caaf-4640-a672-f56b7dd8074d"]
```

You can check the added stubs at the link: http://127.0.0.1:4771/api/stubs.
The result will not make you wait long, you should see the following:
```json
[
  {
    "id": "6c85b0fa-caaf-4640-a672-f56b7dd8074d",
    "service": "Gripmock",
    "method": "SayHello",
    "headers": {
      "equals": null,
      "contains": null,
      "matches": null
    },
    "input": {
      "equals": {
        "name": "gripmock"
      },
      "contains": null,
      "matches": null
    },
    "output": {
      "data": {
        "message": "Hello GripMock"
      },
      "error": ""
    }
  }
]
```

Now try to use the grpc client to our service with the data from the input.

Happened? Well done. You are a fast learner.

It worked! 