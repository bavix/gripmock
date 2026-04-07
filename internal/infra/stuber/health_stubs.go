package stuber

import (
	"github.com/google/uuid"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
)

const (
	// InternalStubIDGripmockHealthCheck is the reserved ID for internal gripmock health Check stub.
	InternalStubIDGripmockHealthCheck = "ffffffff-ffff-ffff-ffff-ffffffffffff"

	// InternalStubIDGripmockHealthWatch is the reserved ID for internal gripmock health Watch stub.
	InternalStubIDGripmockHealthWatch = "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"
)

// SetupGripmockHealthStubs creates internal stubs for gripmock health service.
// These stubs are hidden from user APIs and take precedence in matching.
// The stubs are initially set to NOT_SERVING and should be updated via UpdateGripmockHealthStatus
// when the server becomes ready.
func SetupGripmockHealthStubs(storage InternalStubStorage) {
	if storage == nil {
		return
	}

	storage.PutInternal(newGripmockHealthCheckStub(healthgrpc.HealthCheckResponse_NOT_SERVING.String()))
	storage.PutInternal(newGripmockHealthWatchStub(healthgrpc.HealthCheckResponse_NOT_SERVING.String()))
}

// UpdateGripmockHealthStatus updates the status by re-inserting stubs.
// This ensures thread-safe update via storage's internal locking.
func UpdateGripmockHealthStatus(storage InternalStubStorage, status healthgrpc.HealthCheckResponse_ServingStatus) {
	if storage == nil {
		return
	}

	statusStr := status.String()
	storage.PutInternal(newGripmockHealthCheckStub(statusStr))
	storage.PutInternal(newGripmockHealthWatchStub(statusStr))
}

func newGripmockHealthCheckStub(status string) *Stub {
	return &Stub{
		ID:      uuid.MustParse(InternalStubIDGripmockHealthCheck),
		Service: "grpc.health.v1.Health",
		Method:  "Check",
		Input: InputData{
			Equals: map[string]any{"service": "gripmock"},
		},
		Output: Output{
			Data: map[string]any{"status": status},
		},
	}
}

func newGripmockHealthWatchStub(status string) *Stub {
	return &Stub{
		ID:      uuid.MustParse(InternalStubIDGripmockHealthWatch),
		Service: "grpc.health.v1.Health",
		Method:  "Watch",
		Input: InputData{
			Equals: map[string]any{"service": "gripmock"},
		},
		Output: Output{
			Stream: []any{map[string]any{"status": status}},
		},
	}
}
