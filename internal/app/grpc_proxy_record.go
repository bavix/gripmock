package app

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/bavix/gripmock/v3/internal/infra/proxycapture"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/types"
)

func requestHeadersFromMetadata(md metadata.MD) map[string]any {
	if len(md) == 0 {
		return nil
	}

	return processHeaders(md)
}

func responseHeadersFromMetadata(head metadata.MD, tail metadata.MD) map[string]string {
	return proxycapture.ResponseHeaders(head, tail)
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
		stub.Output.Delay = types.Duration(elapsed)
	}

	m.budgerigar.PutMany(stub)
}

func (m *grpcMocker) recordCapturedStubWithDelays(
	build func() *stuber.Stub,
	recordDelay bool,
	delays []time.Duration,
) {
	stub := build()
	if stub == nil {
		return
	}

	if recordDelay && len(delays) > 0 {
		for i, d := range delays {
			if d == 0 {
				continue
			}

			if stub.Output.Stream[i] == nil {
				continue
			}

			itemMap, ok := stub.Output.Stream[i].(map[string]any)
			if !ok {
				itemMap = map[string]any{"data": stub.Output.Stream[i]}
				stub.Output.Stream[i] = itemMap
			}

			itemMap["delay"] = d.String()
		}
	}

	m.budgerigar.PutMany(stub)
}

func (m *grpcMocker) captureBidiResultWithDelays(
	clientStream grpc.ClientStream,
	captureCtx captureRequestContext,
	requests []map[string]any,
	responses []map[string]any,
	firstErr error,
	secondErr error,
	recordDelay bool,
	delays []time.Duration,
) {
	captureErr := selectCaptureError(firstErr, secondErr)
	captureErr = sanitizeCapturedStreamError(captureErr, len(responses) > 0)

	m.recordCapturedStubWithDelays(
		func() *stuber.Stub {
			return proxycapture.BuildBidiStub(
				m.fullServiceName, m.methodName, captureCtx.sessionID,
				requests, captureCtx.headers, responses,
				responseHeadersFromClientStream(clientStream),
				captureErr,
			)
		},
		recordDelay, delays,
	)
}
