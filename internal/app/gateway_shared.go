package app

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	"github.com/bavix/gripmock/v3/internal/domain/history"
)

//nolint:ireturn
func findMethodDescriptor(files *descriptors.Registry, serviceName, methodName string) (protoreflect.MethodDescriptor, error) {
	if method := findMethodInGlobalFiles(serviceName, methodName); method != nil {
		return method, nil
	}

	if files == nil {
		return nil, &connectMethodNotFoundError{service: serviceName, method: methodName}
	}

	if method := findMethodInFiles(files, serviceName, methodName); method != nil {
		return method, nil
	}

	return nil, &connectMethodNotFoundError{service: serviceName, method: methodName}
}

func recordCall(
	recorder history.Recorder,
	service, method, session string,
	stubID uuid.UUID,
	code uint32,
	ts time.Time,
	requests, responses []map[string]any,
	errMsg string,
) {
	if recorder == nil {
		return
	}

	rec := history.CallRecord{
		StubID:    stubID,
		Service:   service,
		Method:    method,
		Session:   session,
		Code:      code,
		Error:     errMsg,
		Timestamp: ts,
		Requests:  requests,
		Responses: responses,
	}

	if len(requests) > 0 {
		rec.Request = requests[0]
	}

	if len(responses) > 0 {
		rec.Response = responses[0]
	}

	recorder.Record(rec)
}

type baseStreamAdapter struct {
	req *http.Request
	w   http.ResponseWriter

	mu             sync.Mutex
	sendHeaderOnce sync.Once
	endOfStream    atomic.Bool

	ctx context.Context //nolint:containedctx
}

func (a *baseStreamAdapter) Context() context.Context {
	return a.ctx
}

func (a *baseStreamAdapter) SetHeader(md metadata.MD) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for k, v := range md {
		for _, val := range v {
			a.w.Header().Add(k, val)
		}
	}

	return nil
}

func (a *baseStreamAdapter) SendHeader(md metadata.MD) error {
	return a.SetHeader(md)
}

func (a *baseStreamAdapter) SetTrailer(md metadata.MD) {
	_ = a.SetHeader(md)
}
