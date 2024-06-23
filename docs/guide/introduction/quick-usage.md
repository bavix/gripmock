# Quick Usage

## Installation

For ease of installation, the entire GripMock service is packaged into one dockerfile. You only need to install docker and get the image.

I will skip the details of installing docker and using it. Read documentation: https://docs.docker.com/engine/install/.

## Preparation

Let's imagine that we have a gRPC service that we want to mock.

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

## One service

GripMock service, at the moment, can only be run in a docker container.
All proto-files must be mounted in the docker and the path to them must be specified for the gripmock service.

The launch looks something like this:
```bash
docker run \
  -p 4770:4770 \
  -p 4771:4771 \
  -v ./api/proto:/proto:ro \
  bavix/gripmock /proto/simple.proto
```

We mounted the `api/proto` folder with our proto-files, there is a `simple.proto` file there.
We have created this service.

## Many services

GripMock service, at the moment, can only be run in a docker container.
All proto-files must be mounted in the docker and the path to them must be specified for the gripmock service.

The launch looks something like this:
```bash
docker run \
  -p 4770:4770 \
  -p 4771:4771 \
  -v ./api/proto:/proto:ro \
  bavix/gripmock /proto/proto1.proto /proto/proto2.proto ... /proto/protoN.proto
```

We mounted the api/proto folder with our protofiles, there were N-services there.
We have created this service.

## Mocking

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

GripMock server supports [method reflection](https://github.com/grpc/grpc-go/blob/master/Documentation/server-reflection-tutorial.md). You can verify that all services have been created successfully by accessing the gripmock port.

## Stubbing

This means that everything went well. Now let's add the first stub:
```bash
curl -X POST -d '{"service":"Gripmock","method":"SayHello","input":{"equals":{"name":"gripmock"}},"output":{"data":{"message":"Hello GripMock"}}}' 127.0.0.1:4771/api/stubs
```

The stub has been successfully added, you have received a stub ID:
```bash
["6c85b0fa-caaf-4640-a672-f56b7dd8074d"]
```

## Checking

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
