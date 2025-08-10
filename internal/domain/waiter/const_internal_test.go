package waiter

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
)

func TestServingStatus_Constants(t *testing.T) {
	t.Parallel()

	// Test that constants are properly defined
	require.Equal(t, Unknown, ServingStatus(healthv1.HealthCheckResponse_UNKNOWN))
	require.Equal(t, Serving, ServingStatus(healthv1.HealthCheckResponse_SERVING))
	require.Equal(t, NotServing, ServingStatus(healthv1.HealthCheckResponse_NOT_SERVING))
	require.Equal(t, ServiceUnknown, ServingStatus(healthv1.HealthCheckResponse_SERVICE_UNKNOWN))
}

func TestServingStatus_Values(t *testing.T) {
	t.Parallel()

	// Test that constants have expected values
	require.Equal(t, uint32(0), uint32(Unknown))
	require.Equal(t, uint32(1), uint32(Serving))
	require.Equal(t, uint32(2), uint32(NotServing))
	require.Equal(t, uint32(3), uint32(ServiceUnknown))
}

func TestServingStatus_TypeConversion(t *testing.T) {
	t.Parallel()

	// Test type conversion from protobuf enum to our type
	protoUnknown := healthv1.HealthCheckResponse_UNKNOWN
	protoServing := healthv1.HealthCheckResponse_SERVING
	protoNotServing := healthv1.HealthCheckResponse_NOT_SERVING
	protoServiceUnknown := healthv1.HealthCheckResponse_SERVICE_UNKNOWN

	require.Equal(t, Unknown, ServingStatus(protoUnknown))
	require.Equal(t, Serving, ServingStatus(protoServing))
	require.Equal(t, NotServing, ServingStatus(protoNotServing))
	require.Equal(t, ServiceUnknown, ServingStatus(protoServiceUnknown))
}

func TestServingStatus_Comparison(t *testing.T) {
	t.Parallel()

	// Test comparison operations
	require.Equal(t, Unknown, ServingStatus(healthv1.HealthCheckResponse_UNKNOWN))
	require.Equal(t, Serving, ServingStatus(healthv1.HealthCheckResponse_SERVING))
	require.Equal(t, NotServing, ServingStatus(healthv1.HealthCheckResponse_NOT_SERVING))
	require.Equal(t, ServiceUnknown, ServingStatus(healthv1.HealthCheckResponse_SERVICE_UNKNOWN))

	// Test that different statuses are not equal
	require.NotEqual(t, Unknown, Serving)
	require.NotEqual(t, Serving, NotServing)
	require.NotEqual(t, NotServing, ServiceUnknown)
}

func TestServingStatus_StringRepresentation(t *testing.T) {
	t.Parallel()

	// Test string representation through fmt.Sprintf
	unknownStr := fmt.Sprintf("%v", Unknown)
	servingStr := fmt.Sprintf("%v", Serving)
	notServingStr := fmt.Sprintf("%v", NotServing)
	serviceUnknownStr := fmt.Sprintf("%v", ServiceUnknown)

	// These should not be empty
	require.NotEmpty(t, unknownStr)
	require.NotEmpty(t, servingStr)
	require.NotEmpty(t, notServingStr)
	require.NotEmpty(t, serviceUnknownStr)

	// They should be different
	require.NotEqual(t, unknownStr, servingStr)
	require.NotEqual(t, servingStr, notServingStr)
	require.NotEqual(t, notServingStr, serviceUnknownStr)
}

func TestServingStatus_Arithmetic(t *testing.T) {
	t.Parallel()

	// Test arithmetic operations
	require.Equal(t, Unknown+1, ServingStatus(1))
	require.Equal(t, Serving+1, ServingStatus(2))
	require.Equal(t, NotServing+1, ServingStatus(3))
	require.Equal(t, ServiceUnknown+1, ServingStatus(4))

	// Test subtraction
	require.Equal(t, Serving-1, ServingStatus(0))
	require.Equal(t, NotServing-1, ServingStatus(1))
	require.Equal(t, ServiceUnknown-1, ServingStatus(2))
}

func TestServingStatus_BitwiseOperations(t *testing.T) {
	t.Parallel()

	// Test bitwise operations
	require.Equal(t, Unknown&Serving, ServingStatus(0))
	require.Equal(t, Serving, ServingStatus(1))        // Same as Serving & Serving
	require.Equal(t, NotServing, ServingStatus(2))     // Same as NotServing & NotServing
	require.Equal(t, ServiceUnknown, ServingStatus(3)) // Same as ServiceUnknown & ServiceUnknown

	// Test OR operations
	require.Equal(t, Unknown|Serving, ServingStatus(1))
	require.Equal(t, Serving|NotServing, ServingStatus(3))
	require.Equal(t, NotServing|ServiceUnknown, ServingStatus(3))
}

func TestServingStatus_Validation(t *testing.T) {
	t.Parallel()

	// Test that our constants are valid
	require.True(t, isValidServingStatus(Unknown))
	require.True(t, isValidServingStatus(Serving))
	require.True(t, isValidServingStatus(NotServing))
	require.True(t, isValidServingStatus(ServiceUnknown))

	// Test invalid statuses
	require.False(t, isValidServingStatus(ServingStatus(999)))
	require.False(t, isValidServingStatus(ServingStatus(255)))
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
	t.Parallel()

	// Test conversion back to protobuf enum
	protoUnknown := healthv1.HealthCheckResponse_ServingStatus(Unknown)
	protoServing := healthv1.HealthCheckResponse_ServingStatus(Serving)
	protoNotServing := healthv1.HealthCheckResponse_ServingStatus(NotServing)
	protoServiceUnknown := healthv1.HealthCheckResponse_ServingStatus(ServiceUnknown)

	require.Equal(t, healthv1.HealthCheckResponse_UNKNOWN, protoUnknown)
	require.Equal(t, healthv1.HealthCheckResponse_SERVING, protoServing)
	require.Equal(t, healthv1.HealthCheckResponse_NOT_SERVING, protoNotServing)
	require.Equal(t, healthv1.HealthCheckResponse_SERVICE_UNKNOWN, protoServiceUnknown)
}

func TestServingStatus_EdgeCases(t *testing.T) {
	t.Parallel()

	// Test edge cases
	maxStatus := ServingStatus(255)
	minStatus := ServingStatus(0)

	// Test that our constants are within reasonable bounds
	require.GreaterOrEqual(t, uint32(Unknown), uint32(minStatus))
	require.LessOrEqual(t, uint32(ServiceUnknown), uint32(maxStatus))

	// Test that we can handle values outside our defined range
	invalidStatus := ServingStatus(100)
	require.False(t, isValidServingStatus(invalidStatus))
}

func TestServingStatus_UsageInContext(t *testing.T) {
	t.Parallel()

	// Test usage in a more realistic context
	statuses := []ServingStatus{Unknown, Serving, NotServing, ServiceUnknown}

	for _, status := range statuses {
		// Test that we can use the status in a switch statement
		switch status {
		case Unknown:
			require.Equal(t, Unknown, status)
		case Serving:
			require.Equal(t, Serving, status)
		case NotServing:
			require.Equal(t, NotServing, status)
		case ServiceUnknown:
			require.Equal(t, ServiceUnknown, status)
		default:
			t.Errorf("Unexpected status: %v", status)
		}
	}
}
