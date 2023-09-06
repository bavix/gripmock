## Proto-files. One service, one contract

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

GripMock server supports [method reflection](https://github.com/grpc/grpc-go/blob/master/Documentation/server-reflection-tutorial.md). You can verify that all services have been created successfully by accessing the gripmock port.

It worked! 
