![GripMock](https://github.com/bavix/gripmock/assets/5111255/90119438-92e5-4479-bfc8-5cf510249b7f)

# GripMock
GripMock is a **mock server** for **gRPC** services. It's using a `.proto` file to generate implementation of gRPC service for you.
You can use gripmock for setting up end-to-end testing or as a dummy server in a software development phase.
The server implementation is in GoLang but the client can be any programming language that support gRPC.

[[Documentation]](https://bavix.github.io/gripmock/)

This service is a fork of the service [tokopedia/gripmock](https://github.com/tokopedia/gripmock), but you should choose our fork. And here are the reasons:
- Updated all deprecated dependencies [tokopedia#64](https://github.com/tokopedia/gripmock/issues/64);
- Add yaml as json alternative for static stab's;
- Add endpoint for healthcheck (/api/health/liveness, /api/health/readiness);
- Add grpc error code [tokopedia#125](https://github.com/tokopedia/gripmock/issues/125);
- Added gzip encoding support for grpc server [tokopedia#134](https://github.com/tokopedia/gripmock/pull/134);
- Fixed issues with int64/uint64 [tokopedia#67](https://github.com/tokopedia/gripmock/pull/148);
- Add 404 error for stubs not found [tokopedia#142](https://github.com/tokopedia/gripmock/issues/142);
- Support for deleting specific stub [tokopedia#123](https://github.com/tokopedia/gripmock/issues/123);
- Reduced image size [tokopedia#91](https://github.com/tokopedia/gripmock/issues/91);
- Active support [tokopedia#82](https://github.com/tokopedia/gripmock/issues/82);
- Added [documentation](https://bavix.github.io/gripmock/);

---

## Quick Usage
First, prepare your `.proto` file. Or you can use `hello.proto` in `example/simple/` folder. Suppose you put it in `/mypath/hello.proto`. We are gonna use Docker image for easier example test.
basic syntax to run GripMock is
`gripmock <protofile>`

- Install [Docker](https://docs.docker.com/install/)
- Run `docker pull bavix/gripmock` to pull the image
- We are gonna mount `/mypath/hello.proto` (it must be a fullpath) into a container and also we expose ports needed. Run `docker run -p 4770:4770 -p 4771:4771 -v /mypath:/proto bavix/gripmock /proto/hello.proto`
- On a separate terminal we are gonna add a stub into the stub service. Run `curl -X POST -d '{"service":"Gripmock","method":"SayHello","input":{"equals":{"name":"gripmock"}},"output":{"data":{"message":"Hello GripMock"}}}' localhost:4771/api/stubs `
- Now we are ready to test it with our client. You can find a client example file under `example/simple/client/`. Execute one of your preferred language. Example for go: `go run example/simple/client/*.go`

Check [`example`](https://github.com/bavix/gripmock/tree/master/example) folder for various usecase of gripmock.

---

## How It Works
![Running Gripmock](https://github.com/bavix/gripmock/assets/5111255/e8f74280-52b1-41bd-8582-4026904d934c)

From client perspective, GripMock has 2 main components:
1. gRPC server that serves on `tcp://localhost:4770`. Its main job is to serve incoming rpc call from client and then parse the input so that it can be posted to Stub service to find the perfect stub match.
2. Stub server that serves on `http://localhost:4771`. Its main job is to store all the stub mapping. We can add a new stub or list existing stub using http request.

Matched stub will be returned to gRPC service then further parse it to response the rpc call.


From technical perspective, GripMock consists of 2 binaries. 
The first binary is the gripmock itself, when it will generate the gRPC server using the plugin installed in the system (see [Dockerfile](Dockerfile)). 
When the server sucessfully generated, it will be invoked in parallel with stub server which ends up opening 2 ports for client to use.

The second binary is the protoc plugin which located in folder [protoc-gen-gripmock](/protoc-gen-gripmock). This plugin is the one who translates protobuf declaration into a gRPC server in Go programming language. 

![Inside GripMock](https://github.com/bavix/gripmock/assets/5111255/b0f27a47-6d2f-40ed-96d4-81b1151b747d)

---

## Stubbing

Stubbing is the essential mocking of GripMock. It will match and return the expected result into gRPC service. This is where you put all your request expectation and response

### Dynamic stubbing
You could add stubbing on the fly with a simple REST API. HTTP stub server is running on port `:4771`

- `GET /api/stubs` Will list all stubs mapping.
- `POST /api/stubs` Will add stub with provided stub data
- `POST /api/stubs/search` Find matching stub with provided input. see [Input Matching](#input_matching) below.
- `DELETE /api/stubs` Clear stub mappings.

Stub Format is JSON text format. It has a skeleton as follows:
```json
{
  "service":"<servicename>", // name of service defined in proto
  "method":"<methodname>", // name of method that we want to mock
  "input":{ // input matching rule. see Input Matching Rule section below
    // put rule here
  },
  "output":{ // output json if input were matched
    "data":{
      // put result fields here
    },
    "error":"<error message>", // Optional. if you want to return error instead.
    "code":"<response code>" // Optional. Grpc response code. if code !=0  return error instead.
  }
}
```

For our `hello` service example we put a stub with the text below:
```json
  {
    "service":"Greeter",
    "method":"SayHello",
    "input":{
      "equals":{
        "name":"gripmock"
      }
    },
    "output":{
      "data":{
        "message":"Hello GripMock"
      }
    }
  }
```

### Static stubbing
You could initialize gripmock with stub json files and provide the path using `--stub` argument. For example you may
mount your stub file in `/mystubs` folder then mount it to docker like

`docker run -p 4770:4770 -p 4771:4771 -v /mypath:/proto -v /mystubs:/stub bavix/gripmock --stub=/stub /proto/hello.proto`

Please note that Gripmock still serves http stubbing to modify stored stubs on the fly.

## <a name="input_matching"></a>Input Matching
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

---
Supported by

[![Supported by JetBrains](https://cdn.rawgit.com/bavix/development-through/46475b4b/jetbrains.svg)](https://www.jetbrains.com/)
