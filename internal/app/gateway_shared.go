package app

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/infra/proxyroutes"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
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

// gatewayHandler holds shared dependencies for ConnectRPC and gRPC-Web gateways.
type gatewayHandler struct {
	budgerigar     *stuber.Budgerigar
	descriptors    *descriptors.Registry
	recorder       history.Recorder
	proxies        *proxyroutes.Registry
	validator      *validator.Validate
	errorFormatter *ErrorFormatter
}

func newGatewayHandler(
	budgerigar *stuber.Budgerigar,
	descriptorRegistry *descriptors.Registry,
	recorder history.Recorder,
	proxies *proxyroutes.Registry,
	validator *validator.Validate,
	errorFormatter *ErrorFormatter,
) gatewayHandler {
	e := errorFormatter
	if e == nil {
		e = NewErrorFormatter()
	}

	return gatewayHandler{
		budgerigar:     budgerigar,
		descriptors:    descriptorRegistry,
		recorder:       recorder,
		proxies:        proxies,
		validator:      validator,
		errorFormatter: e,
	}
}

func (h *gatewayHandler) buildMocker(r *http.Request, service, method, fullMethod string,
	methodDesc protoreflect.MethodDescriptor,
) *grpcMocker {
	return &grpcMocker{
		budgerigar:     h.budgerigar,
		templateEngine: template.New(r.Context(), nil),
		errorFormatter: h.errorFormatter,
		recorder:       h.recorder,
		descriptorResolver: &dynamicDescriptorResolver{
			static:  protoregistry.GlobalFiles,
			dynamic: h.descriptors,
		},
		proxies:            h.proxies,
		validator:          h.validator,
		fullServiceName:    service,
		serviceName:        service,
		methodName:         method,
		fullMethod:         fullMethod,
		inputDesc:          methodDesc.Input(),
		outputDesc:         methodDesc.Output(),
		serverStream:       methodDesc.IsStreamingServer(),
		clientStream:       methodDesc.IsStreamingClient(),
		strictServiceMatch: h.proxies != nil && h.proxies.RouteByMethod(fullMethod) != nil,
	}
}

// withoutDescriptorResponse abstracts protocol-specific response writing
// for the handleWithoutDescriptor flow.
type withoutDescriptorResponse interface {
	WriteError(w http.ResponseWriter, r *http.Request, code codes.Code, msg string)
	WriteSuccess(w http.ResponseWriter, r *http.Request)
}

//nolint:funlen
func (h *gatewayHandler) handleWithoutDescriptor(
	w http.ResponseWriter, r *http.Request,
	serviceName, methodName string,
	resp withoutDescriptorResponse,
) {
	_, _ = io.Copy(io.Discard, r.Body)

	requestTime := time.Now()
	emptyInput := map[string]any{}

	query := stuber.Query{
		Service: serviceName,
		Method:  methodName,
		Input:   []map[string]any{emptyInput},
		Headers: extractConnectHeaders(r.Header),
		Session: strings.TrimSpace(r.Header.Get("X-Gripmock-Session")),
	}

	result, findErr := h.budgerigar.FindByQuery(query)
	if findErr != nil || result == nil || result.Found() == nil {
		if result == nil {
			result = &stuber.Result{}
		}

		notFoundMsg := h.errorFormatter.FormatStubNotFoundError(query, result).Error()
		recordCall(h.recorder, serviceName, methodName, query.Session, uuid.Nil, uint32(codes.NotFound),
			requestTime, []map[string]any{emptyInput}, nil, notFoundMsg)
		resp.WriteError(w, r, codes.NotFound, notFoundMsg)

		return
	}

	found := result.Found()

	if err := delayResponse(r.Context(), found.Output.Delay); err != nil {
		st, _ := status.FromError(err)
		recordCall(h.recorder, serviceName, methodName, query.Session, found.ID, uint32(st.Code()),
			requestTime, []map[string]any{emptyInput}, nil, st.Message())
		resp.WriteError(w, r, st.Code(), st.Message())

		return
	}

	outputToUse := found.Output

	if st := outputStatusBase(outputToUse); st != nil {
		recordCall(h.recorder, serviceName, methodName, query.Session, found.ID, uint32(st.Code()),
			requestTime, []map[string]any{emptyInput}, nil, st.Message())
		resp.WriteError(w, r, st.Code(), st.Message())

		return
	}

	if outputToUse.Data != nil {
		recordCall(h.recorder, serviceName, methodName, query.Session, found.ID, uint32(codes.Unimplemented),
			requestTime, []map[string]any{emptyInput}, nil,
			"proto descriptor required to encode non-empty output for "+serviceName+"/"+methodName)
		resp.WriteError(w, r, codes.Unimplemented,
			"proto descriptor required to encode non-empty output for "+serviceName+"/"+methodName)

		return
	}

	for k, v := range outputToUse.Headers {
		w.Header().Set(k, v)
	}

	resp.WriteSuccess(w, r)

	recordCall(h.recorder, serviceName, methodName, query.Session, found.ID, uint32(codes.OK),
		requestTime, []map[string]any{emptyInput}, []map[string]any{{}}, "")
}

func decodeMessageData(data []byte, msg proto.Message, ct string, isJSONType func(string) bool) error {
	if isJSONType(ct) {
		return protojson.Unmarshal(data, msg)
	}

	return proto.Unmarshal(data, msg)
}

func encodeMessageData(msg proto.Message, ct string, isJSONType func(string) bool) ([]byte, error) {
	if isJSONType(ct) {
		return protojson.Marshal(msg)
	}

	return proto.Marshal(msg)
}
