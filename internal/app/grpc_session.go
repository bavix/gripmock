package app

import (
	"context"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/infra/session"
)

func sessionFromMetadata(md metadata.MD) string {
	for _, v := range md.Get(sessionHeaderKey) {
		if sessionID := strings.TrimSpace(v); sessionID != "" {
			session.Touch(sessionID)

			return sessionID
		}
	}

	return ""
}

func sessionFromContext(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		return sessionFromMetadata(md)
	}

	return ""
}

type bidiRecordingStream struct {
	grpc.ServerStream

	requests  []map[string]any
	responses []map[string]any
	stubID    uuid.UUID
	maxItems  int
}

func (s *bidiRecordingStream) RecvMsg(m any) error {
	err := s.ServerStream.RecvMsg(m)
	if err != nil {
		return err
	}

	if msgMap := protoToMap(m); msgMap != nil && len(s.requests) < s.maxItems {
		s.requests = append(s.requests, msgMap)
	}

	return nil
}

func (s *bidiRecordingStream) SendMsg(m any) error {
	err := s.ServerStream.SendMsg(m)
	if err != nil {
		return err
	}

	if msgMap := protoToMap(m); msgMap != nil && len(s.responses) < s.maxItems {
		s.responses = append(s.responses, msgMap)
	}

	return nil
}

func (s *bidiRecordingStream) getRequests() []map[string]any { return s.requests }

func (s *bidiRecordingStream) getResponses() []map[string]any { return s.responses }

func (s *bidiRecordingStream) setStubID(id uuid.UUID) { s.stubID = id }

func (s *bidiRecordingStream) getStubID() uuid.UUID { return s.stubID }

func (m *grpcMocker) recordCall(
	ctx context.Context,
	stubID uuid.UUID,
	code uint32,
	timestamp time.Time,
	requests []map[string]any,
	responses []any,
	errMsg string,
) {
	if m.recorder == nil || len(requests) == 0 {
		return
	}

	recordedResponses := make([]map[string]any, 0, len(responses))
	for _, r := range responses {
		if m, ok := r.(map[string]any); ok {
			recordedResponses = append(recordedResponses, m)
		}
	}

	rec := history.CallRecord{
		Service:   m.fullServiceName,
		Method:    m.methodName,
		Session:   sessionFromContext(ctx),
		Requests:  requests,
		Responses: recordedResponses,
		Error:     errMsg,
		Code:      code,
		StubID:    stubID,
		ElapsedMS: time.Since(timestamp).Milliseconds(),
		Timestamp: timestamp,
	}

	if len(requests) > 0 {
		rec.Request = requests[0]
	}

	if len(recordedResponses) > 0 {
		rec.Response = recordedResponses[0]
	}

	m.recorder.Record(rec)
}

func processHeaders(md metadata.MD) map[string]any {
	if len(md) == 0 {
		return nil
	}

	headers := make(map[string]any, len(md))

	for k, v := range md {
		if _, excluded := excludedHeaders[k]; !excluded {
			headers[k] = strings.Join(v, ";")
		}
	}

	return headers
}

func sendStreamMessage(stream grpc.ServerStream, msg *dynamicpb.Message) error {
	if err := stream.SendMsg(msg); err != nil {
		return errors.Wrap(err, "failed to send response")
	}

	return nil
}

func receiveStreamMessage(stream grpc.ServerStream, msg *dynamicpb.Message) error {
	err := stream.RecvMsg(msg)
	if err != nil {
		return errors.Wrap(err, "failed to receive message")
	}

	return nil
}
