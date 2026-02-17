# Descriptor API (`/api/descriptors`) <VersionTag version="v3.7.0" />

Runtime loading of protobuf descriptors (`google.protobuf.FileDescriptorSet`) without restarting GripMock.

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

- `message`: fixed `ok` for successful load
- `serviceIDs`: fully-qualified service names extracted from descriptor
- `time`: server timestamp (dynamic)

Common error (`400 Bad Request`):

```json
{"error":"invalid FileDescriptorSet: proto: cannot parse invalid wire-format data"}
```

This means the uploaded file is not a valid `FileDescriptorSet`.

## Validated workflow (unitconverter)

The sequence below was re-run against release binary (`gripmock` via Homebrew v3.7.0).

### 1) Start server

```bash
gripmock
```

Local source equivalent:

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

Expected: `message=ok`, `serviceIDs` includes `unitconverter.v1.UnitConversionService`.

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

`/api/descriptors` accepts only descriptor sets, not arbitrary protobuf payloads.

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

- Use `--data-binary`, not `-d`, for binary uploads.
- If `localhost` is not reachable in your environment, use `127.0.0.1`.
- If proxy env vars interfere with local calls, use:

```bash
curl --noproxy '*' -X POST http://localhost:4771/api/descriptors \
  -H "Content-Type: application/octet-stream" \
  --data-binary "@service.pb"
```

## Troubleshooting

`invalid FileDescriptorSet`:

- rebuild descriptor as `FileDescriptorSet`
- for `protoc`, keep `--include_imports` when imports exist

`Test path does not exist` in `grpctestify`:

- run from `examples/projects/unitconverter`

Service not found during tests:

- load descriptors first, then stubs, then run tests

## Related endpoints

- `POST /api/stubs`
- `GET /api/stubs`
- `DELETE /api/stubs`
