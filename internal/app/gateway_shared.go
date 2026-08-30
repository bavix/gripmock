package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	"github.com/bavix/gripmock/v3/internal/domain/history"
	protosetinfra "github.com/bavix/gripmock/v3/internal/infra/protoset"
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
	requests []map[string]any,
	responses []any,
	respHeaders map[string]string,
	errMsg string,
) {
	if recorder == nil {
		return
	}

	rec := history.CallRecord{
		StubID:          stubID,
		Service:         service,
		Method:          method,
		Session:         session,
		Code:            code,
		Error:           errMsg,
		ElapsedMS:       time.Since(ts).Milliseconds(),
		Timestamp:       ts,
		Requests:        requests,
		Responses:       responses,
		ResponseHeaders: respHeaders,
	}

	recordOwned(recorder, rec)
}

// recordOwned hands the freshly built maps to the store without a defensive
// clone; every internal call site constructs them per call and never touches
// them afterwards.
func recordOwned(recorder history.Recorder, rec history.CallRecord) {
	if owned, ok := recorder.(interface{ RecordOwned(rec history.CallRecord) }); ok {
		owned.RecordOwned(rec)

		return
	}

	recorder.Record(rec)
}

type baseStreamAdapter struct {
	req *http.Request
	w   http.ResponseWriter

	typeResolver *protosetinfra.TypeResolver

	frameEncoding string

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

func (a *baseStreamAdapter) SetTrailer(_ metadata.MD) {}

type gatewayHandler struct {
	budgerigar     *stuber.Budgerigar
	descriptors    *descriptors.Registry
	recorder       history.Recorder
	proxyRoutesRef *atomic.Pointer[proxyroutes.Registry]
	validator      *validator.Validate
	errorFormatter *ErrorFormatter
	reflection     *gatewayReflection

	// templateEngine is built once: it carries the whole plugin function table
	// and the parsed-template cache, both of which were rebuilt per request.
	templateEngine *template.Engine

	typeResolver *protosetinfra.TypeResolver
}

func newGatewayHandler(
	ctx context.Context,
	budgerigar *stuber.Budgerigar,
	descriptorRegistry *descriptors.Registry,
	recorder history.Recorder,
	proxyRoutesRef *atomic.Pointer[proxyroutes.Registry],
	validator *validator.Validate,
	errorFormatter *ErrorFormatter,
	engines ...*template.Engine,
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
		reflection:     newGatewayReflection(descriptorRegistry),
		templateEngine: engineOr(context.WithoutCancel(ctx), engines),
		typeResolver: protosetinfra.NewTypeResolver(&dynamicDescriptorResolver{
			static:  protoregistry.GlobalFiles,
			dynamic: descriptorRegistry,
		}),
	}
}

func (h *gatewayHandler) buildMocker(_ *http.Request, service, method, fullMethod string,
	methodDesc protoreflect.MethodDescriptor,
) *grpcMocker {
	var proxies *proxyroutes.Registry
	if h.proxyRoutesRef != nil {
		proxies = h.proxyRoutesRef.Load()
	}

	return &grpcMocker{
		budgerigar:         h.budgerigar,
		templateEngine:     h.templateEngine,
		errorFormatter:     h.errorFormatter,
		recorder:           h.recorder,
		typeResolver:       h.typeResolver,
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
			requestTime, []map[string]any{emptyInput}, nil, nil, notFoundMsg)
		resp.WriteError(w, r, codes.NotFound, notFoundMsg)

		return
	}

	found := result.Found()

	td := newTemplateData(emptyInput, query.Headers, 0, requestTime,
		[]any{emptyInput}, found, result.MatchNumber())

	err := delayTemplated(r.Context(), h.templateEngine, found.Output.Delay, td)
	if err != nil {
		st, _ := status.FromError(err)
		recordCall(h.recorder, serviceName, methodName, query.Session, found.ID, uint32(st.Code()),
			requestTime, []map[string]any{emptyInput}, nil, nil, st.Message())
		resp.WriteError(w, r, st.Code(), st.Message())

		return
	}

	outputToUse := found.Output

	if st := outputStatusBase(outputToUse); st != nil {
		recordCall(h.recorder, serviceName, methodName, query.Session, found.ID, uint32(st.Code()),
			requestTime, []map[string]any{emptyInput}, nil, nil, st.Message())
		resp.WriteError(w, r, st.Code(), st.Message())

		return
	}

	if outputToUse.Data != nil || outputToUse.HasTemplate() {
		recordCall(h.recorder, serviceName, methodName, query.Session, found.ID, uint32(codes.Unimplemented),
			requestTime, []map[string]any{emptyInput}, nil, nil,
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
		requestTime, []map[string]any{emptyInput}, []any{map[string]any{}}, outputToUse.Headers, "")
}

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

// fieldMaskPathsToString flattens a decoded FieldMask object into its comma-separated JSON form.
//

func fieldMaskPathsToString(val any) (string, bool) {
	obj, ok := val.(map[string]any)
	if !ok {
		return "", false
	}

	pathsRaw, ok := obj["paths"]
	if !ok {
		return "", false
	}

	pathsArr, ok := pathsRaw.([]any)
	if !ok || len(pathsArr) == 0 {
		return "", false
	}

	paths := make([]string, 0, len(pathsArr))
	for _, p := range pathsArr {
		if s, ok := p.(string); ok {
			paths = append(paths, s)
		}
	}

	if len(paths) == 0 {
		return "", false
	}

	return strings.Join(paths, ","), true
}

func normalizeFieldMaskJSON(data []byte, msg proto.Message) []byte {
	fieldMaskNames := collectFieldMaskNames(msg)
	if len(fieldMaskNames) == 0 {
		return data
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	var rawMap map[string]any

	err := decoder.Decode(&rawMap)
	if err != nil {
		return data
	}

	modified := false

	for key, val := range rawMap {
		if _, ok := fieldMaskNames[key]; !ok {
			continue
		}

		joined, ok := fieldMaskPathsToString(val)
		if !ok {
			continue
		}

		rawMap[key] = joined
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

const anyTypeURLPrefix = "type.googleapis.com/"

func serializeErrorStatus(st *status.Status) connectError {
	sp := st.Proto()

	details := make([]connectErrorDetail, 0, len(sp.GetDetails()))
	debug := debugRenderedDetails(sp)

	for i, detail := range sp.GetDetails() {
		entry := connectErrorDetail{
			Type:  strings.TrimPrefix(detail.GetTypeUrl(), anyTypeURLPrefix),
			Value: base64.RawStdEncoding.EncodeToString(detail.GetValue()),
		}

		if i < len(debug) {
			entry.Debug = debug[i]
		}

		details = append(details, entry)
	}

	return connectError{
		Code:    ErrorCodeToString(st.Code()),
		Message: st.Message(),
		Details: details,
	}
}

func debugRenderedDetails(sp *spb.Status) []json.RawMessage {
	if len(sp.GetDetails()) == 0 {
		return nil
	}

	statusData, err := protosetinfra.GlobalTypeResolver().Marshal(sp)
	if err != nil {
		return nil
	}

	var rendered struct {
		Details []json.RawMessage `json:"details"`
	}

	err = json.Unmarshal(statusData, &rendered)
	if err != nil {
		return nil
	}

	return rendered.Details
}

func normalizeHealthError(st *status.Status, serviceName string) *status.Status {
	if serviceName == "grpc.health.v1.Health" && st.Code() == codes.NotFound {
		return status.New(codes.NotFound, "unknown service")
	}

	return st
}

func decodeMessageData(
	data []byte,
	msg proto.Message,
	ct string,
	isJSONType func(string) bool,
	resolver *protosetinfra.TypeResolver,
) error {
	if isJSONType(ct) {
		normalized := normalizeFieldMaskJSON(data, msg)

		return resolver.Unmarshal(normalized, msg)
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

	err := decodeMessageData(data, inputMsg, contentType, isJSONType, mocker.typeResolver)
	if err != nil {
		writeError(status.New(codes.InvalidArgument, "failed to unmarshal: "+err.Error()))

		return nil, err
	}

	resp, err := mocker.handleUnaryWithProxy(ctx, stream, inputMsg)
	if err != nil {
		st, _ := status.FromError(err)
		writeError(st)

		return nil, err
	}

	return resp, nil
}

func encodeMessageData(
	msg proto.Message,
	ct string,
	isJSONType func(string) bool,
	resolver *protosetinfra.TypeResolver,
) ([]byte, error) {
	if isJSONType(ct) {
		return resolver.MarshalProtoNames(msg)
	}

	return proto.Marshal(msg)
}
