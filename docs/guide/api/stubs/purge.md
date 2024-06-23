# Stub API. Stubs Purge

Stubs Purge — endpoint removes all added stubs.

> In the future there will be a flag to remove non-static stubs.

Enough to knock on the handle `DELETE /api/stubs`:
```bash
curl -X DELETE http://127.0.0.1:4771/api/stubs
```

The endpoint will respond with code 204, everything is deleted.
