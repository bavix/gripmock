# Descriptor API (`/api/descriptors`) <VersionTag version="v3.7.0" />

Load protobuf descriptors (`google.protobuf.FileDescriptorSet`) into a running GripMock instance without restart.

## Endpoint contract

- Method: `POST`
- Path: `/api/descriptors`
- Header: `Content-Type: application/octet-stream`
- Body: binary descriptor set (`.pb`)

Request example:

```bash
curl -X POST http://localhost:4771/api/descriptors \
  -H "Content-Type: application/octet-stream" \
  --data-binary "@service.pb"
```

Success response (`200 OK`):

```json
{
  "message": "ok",
  "serviceIDs": ["unitconverter.v1.UnitConversionService"],
  "time": "2026-02-17T22:33:17.162507+03:00"
}
```

Field notes:

- `message`: `ok` when descriptor loading succeeds
- `serviceIDs`: fully-qualified service names found in the descriptor
- `time`: server timestamp (dynamic)

Common error (`400 Bad Request`):

```json
{"error":"invalid FileDescriptorSet: proto: cannot parse invalid wire-format data"}
```

This usually means the uploaded file is not a valid `FileDescriptorSet`.

## Validated workflow (unitconverter)

This flow was re-checked on the Homebrew release binary (`gripmock` v3.7.0).

### 1) Start server

```bash
gripmock
```

If you run from this repository source:

```bash
go run main.go
```

### 2) Move to example project

```bash
cd examples/projects/unitconverter
```

### 3) Load descriptor

```bash
curl -X POST http://localhost:4771/api/descriptors \
  -H "Content-Type: application/octet-stream" \
  --data-binary "@service.pb"
```

Expected: `message=ok`, and `serviceIDs` includes `unitconverter.v1.UnitConversionService`.

### 4) Load stub

```bash
curl -X POST http://localhost:4771/api/stubs \
  -H "Content-Type: application/json" \
  --data-binary "@convert_weight/stub_single.json"
```

Expected: JSON array with one UUID (stub ID).

### 5) Execute test

```bash
grpctestify convert_weight/case_success.gctf
```

Expected: `1 passed`.

## Building `service.pb`

`/api/descriptors` accepts descriptor sets only.

With `protoc`:

```bash
protoc \
  -I . \
  --include_imports \
  --descriptor_set_out=service.pb \
  service.proto
```

With `buf`:

```bash
buf build -o service.pb
```

## Transport details

- Use `--data-binary`, not `-d`, for binary uploads
- If `localhost` is not reachable in your environment, use `127.0.0.1`
- If HTTP proxy variables interfere with local calls, use:

```bash
curl --noproxy '*' -X POST http://localhost:4771/api/descriptors \
  -H "Content-Type: application/octet-stream" \
  --data-binary "@service.pb"
```

## Troubleshooting

`invalid FileDescriptorSet`:

- rebuild descriptor as `FileDescriptorSet`
- with `protoc`, keep `--include_imports` when imports exist

`Test path does not exist` in `grpctestify`:

- run from `examples/projects/unitconverter`

Service not found during tests:

- load descriptors first, then stubs, then run tests

Quick sanity check for admin API reachability:

```bash
curl -s http://localhost:4771/api/stubs
```

## Related endpoints

- [`POST /api/stubs` (upsert)](/guide/api/stubs/upsert)
- [`GET /api/stubs` (list)](/guide/api/stubs/list)
- [`DELETE /api/stubs` (purge)](/guide/api/stubs/purge)
