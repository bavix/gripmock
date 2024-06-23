# Headers Matching Rule

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

Headers matching has 3 rules to match an input: **equals**,**contains** and **matches**
<br>
Headers can consist of a key and a value. If there are several values, then you need to list them separated by ";". Data type string.
<br>
**Gripmock** recursively goes over the fields and tries to match with given input.
<br>

## Header Equals

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

## Header Contains

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

## Header Matches

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
