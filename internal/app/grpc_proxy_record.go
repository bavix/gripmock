package app

import (
	"time"

	"google.golang.org/grpc/metadata"
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

func messageToMap(message proto.Message) map[string]any {
	return proxycapture.MessageToMap(message)
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
