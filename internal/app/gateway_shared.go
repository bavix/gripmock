package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"

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
		ElapsedMS: time.Since(ts).Milliseconds(),
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
	proxyRoutesRef *atomic.Pointer[proxyroutes.Registry]
	validator      *validator.Validate
	errorFormatter *ErrorFormatter
}

func newGatewayHandler(
	budgerigar *stuber.Budgerigar,
	descriptorRegistry *descriptors.Registry,
	recorder history.Recorder,
	proxyRoutesRef *atomic.Pointer[proxyroutes.Registry],
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
		proxyRoutesRef: proxyRoutesRef,
		validator:      validator,
		errorFormatter: e,
	}
}

func (h *gatewayHandler) buildMocker(r *http.Request, service, method, fullMethod string,
	methodDesc protoreflect.MethodDescriptor,
) *grpcMocker {
	var proxies *proxyroutes.Registry
	if h.proxyRoutesRef != nil {
		proxies = h.proxyRoutesRef.Load()
	}

	return &grpcMocker{
		budgerigar:     h.budgerigar,
		templateEngine: template.New(r.Context(), nil),
		errorFormatter: h.errorFormatter,
		recorder:       h.recorder,
		descriptorResolver: &dynamicDescriptorResolver{
			static:  protoregistry.GlobalFiles,
			dynamic: h.descriptors,
		},
		proxies:            proxies,
		validator:          h.validator,
		fullServiceName:    service,
		serviceName:        service,
		methodName:         method,
		fullMethod:         fullMethod,
		inputDesc:          methodDesc.Input(),
		outputDesc:         methodDesc.Output(),
		serverStream:       methodDesc.IsStreamingServer(),
		clientStream:       methodDesc.IsStreamingClient(),
		strictServiceMatch: proxies != nil && proxies.RouteByMethod(fullMethod) != nil,
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

// collectFieldMaskNames returns a set of JSON field names that are
// google.protobuf.FieldMask in the given message descriptor.
func collectFieldMaskNames(msg proto.Message) map[string]struct{} {
	if msg == nil {
		return nil
	}

	desc := msg.ProtoReflect().Descriptor()
	if desc == nil {
		return nil
	}

	fields := desc.Fields()
	result := make(map[string]struct{}, fields.Len())

	for i := range fields.Len() {
		fd := fields.Get(i)
		if fd.Kind() == protoreflect.MessageKind &&
			string(fd.Message().FullName()) == "google.protobuf.FieldMask" {
			result[string(fd.Name())] = struct{}{}
		}
	}

	return result
}

// normalizeFieldMaskJSON converts well-known FieldMask fields from object JSON
// form {"paths": ["a","b"]} to string form "a,b" which is what protojson
// expects for google.protobuf.FieldMask. Only fields confirmed as FieldMask
// via the message descriptor are normalized. Returns the original data unchanged
// if no FieldMask fields are found or if conversion isn't needed.
//
//nolint:cyclop
func normalizeFieldMaskJSON(data []byte, msg proto.Message) []byte {
	fieldMaskNames := collectFieldMaskNames(msg)
	if len(fieldMaskNames) == 0 {
		return data
	}

	var rawMap map[string]any
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return data
	}

	modified := false

	for key, val := range rawMap {
		if _, ok := fieldMaskNames[key]; !ok {
			continue
		}

		obj, ok := val.(map[string]any)
		if !ok {
			continue
		}

		pathsRaw, ok := obj["paths"]
		if !ok {
			continue
		}

		pathsArr, ok := pathsRaw.([]any)
		if !ok || len(pathsArr) == 0 {
			continue
		}

		paths := make([]string, 0, len(pathsArr))
		for _, p := range pathsArr {
			if s, ok := p.(string); ok {
				paths = append(paths, s)
			}
		}

		if len(paths) == 0 {
			continue
		}

		rawMap[key] = strings.Join(paths, ",")
		modified = true
	}

	if !modified {
		return data
	}

	result, err := json.Marshal(rawMap)
	if err != nil {
		return data
	}

	return result
}

// serializeErrorStatus converts a *status.Status to a connectError JSON struct
// with properly serialized error details (including @type annotations via protojson).
//
//nolint:nestif
func serializeErrorStatus(st *status.Status) connectError {
	sp := st.Proto()

	details := make([]map[string]any, 0, len(sp.GetDetails()))
	if len(sp.GetDetails()) > 0 {
		statusData, err := protojson.MarshalOptions{UseProtoNames: false}.Marshal(sp)
		if err == nil {
			var statusObj map[string]any
			if err := json.Unmarshal(statusData, &statusObj); err == nil {
				if d, ok := statusObj["details"].([]any); ok {
					details = make([]map[string]any, 0, len(d))
					for _, item := range d {
						if m, ok := item.(map[string]any); ok {
							details = append(details, m)
						}
					}
				}
			}
		}
	}

	return connectError{
		Code:    ErrorCodeToString(st.Code()),
		Message: st.Message(),
		Details: details,
	}
}

// normalizeHealthError returns "unknown service" NotFound for unknown health check services,
// matching the standard gRPC health check protocol behavior.
func normalizeHealthError(st *status.Status, serviceName string) *status.Status {
	if serviceName == "grpc.health.v1.Health" && st.Code() == codes.NotFound {
		return status.New(codes.NotFound, "unknown service")
	}

	return st
}

func decodeMessageData(data []byte, msg proto.Message, ct string, isJSONType func(string) bool) error {
	if isJSONType(ct) {
		// Preprocess FieldMask fields (accept {"paths": [...]} format)
		normalized := normalizeFieldMaskJSON(data, msg)

		return protojson.Unmarshal(normalized, msg)
	}

	return proto.Unmarshal(data, msg)
}

func handleUnaryCore(
	ctx context.Context,
	stream grpc.ServerStream,
	data []byte,
	mocker *grpcMocker,
	contentType string,
	isJSONType func(string) bool,
	writeError func(*status.Status),
) (any, error) {
	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)
	if isJSONType(contentType) {
		normalized := normalizeFieldMaskJSON(data, inputMsg)
		if err := protojson.Unmarshal(normalized, inputMsg); err != nil {
			writeError(status.New(codes.InvalidArgument, "failed to unmarshal: "+err.Error()))

			return nil, err
		}
	} else {
		if err := proto.Unmarshal(data, inputMsg); err != nil {
			writeError(status.New(codes.InvalidArgument, "failed to unmarshal: "+err.Error()))

			return nil, err
		}
	}

	resp, err := mocker.handleUnaryWithProxy(ctx, stream, inputMsg)
	if err != nil {
		st, _ := status.FromError(err)
		writeError(st)

		return nil, err
	}

	return resp, nil
}

func encodeMessageData(msg proto.Message, ct string, isJSONType func(string) bool) ([]byte, error) {
	if isJSONType(ct) {
		return protojson.MarshalOptions{
			UseProtoNames: true,
		}.Marshal(msg)
	}

	return proto.Marshal(msg)
}
