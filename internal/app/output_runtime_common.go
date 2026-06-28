package app

import (
	"context"
	"fmt"
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
		// Per-event delay was specified but has an unsupported type.
		// Treat it as an explicit configuration error rather than
		// silently falling back to the uniform output.delay.
		return 0, true, fmt.Errorf("unsupported delay type %T for value %v", delayVal, delayVal) //nolint:err113
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
