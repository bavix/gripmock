## Static Stubs. JSON

With static stubs, you can use gripmock without the handle API. 
This is useful when you don't want to rely on the http protocol in your tests, or if your data is all static and doesn't change. 
It can also be useful when there are a lot of stubs.

So what do you need to work? It is enough to mount a folder with stubs in your container and tell the service the path to the stubs.

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

We have created a folder for `stubs` stubs.
Now you need to create the first stub in this folder `single.json`.

```json
{
  "service": "Gripmock",
  "method": "SayHello",
  "input": {
    "equals": {
      "name": "tokopedia-single"
    }
  },
  "output": {
    "data": {
      "message": "Hello Tokopedia",
      "return_code": 1
    }
  }
}
```

Let's create a second stub `multi-stabs.json`.

```json
[
  {
    "service": "Gripmock",
    "method": "SayHello",
    "input": {
      "equals": {
        "name": "tokopedia"
      }
    },
    "output": {
      "data": {
        "message": "Hello Tokopedia",
        "return_code": 1
      }
    }
  },
  {
    "service": "Gripmock",
    "method": "SayHello",
    "input": {
      "equals": {
        "name": "world"
      }
    },
    "output": {
      "data": {
        "message": "Hello World",
        "return_code": 1
      }
    }
  }
]
```

The launch looks something like this:
```bash
docker run \
  -p 4770:4770 \
  -p 4771:4771 \
  -v ./stubs:/stubs:ro \
  -v ./api/proto:/proto:ro \
  bavix/gripmock --stub=/stubs /proto/simple.proto
```

You can verify that the stubs have been loaded successfully by running a query:
```bash
curl http://127.0.0.1:4771/api/stubs
```

It worked! 
