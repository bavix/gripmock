package app

import (
	"context"
	"net/http"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/types"
)

func outputStatusBase(output stuber.Output) *status.Status {
	if output.Error == "" && output.Code == nil {
		return nil
	}

	if output.Code != nil && *output.Code == codes.OK {
		return nil
	}

	code := codes.Aborted
	if output.Code != nil {
		code = *output.Code
	}

	return status.New(code, output.Error)
}

func delayResponse(ctx context.Context, delayDur types.Duration) error {
	if delayDur == 0 {
		return nil
	}

	if err := ctx.Err(); err != nil {
		return status.FromContextError(err).Err()
	}

	timer := time.NewTimer(time.Duration(delayDur))
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return status.FromContextError(ctx.Err()).Err()
	case <-timer.C:
		if err := ctx.Err(); err != nil {
			return status.FromContextError(ctx.Err()).Err()
		}

		return nil
	}
}

func extractStreamDelay(item any) (types.Duration, bool, error) {
	itemMap, ok := item.(map[string]any)
	if !ok {
		return 0, false, nil
	}

	delayVal, hasDelay := itemMap["delay"]
	if !hasDelay || delayVal == nil {
		return 0, false, nil
	}

	switch v := delayVal.(type) {
	case float64:
		return types.Duration(time.Duration(v)), true, nil
	case int64:
		return types.Duration(time.Duration(v)), true, nil
	case int:
		return types.Duration(time.Duration(v)), true, nil
	case string:
		d, err := time.ParseDuration(v)
		if err != nil {
			return 0, true, err
		}

		return types.Duration(d), true, nil
	default:
		return 0, false, nil
	}
}

func extractStreamData(item any) (map[string]any, bool) {
	itemMap, ok := item.(map[string]any)
	if !ok {
		return nil, false
	}

	if data, ok := itemMap["data"].(map[string]any); ok {
		return data, true
	}

	return itemMap, true
}

//nolint:cyclop,exhaustive
func ErrorCodeToString(code codes.Code) string {
	switch code {
	case codes.Canceled:
		return "canceled"
	case codes.Unknown:
		return "unknown"
	case codes.InvalidArgument:
		return "invalid_argument"
	case codes.DeadlineExceeded:
		return "deadline_exceeded"
	case codes.NotFound:
		return "not_found"
	case codes.AlreadyExists:
		return "already_exists"
	case codes.PermissionDenied:
		return "permission_denied"
	case codes.ResourceExhausted:
		return "resource_exhausted"
	case codes.FailedPrecondition:
		return "failed_precondition"
	case codes.Aborted:
		return "aborted"
	case codes.OutOfRange:
		return "out_of_range"
	case codes.Unimplemented:
		return "unimplemented"
	case codes.Internal:
		return "internal"
	case codes.Unavailable:
		return "unavailable"
	case codes.DataLoss:
		return "data_loss"
	case codes.Unauthenticated:
		return "unauthenticated"
	default:
		return "internal"
	}
}

//nolint:cyclop,exhaustive
func ErrorCodeToHTTPStatus(code codes.Code) int {
	switch code {
	case codes.Canceled:
		return http.StatusRequestTimeout
	case codes.Unknown:
		return http.StatusInternalServerError
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.FailedPrecondition:
		return http.StatusBadRequest
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}
