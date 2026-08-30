package app

import (
	"context"
	"strings"
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
	stub *stuber.Stub,
	attemptNumber int,
) template.Data {
	var (
		stubID      string
		maxAttempts int
	)

	if stub != nil {
		stubID, maxAttempts = stub.ID.String(), stub.EffectiveTimes()
	}

	return template.Data{
		Request:       request,
		AttemptNumber: attemptNumber,
		AttemptIndex:  attemptNumber,
		MaxAttempts:   maxAttempts,
		TotalAttempts: maxAttempts,
		Headers:       headers,
		MessageIndex:  messageIndex,
		RequestTime:   requestTime,
		Timestamp:     requestTime,
		State:         make(map[string]any),
		Requests:      requests,
		StubID:        stubID,
		RequestID:     stubID,
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

func resolveDelay(engine *template.Engine, delay types.Delay, data template.Data) (types.Duration, error) {
	if !delay.IsTemplate() {
		return delay.Static(), nil
	}

	rendered, err := engine.Render(string(delay), data)
	if err != nil {
		return 0, status.Errorf(codes.Internal, "failed to process delay template: %v", err)
	}

	rendered = strings.TrimSpace(rendered)
	if rendered == "" {
		return 0, nil
	}

	parsed, err := time.ParseDuration(rendered)
	if err != nil {
		return 0, status.Errorf(codes.Internal, "delay template rendered an invalid duration %q: %v", rendered, err)
	}

	return types.Duration(max(0, parsed)), nil
}

func engineOr(ctx context.Context, engines []*template.Engine) *template.Engine {
	if len(engines) > 0 && engines[0] != nil {
		return engines[0]
	}

	return template.New(ctx, nil)
}

func validateDelay(engine *template.Engine, delay types.Delay) error {
	if delay.IsTemplate() {
		return engine.Validate(string(delay)) //nolint:wrapcheck
	}

	_, err := delay.Parse()

	return err //nolint:wrapcheck
}

func elementDelay(delay types.Delay, element stuber.GripMockElement) types.Delay {
	if element.HasDelay {
		return element.Delay
	}

	return delay
}

func delayTemplated(ctx context.Context, engine *template.Engine, delay types.Delay, data template.Data) error {
	delayDur, err := resolveDelay(engine, delay, data)
	if err != nil {
		return err
	}

	return delayResponse(ctx, delayDur)
}

func delayResponse(ctx context.Context, delayDur types.Duration) error {
	if delayDur == 0 {
		return nil
	}

	err := ctx.Err()
	if err != nil {
		return status.FromContextError(err).Err()
	}

	timer := time.NewTimer(time.Duration(delayDur))
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return status.FromContextError(ctx.Err()).Err()
	case <-timer.C:
		err := ctx.Err()
		if err != nil {
			return status.FromContextError(ctx.Err()).Err()
		}

		return nil
	}
}
