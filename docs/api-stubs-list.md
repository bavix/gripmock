## Rest API. Stubs List

Stubs List — endpoint returns a list of all registered stub files. It can be helpful to debbug your integration tests.

Enough to knock on the handle `GET /api/stubs`:
```bash
curl http://127.0.0.1:4771/api/stubs
[
  {
    "id": "6c85b0fa-caaf-4640-a672-f56b7dd8074d",
    "service": "Gripmock",
    "method": "SayHello",
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

It worked! 