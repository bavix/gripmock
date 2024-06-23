# Stub API. Stubs Delete

Stubs Delete — endpoint removes stub by ID.

Enough to knock on the handle `DELETE /api/stubs/{uuid}`:
```bash
curl -X DELETE http://127.0.0.1:4771/api/stubs/6c85b0fa-caaf-4640-a672-f56b7dd8074d
```

The endpoint will respond with code 204, the stub has been removed.
