package app

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
	"github.com/bavix/gripmock/v3/internal/infra/types"
)

// newTemplateData builds the render context shared by every mock handler.
// RequestTime and Timestamp always mirror; StubID and RequestID always mirror.
func newTemplateData(
	request map[string]any,
	headers map[string]any,
	messageIndex int,
	requestTime time.Time,
	requests []any,
	stubID string,
) template.Data {
	return template.Data{
		Request:      request,
		Headers:      headers,
		MessageIndex: messageIndex,
		RequestTime:  requestTime,
		Timestamp:    requestTime,
		State:        make(map[string]any),
		Requests:     requests,
		StubID:       stubID,
		RequestID:    stubID,
	}
}

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
