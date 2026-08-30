package app

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/bavix/gripmock/v3/internal/infra/proxycapture"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/types"
)

//nolint:gochecknoglobals
var capturedHeaderDenylist = map[string]struct{}{
	"authorization":       {},
	"proxy-authorization": {},
	"cookie":              {},
	"set-cookie":          {},
	"x-api-key":           {},
	"api-key":             {},
	"x-auth-token":        {},
	"traceparent":         {},
	"tracestate":          {},
	"b3":                  {},
	"x-b3-traceid":        {},
	"x-b3-spanid":         {},
	"x-b3-parentspanid":   {},
	"x-b3-sampled":        {},
	"x-request-id":        {},
	"x-correlation-id":    {},
	"grpc-timeout":        {},
	sessionHeaderKey:      {},
}

func requestHeadersFromMetadata(md metadata.MD) map[string]any {
	if len(md) == 0 {
		return nil
	}

	headers := processHeaders(md)
	for key := range headers {
		if _, denied := capturedHeaderDenylist[strings.ToLower(key)]; denied {
			delete(headers, key)
		}
	}

	if len(headers) == 0 {
		return nil
	}

	return headers
}

func responseHeadersFromMetadata(head metadata.MD, tail metadata.MD) map[string]string {
	return proxycapture.ResponseHeaders(head, tail)
}

func captureMetadata(head metadata.MD, tail metadata.MD) proxycapture.ResponseMetadata {
	return proxycapture.CaptureMetadata(head, tail)
}

func messageToAny(message proto.Message) any {
	return proxycapture.MessageToAny(message)
}

func selectCaptureError(firstErr, secondErr error) error {
	if firstErr != nil {
		return firstErr
	}

	return secondErr
}

func sanitizeCapturedStreamError(err error, hasResponses bool) error {
	if err == nil {
		return nil
	}

	if !hasResponses {
		return err
	}

	if status.Code(err) == codes.Canceled {
		return nil
	}

	return err
}

// recordProxyCall writes a proxied call into history (stubID is Nil — no stub
// served it). Runs for every proxy mode so real upstream traffic is inspectable
// and can seed new stubs. Health probes are excluded: periodic Check/Watch
// traffic would evict real calls from the bounded history store.
func (m *grpcMocker) recordProxyCall(
	ctx context.Context,
	startTime time.Time,
	requests []map[string]any,
	responses []any,
	respHeaders map[string]string,
	callErr error,
) {
	if m.recorder == nil || strings.HasPrefix(m.fullMethod, healthServicePrefix) {
		return
	}

	code := uint32(codes.OK)
	errMsg := ""

	if callErr != nil {
		code = uint32(status.Code(callErr))
		errMsg = callErr.Error()
	}

	recordCall(m.recorder, m.fullServiceName, m.methodName, sessionFromContext(ctx),
		uuid.Nil, code, startTime, requests, responses, respHeaders, errMsg)
}

func capturableResult(ctx context.Context, requests, responses int, callErr error) bool {
	if requests == 0 && responses == 0 {
		return false
	}

	if callErr != nil && status.Code(callErr) == codes.Canceled && ctx.Err() != nil {
		return false
	}

	return true
}

func (m *grpcMocker) recordCapturedStub(
	build func() *stuber.Stub,
	recordDelay bool,
	elapsed time.Duration,
) {
	stub := build()
	if stub == nil {
		return
	}

	if recordDelay && elapsed > 0 {
		stub.Output.Delay = types.NewDelay(elapsed)
	}

	m.budgerigar.PutMany(stub)
}

func (m *grpcMocker) captureBidiResult(
	downstreamCtx context.Context,
	clientStream grpc.ClientStream,
	captureCtx captureRequestContext,
	requests []map[string]any,
	responses []any,
	firstErr error,
	secondErr error,
	recordDelay bool,
	elapsed time.Duration,
) {
	captureErr := selectCaptureError(firstErr, secondErr)
	captureErr = sanitizeCapturedStreamError(captureErr, len(responses) > 0)

	if !capturableResult(downstreamCtx, len(requests), len(responses), captureErr) {
		return
	}

	m.recordCapturedStub(
		func() *stuber.Stub {
			return proxycapture.BuildBidiStub(
				m.fullServiceName, m.methodName, captureCtx.sessionID,
				requests, captureCtx.headers, responses,
				captureMetadataFromClientStream(clientStream), captureErr,
			)
		},
		recordDelay, elapsed,
	)
}
