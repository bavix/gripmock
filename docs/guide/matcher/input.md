# Input Matching Rule

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

Input matching has 3 rules to match an input: **equals**,**contains** and **matches**
<br>
Nested fields are allowed for input matching too for all JSON data types. (`string`, `bool`, `array`, etc.)
<br>
**Gripmock** recursively goes over the fields and tries to match with given input.
<br>

## Input Equals

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

## Input Contains

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

The **contains** rule allows you to specify that the input must contain certain values. This is useful for cases where you want to ensure that a specific field is present in the input, regardless of its value.

For example, if you have a service that takes a JSON object as input and you want to ensure that the object contains a field called "name", you can use the **contains** rule like this:

Example 1:
```json
{
  "input": {
    "contains": {
      "name": "anyValue"
    }
  }
}
```

Example 2:
```json
{
  "input": {
    "contains": {
      "address": {
        "city": "anyCity"
      }
    }
  }
}
```

These examples demonstrate how to use the **contains** rule to check for the presence of specific fields without caring about their actual values.

## Input Matches

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

To implement the functionality for using `github.com/gripmock/deeply` for matching, we need to create an algorithm that leverages the library's capabilities to perform deep matching of JSON objects using regular expressions. The `deeply` package is designed to traverse complex data structures and apply specified matching conditions.

### Algorithm Description

1. **Input Parsing**: 
   - Accept the JSON input data and the matching criteria. The criteria will include fields with regex patterns for matching.

2. **Deep Traversal**:
   - Utilize the `deeply` library to recursively traverse the input data structure.
   - At each node, check if the current path matches any of the specified patterns in the criteria.

3. **Regex Matching**:
   - For fields specified in the `matches` criteria, compile the regex patterns.
   - Apply the compiled regex to the corresponding fields in the input data.
   - If a field matches the regex, mark it as a successful match.

4. **Result Compilation**:
   - Gather all matching results and determine if the overall input satisfies the criteria.
   - Return a boolean indicating the success of the match and details of any matched fields.

### Example

**Criteria**:
```json
{
  "matches": {
    "name": "^grip.*$",
    "cities": ["Jakarta", "Istanbul", ".*grad$"]
  }
}
```

**Input Data**:
```json
{
  "name": "gripmock",
  "cities": ["Jakarta", "Belgrade"]
}
```

**Matching Process**:
- The `name` field is checked against the regex `^grip.*$` and matches successfully with "gripmock".
- The `cities` array is checked:
  - "Jakarta" matches with the exact string "Jakarta".
  - "Belgrade" matches the pattern ".*grad$".

The algorithm concludes with a successful match, indicating the input data meets the specified criteria using `github.com/gripmock/deeply`.

## Input Flag ignoreArrayOrder

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

The ignoreArrayOrder flag in the input of the stub works as follows:

1. The input field is parsed as a JSON object.
2. The JSON object is traversed depth-first, and all arrays are extracted.
3. Each array is sorted in ascending order by the value of the elements.
4. The sorted arrays are then compared element-wise with the expected array.
5. If the arrays are equal, the stub is considered a match.

This flag allows you to disable the sorting check, which can be useful when the order of the transmitted values doesn't matter to you.
