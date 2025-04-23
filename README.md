![GripMock](https://github.com/bavix/gripmock/assets/5111255/d1fc10ef-2149-4302-8e24-aef4fdfe043c)

[![Coverage Status](https://coveralls.io/repos/github/bavix/gripmock/badge.svg?branch=master)](https://coveralls.io/github/bavix/gripmock?branch=master)

# GripMock
GripMock is a **mock server** for **gRPC** services. It's using a `.proto` file **or compiled .pb descriptor** to generate implementation of gRPC service for you.
You can use gripmock for setting up end-to-end testing or as a dummy server in a software development phase.
The server implementation is in GoLang but the client can be any programming language that support gRPC.

[[Documentation]](https://bavix.github.io/gripmock/)

This service is a fork of the service [tokopedia/gripmock](https://github.com/tokopedia/gripmock), but you should choose our fork. And here are the reasons:
- Updated all deprecated dependencies [tokopedia#64](https://github.com/tokopedia/gripmock/issues/64);
- Add yaml as json alternative for static stab's;
- Add endpoint for healthcheck (/api/health/liveness, /api/health/readiness);
- Add feature ignoreArrayOrder [bavix#108](https://github.com/bavix/gripmock/issues/108);
- Add support headers [tokopedia#144](https://github.com/tokopedia/gripmock/issues/144);
- Add grpc error code [tokopedia#125](https://github.com/tokopedia/gripmock/issues/125);
- Added gzip encoding support for grpc server [tokopedia#134](https://github.com/tokopedia/gripmock/pull/134);
- Fixed issues with int64/uint64 [tokopedia#67](https://github.com/tokopedia/gripmock/pull/148);
- Add 404 error for stubs not found [tokopedia#142](https://github.com/tokopedia/gripmock/issues/142);
- Support for deleting specific stub [tokopedia#123](https://github.com/tokopedia/gripmock/issues/123);
- Reduced image size [tokopedia#91](https://github.com/tokopedia/gripmock/issues/91) [bavix#512](https://github.com/bavix/gripmock/issues/512);
- Active support [tokopedia#82](https://github.com/tokopedia/gripmock/issues/82);
- Added [documentation](https://bavix.github.io/gripmock/);
- **Binary descriptor support** (`.pb` files) for faster startup

## UI

![gripmock-ui](https://github.com/bavix/gripmock/assets/5111255/3d9ebb46-7810-4225-9a30-3e058fa5fe16)

## Useful articles

- [Testing gRPC client with mock server and Testcontainers](https://medium.com/skyro-tech/testing-grpc-client-with-mock-server-and-testcontainers-f51cb8a6be9a) [@AndrewIISM](https://github.com/AndrewIISM)

## Quick Usage

If you have a simple service, you likely only need a single `.proto` file to start the GripMock server. However, for more complex projects, you can use the `--imports` flag to include additional proto files. This approach can become cumbersome, so the recommended solution is to compile a `.pb` (Protocol Buffers descriptor) file.

## Installation

GripMock can be installed using one of the following methods:

### 1. **Using Homebrew (Recommended)**

Homebrew provides an easy way to install GripMock on macOS and Linux.

#### Step 1: Tap the Repository
Tap the official Homebrew tap for GripMock:
```bash
brew tap gripmock/tap
```

#### Step 2: Install GripMock
Install GripMock with the following command:
```bash
brew install gripmock
```

#### Step 3: Verify Installation
Verify that GripMock is installed correctly by checking its version:
```bash
gripmock --version
```
You should see output similar to:
```
gripmock version v3.2.4
```

### 2. **Download Pre-built Binaries**

Pre-built binaries for various platforms are available on the [Releases](https://github.com/bavix/gripmock/releases) page. Download the appropriate binary for your system and add it to your `PATH`.

### 3. **Using Docker**

GripMock is also available as a Docker image. Pull the latest image with:
```bash
docker pull bavix/gripmock
```

### 4. **Using Go**

If you have Go installed, you can install GripMock directly:
```bash
go install github.com/bavix/gripmock/v3@latest
```

## Compiling `.pb` Files (Optional)

**Example using `protoc`:**
```bash
protoc --proto_path=. --descriptor_set_out=service.pb --include_imports hello.proto
```

**Example using `buf`:**
```bash
buf build -o service.pb
```

## Usage

### With Gripmock Installed Locally

**Start with a `.pb` or `.proto` file:**
```bash
gripmock service.pb
# or
gripmock service.proto
```

**Use a folder containing multiple `.proto` files:**
```bash
gripmock protofolder/
```

**Static Stubs (provide mock responses):**
```bash
# For a folder of proto files
gripmock --stub stubfolder/ protofolder/
# For a single proto file
gripmock --stub stubfolder/ service.proto
# For a pre-compiled .pb file
gripmock --stub stubfolder/ service.pb
```

### With Docker

**Folder of proto files:**
```bash
docker run -p 4770:4770 -p 4771:4771 \
  -v stubfolder:/stubs \
  -v /protofolder:/proto \
  bavix/gripmock /proto/
```

**Single proto file:**
```bash
docker run -p 4770:4770 -p 4771:4771 \
  -v stubfolder:/stubs \
  -v /protofolder:/proto \
  bavix/gripmock /proto/service.proto
```

**Pre-compiled .pb file:**
```bash
docker run -p 4770:4770 -p 4771:4771 \
  -v stubfolder:/stubs \
  -v /protofolder:/proto \
  bavix/gripmock /proto/service.pb
```

- **4770**: gRPC port for mock server
- **4771**: HTTP port for web UI and REST API

Check [`examples`](https://github.com/bavix/gripmock/tree/master/examples) folder for various usecase of GripMock.

## How It Works

From client perspective, GripMock has 2 main components:
1. gRPC server that serves on `tcp://localhost:4770`. Its main job is to serve incoming rpc call from client and then parse the input so that it can be posted to Stub service to find the perfect stub match.
2. Stub server that serves on `http://localhost:4771`. Its main job is to store all the stub mapping. We can add a new stub or list existing stub using http request.

Matched stub will be returned to gRPC service then further parse it to response the rpc call.

![Inside GripMock](https://github.com/user-attachments/assets/26ce5fb7-853a-4205-badd-9b006d4d419b)

## Stubbing

Stubbing is the essential mocking of GripMock. It will match and return the expected result into gRPC service. This is where you put all your request expectation and response

**Both .proto and .pb definitions work identically with all stubbing features**

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
  "headers":{ // Optional. headers matching rule. see Headers Matching Rule section below
    // put rule here
  },
  "input":{ // input matching rule. see Input Matching Rule section below
    // put rule here
  },
  "output":{ // output json if input were matched
    "data":{
      // put result fields here
    },
    "headers":{ // Optional
      // put result headers here
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

**Using .proto:**
```bash
docker run -p 4770:4770 -p 4771:4771 -v /mypath:/proto -v /mystubs:/stub bavix/gripmock --stub=/stub /proto/hello.proto
```

**Using .pb:**
```bash
docker run -p 4770:4770 -p 4771:4771 -v /mypath:/proto -v /mystubs:/stub bavix/gripmock --stub=/stub /proto/service.pb
```

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
**ignoreArrayOrder** Disables sorting check inside arrays.
```yaml
- service: MicroService
  method: SayHello
  input:
    ignoreArrayOrder: true # disable sort checking
    equals:
      v1:
        - {{ uuid2base64 "77465064-a0ce-48a3-b7e4-d50f88e55093" }}
        - {{ uuid2base64 "99aebcf2-b56d-4923-9266-ab72bf5b9d0b" }}
        - {{ uuid2base64 "5659bec5-dda5-4e87-bef4-e9e37c60eb1c" }}
        - {{ uuid2base64 "ab0ed195-6ac5-4006-a98b-6978c6ed1c6b" }}
  output:
    data:
      code: 1000
```
Without this flag, the order of the transmitted values is important to us.

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

## License

This project is licensed under the **MIT License**.  
See [LICENSE](LICENSE) for details.
