package app

import (
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"

	"github.com/bavix/gripmock/v3/internal/infra/proxycapture"
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

func messageToMap(message proto.Message) map[string]any {
	return proxycapture.MessageToMap(message)
}

func (m *grpcMocker) recordCapturedUnaryStub(
	request map[string]any,
	requestHeaders map[string]any,
	response map[string]any,
	responseHeaders map[string]string,
	callErr error,
	sessionID string,
	recordDelay bool,
	elapsed time.Duration,
) {
	stub := proxycapture.BuildUnaryStub(
		m.fullServiceName,
		m.methodName,
		sessionID,
		request,
		requestHeaders,
		response,
		responseHeaders,
		callErr,
	)

	if recordDelay && elapsed > 0 {
		stub.Output.Delay = types.Duration(elapsed)
	}

	m.budgerigar.PutMany(stub)
}

func (m *grpcMocker) recordCapturedServerStreamStub(
	request map[string]any,
	requestHeaders map[string]any,
	responses []map[string]any,
	responseHeaders map[string]string,
	callErr error,
	sessionID string,
	recordDelay bool,
	elapsed time.Duration,
) {
	stub := proxycapture.BuildServerStreamStub(
		m.fullServiceName,
		m.methodName,
		sessionID,
		request,
		requestHeaders,
		responses,
		responseHeaders,
		callErr,
	)

	if recordDelay && elapsed > 0 {
		stub.Output.Delay = types.Duration(elapsed)
	}

	m.budgerigar.PutMany(stub)
}

func (m *grpcMocker) recordCapturedClientStreamStub(
	requests []map[string]any,
	requestHeaders map[string]any,
	response map[string]any,
	responseHeaders map[string]string,
	callErr error,
	sessionID string,
	recordDelay bool,
	elapsed time.Duration,
) {
	stub := proxycapture.BuildClientStreamStub(
		m.fullServiceName,
		m.methodName,
		sessionID,
		requests,
		requestHeaders,
		response,
		responseHeaders,
		callErr,
	)

	if recordDelay && elapsed > 0 {
		stub.Output.Delay = types.Duration(elapsed)
	}

	m.budgerigar.PutMany(stub)
}

func (m *grpcMocker) recordCapturedBidiStub(
	requests []map[string]any,
	requestHeaders map[string]any,
	responses []map[string]any,
	responseHeaders map[string]string,
	callErr error,
	sessionID string,
	recordDelay bool,
	elapsed time.Duration,
) {
	stub := proxycapture.BuildBidiStub(
		m.fullServiceName,
		m.methodName,
		sessionID,
		requests,
		requestHeaders,
		responses,
		responseHeaders,
		callErr,
	)

	if recordDelay && elapsed > 0 {
		stub.Output.Delay = types.Duration(elapsed)
	}

	m.budgerigar.PutMany(stub)
}
