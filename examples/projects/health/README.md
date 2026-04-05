# Health Service Mocking Example

- Service: `grpc.health.v1.Health`
- Methods: `Check`, `Watch`

This project demonstrates health mocking for custom service names while keeping the internal `gripmock` key protected.

Notes:
- `service: "gripmock"` is reserved and always returns real server health.
- Custom names (for example `examples.health.backend`) can be mocked using regular stubs.
- Even if a stub for `service: "gripmock"` exists, it is ignored at runtime.
- No local `service.proto` is required because GripMock registers the standard gRPC health service.

Canonical proto reference:
- https://github.com/grpc/grpc-proto/blob/master/grpc/health/v1/health.proto

Run:
- Server: `go run main.go -s examples examples`
- Tests:
  - `grpctestify examples/projects/health/case_check_mocked_not_serving.gctf`
  - `grpctestify examples/projects/health/case_check_gripmock_protected.gctf`
  - `grpctestify examples/projects/health/case_check_unknown_service.gctf`
  - `grpctestify examples/projects/health/case_watch_mocked_stream.gctf`
  - `grpctestify examples/projects/health/case_watch_timing_cumulative.gctf`
