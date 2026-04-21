package app

import (
	"context"
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
			return status.FromContextError(err).Err()
		}

		return nil
	}
}
