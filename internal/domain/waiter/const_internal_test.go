package waiter

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
)

func TestServingStatus_Constants(t *testing.T) {
	// Test that constants are properly defined
	assert.Equal(t, Unknown, ServingStatus(healthv1.HealthCheckResponse_UNKNOWN))
	assert.Equal(t, Serving, ServingStatus(healthv1.HealthCheckResponse_SERVING))
	assert.Equal(t, NotServing, ServingStatus(healthv1.HealthCheckResponse_NOT_SERVING))
	assert.Equal(t, ServiceUnknown, ServingStatus(healthv1.HealthCheckResponse_SERVICE_UNKNOWN))
}

func TestServingStatus_Values(t *testing.T) {
	// Test that constants have expected values
	assert.Equal(t, uint32(0), uint32(Unknown))
	assert.Equal(t, uint32(1), uint32(Serving))
	assert.Equal(t, uint32(2), uint32(NotServing))
	assert.Equal(t, uint32(3), uint32(ServiceUnknown))
}

func TestServingStatus_TypeConversion(t *testing.T) {
	// Test type conversion from protobuf enum to our type
	protoUnknown := healthv1.HealthCheckResponse_UNKNOWN
	protoServing := healthv1.HealthCheckResponse_SERVING
	protoNotServing := healthv1.HealthCheckResponse_NOT_SERVING
	protoServiceUnknown := healthv1.HealthCheckResponse_SERVICE_UNKNOWN

	assert.Equal(t, Unknown, ServingStatus(protoUnknown))
	assert.Equal(t, Serving, ServingStatus(protoServing))
	assert.Equal(t, NotServing, ServingStatus(protoNotServing))
	assert.Equal(t, ServiceUnknown, ServingStatus(protoServiceUnknown))
}

func TestServingStatus_Comparison(t *testing.T) {
	// Test comparison operations
	assert.Equal(t, Unknown, ServingStatus(healthv1.HealthCheckResponse_UNKNOWN))
	assert.Equal(t, Serving, ServingStatus(healthv1.HealthCheckResponse_SERVING))
	assert.Equal(t, NotServing, ServingStatus(healthv1.HealthCheckResponse_NOT_SERVING))
	assert.Equal(t, ServiceUnknown, ServingStatus(healthv1.HealthCheckResponse_SERVICE_UNKNOWN))

	// Test that different statuses are not equal
	assert.NotEqual(t, Unknown, Serving)
	assert.NotEqual(t, Serving, NotServing)
	assert.NotEqual(t, NotServing, ServiceUnknown)
}

func TestServingStatus_StringRepresentation(t *testing.T) {
	// Test string representation through fmt.Sprintf
	unknownStr := fmt.Sprintf("%v", Unknown)
	servingStr := fmt.Sprintf("%v", Serving)
	notServingStr := fmt.Sprintf("%v", NotServing)
	serviceUnknownStr := fmt.Sprintf("%v", ServiceUnknown)

	// These should not be empty
	assert.NotEmpty(t, unknownStr)
	assert.NotEmpty(t, servingStr)
	assert.NotEmpty(t, notServingStr)
	assert.NotEmpty(t, serviceUnknownStr)

	// They should be different
	assert.NotEqual(t, unknownStr, servingStr)
	assert.NotEqual(t, servingStr, notServingStr)
	assert.NotEqual(t, notServingStr, serviceUnknownStr)
}

func TestServingStatus_Arithmetic(t *testing.T) {
	// Test arithmetic operations
	assert.Equal(t, Unknown+1, ServingStatus(1))
	assert.Equal(t, Serving+1, ServingStatus(2))
	assert.Equal(t, NotServing+1, ServingStatus(3))
	assert.Equal(t, ServiceUnknown+1, ServingStatus(4))

	// Test subtraction
	assert.Equal(t, Serving-1, ServingStatus(0))
	assert.Equal(t, NotServing-1, ServingStatus(1))
	assert.Equal(t, ServiceUnknown-1, ServingStatus(2))
}

func TestServingStatus_BitwiseOperations(t *testing.T) {
	// Test bitwise operations
	assert.Equal(t, Unknown&Serving, ServingStatus(0))
	assert.Equal(t, Serving, ServingStatus(1))        // Same as Serving & Serving
	assert.Equal(t, NotServing, ServingStatus(2))     // Same as NotServing & NotServing
	assert.Equal(t, ServiceUnknown, ServingStatus(3)) // Same as ServiceUnknown & ServiceUnknown

	// Test OR operations
	assert.Equal(t, Unknown|Serving, ServingStatus(1))
	assert.Equal(t, Serving|NotServing, ServingStatus(3))
	assert.Equal(t, NotServing|ServiceUnknown, ServingStatus(3))
}

func TestServingStatus_Validation(t *testing.T) {
	// Test that our constants are valid
	assert.True(t, isValidServingStatus(Unknown))
	assert.True(t, isValidServingStatus(Serving))
	assert.True(t, isValidServingStatus(NotServing))
	assert.True(t, isValidServingStatus(ServiceUnknown))

	// Test invalid statuses
	assert.False(t, isValidServingStatus(ServingStatus(999)))
	assert.False(t, isValidServingStatus(ServingStatus(255)))
}

// Helper function to validate serving status.
func isValidServingStatus(status ServingStatus) bool {
	switch status {
	case Unknown, Serving, NotServing, ServiceUnknown:
		return true
	default:
		return false
	}
}

func TestServingStatus_ConversionToProto(t *testing.T) {
	// Test conversion back to protobuf enum
	protoUnknown := healthv1.HealthCheckResponse_ServingStatus(Unknown)
	protoServing := healthv1.HealthCheckResponse_ServingStatus(Serving)
	protoNotServing := healthv1.HealthCheckResponse_ServingStatus(NotServing)
	protoServiceUnknown := healthv1.HealthCheckResponse_ServingStatus(ServiceUnknown)

	assert.Equal(t, healthv1.HealthCheckResponse_UNKNOWN, protoUnknown)
	assert.Equal(t, healthv1.HealthCheckResponse_SERVING, protoServing)
	assert.Equal(t, healthv1.HealthCheckResponse_NOT_SERVING, protoNotServing)
	assert.Equal(t, healthv1.HealthCheckResponse_SERVICE_UNKNOWN, protoServiceUnknown)
}

func TestServingStatus_EdgeCases(t *testing.T) {
	// Test edge cases
	maxStatus := ServingStatus(255)
	minStatus := ServingStatus(0)

	// Test that our constants are within reasonable bounds
	assert.GreaterOrEqual(t, uint32(Unknown), uint32(minStatus))
	assert.LessOrEqual(t, uint32(ServiceUnknown), uint32(maxStatus))

	// Test that we can handle values outside our defined range
	invalidStatus := ServingStatus(100)
	assert.False(t, isValidServingStatus(invalidStatus))
}

func TestServingStatus_UsageInContext(t *testing.T) {
	// Test usage in a more realistic context
	statuses := []ServingStatus{Unknown, Serving, NotServing, ServiceUnknown}

	for _, status := range statuses {
		// Test that we can use the status in a switch statement
		switch status {
		case Unknown:
			assert.Equal(t, Unknown, status)
		case Serving:
			assert.Equal(t, Serving, status)
		case NotServing:
			assert.Equal(t, NotServing, status)
		case ServiceUnknown:
			assert.Equal(t, ServiceUnknown, status)
		default:
			t.Errorf("Unexpected status: %v", status)
		}
	}
}
