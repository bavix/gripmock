# Greeter (dynamic stub)

- Service: `helloworld.Greeter/SayHello`
- Dynamic response: `"Hello, {{.Request.name}}!"`

Original proto reference:
- gRPC Hello World proto: https://github.com/grpc/grpc-go/blob/master/examples/helloworld/helloworld/helloworld.proto

Run:
- Server: `go run main.go examples/projects/greeter/service.proto --stub examples/projects/greeter`
- Tests:
  - `grpctestify examples/projects/greeter/case_say_hello_alice.gctf`
  - `grpctestify examples/projects/greeter/case_say_hello_alex.gctf`
  - `grpctestify examples/projects/greeter/case_say_hello_bob.gctf`
