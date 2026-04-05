package app

import (
	stderrors "errors"
	"fmt"
	"math"
	"strings"

	"google.golang.org/grpc/codes"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protodesc"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

var (
	errUnsupportedHealthStatus     = stderrors.New("unsupported health status")
	errUnsupportedHealthStatusType = stderrors.New("unsupported health status type")
)

func healthResponsesFromOutput(output stuber.Output) ([]*healthgrpc.HealthCheckResponse, error) {
	if len(output.Stream) > 0 {
		responses := make([]*healthgrpc.HealthCheckResponse, 0, len(output.Stream))

		for i, item := range output.Stream {
			itemMap, ok := item.(map[string]any)
			if !ok {
				return nil, status.Errorf(codes.Internal, "health watch stream[%d] has invalid type %T", i, item)
			}

			healthStatus, err := healthStatusFromMap(itemMap)
			if err != nil {
				return nil, err
			}

			responses = append(responses, &healthgrpc.HealthCheckResponse{Status: healthStatus})
		}

		return responses, nil
	}

	healthStatus, err := healthStatusFromMap(output.Data)
	if err != nil {
		return nil, err
	}

	return []*healthgrpc.HealthCheckResponse{{Status: healthStatus}}, nil
}

func healthStatusFromMap(data map[string]any) (healthgrpc.HealthCheckResponse_ServingStatus, error) {
	if data == nil {
		return 0, status.Error(codes.Internal, "health stub output.data is required")
	}

	rawStatus, ok := data["status"]
	if !ok {
		return 0, status.Error(codes.Internal, "health stub output.data.status is required")
	}

	healthStatus, err := parseHealthStatus(rawStatus)
	if err != nil {
		return 0, status.Error(codes.Internal, err.Error())
	}

	return healthStatus, nil
}

func parseHealthStatus(value any) (healthgrpc.HealthCheckResponse_ServingStatus, error) {
	switch v := value.(type) {
	case healthgrpc.HealthCheckResponse_ServingStatus:
		return v, nil
	case int:
		return parseIntHealthStatus(int64(v))
	case int32:
		return parseIntHealthStatus(int64(v))
	case int64:
		return parseIntHealthStatus(v)
	case float64:
		return parseFloatHealthStatus(v)
	case string:
		return parseStringHealthStatus(v)
	default:
		return 0, fmt.Errorf("%w: %T", errUnsupportedHealthStatusType, value)
	}
}

func parseIntHealthStatus(value int64) (healthgrpc.HealthCheckResponse_ServingStatus, error) {
	if value < math.MinInt32 || value > math.MaxInt32 {
		return 0, fmt.Errorf("%w: %d", errUnsupportedHealthStatus, value)
	}

	return healthgrpc.HealthCheckResponse_ServingStatus(int32(value)), nil
}

func parseFloatHealthStatus(value float64) (healthgrpc.HealthCheckResponse_ServingStatus, error) {
	if value != math.Trunc(value) {
		return 0, fmt.Errorf("%w: %v", errUnsupportedHealthStatus, value)
	}

	return parseIntHealthStatus(int64(value))
}

func parseStringHealthStatus(value string) (healthgrpc.HealthCheckResponse_ServingStatus, error) {
	normalized := strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(value)), "HEALTHCHECKRESPONSE_")

	statuses := map[string]healthgrpc.HealthCheckResponse_ServingStatus{
		"UNKNOWN":         healthgrpc.HealthCheckResponse_UNKNOWN,
		"SERVING":         healthgrpc.HealthCheckResponse_SERVING,
		"NOT_SERVING":     healthgrpc.HealthCheckResponse_NOT_SERVING,
		"SERVICE_UNKNOWN": healthgrpc.HealthCheckResponse_SERVICE_UNKNOWN,
	}

	statusValue, ok := statuses[normalized]
	if !ok {
		return 0, fmt.Errorf("%w: %q", errUnsupportedHealthStatus, value)
	}

	return statusValue, nil
}

func statusFromHealthOutput(output stuber.Output, resolver protodesc.Resolver) (*status.Status, error) {
	return statusFromOutputWithDetails(output, resolver)
}
