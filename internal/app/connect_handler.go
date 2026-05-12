package app

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
)

// connectErrorCodeStr maps gRPC codes to Connect protocol error code strings.
//
//nolint:gochecknoglobals
var connectErrorCodeStr = map[codes.Code]string{
	codes.Canceled:           "canceled",
	codes.Unknown:            "unknown",
	codes.InvalidArgument:    "invalid_argument",
	codes.DeadlineExceeded:   "deadline_exceeded",
	codes.NotFound:           "not_found",
	codes.AlreadyExists:      "already_exists",
	codes.PermissionDenied:   "permission_denied",
	codes.ResourceExhausted:  "resource_exhausted",
	codes.FailedPrecondition: "failed_precondition",
	codes.Aborted:            "aborted",
	codes.OutOfRange:         "out_of_range",
	codes.Unimplemented:      "unimplemented",
	codes.Internal:           "internal",
	codes.Unavailable:        "unavailable",
	codes.DataLoss:           "data_loss",
	codes.Unauthenticated:    "unauthenticated",
}

// connectHTTPStatus maps gRPC codes to HTTP status codes per the Connect protocol spec.
//
//nolint:gochecknoglobals
var connectHTTPStatus = map[codes.Code]int{
	codes.Canceled:           http.StatusRequestTimeout,
	codes.Unknown:            http.StatusInternalServerError,
	codes.InvalidArgument:    http.StatusBadRequest,
	codes.DeadlineExceeded:   http.StatusGatewayTimeout,
	codes.NotFound:           http.StatusNotFound,
	codes.AlreadyExists:      http.StatusConflict,
	codes.PermissionDenied:   http.StatusForbidden,
	codes.ResourceExhausted:  http.StatusTooManyRequests,
	codes.FailedPrecondition: http.StatusBadRequest,
	codes.Aborted:            http.StatusConflict,
	codes.OutOfRange:         http.StatusBadRequest,
	codes.Unimplemented:      http.StatusNotImplemented,
	codes.Internal:           http.StatusInternalServerError,
	codes.Unavailable:        http.StatusServiceUnavailable,
	codes.DataLoss:           http.StatusInternalServerError,
	codes.Unauthenticated:    http.StatusUnauthorized,
}

// connectExcludedHeaders are HTTP headers that should not be forwarded to stub matching.
//
//nolint:gochecknoglobals
var connectExcludedHeaders = map[string]struct{}{
	"accept":                   {},
	"accept-encoding":          {},
	"content-encoding":         {},
	"content-length":           {},
	"content-type":             {},
	"connect-protocol-version": {},
	"connect-timeout-ms":       {},
	"user-agent":               {},
}

// ConnectHandler serves the Connect RPC protocol (unary only) over plain HTTP.
type ConnectHandler struct {
	budgerigar  *stuber.Budgerigar
	descriptors *descriptors.Registry
	recorder    history.Recorder
}

// NewConnectHandler creates a ConnectHandler.
func NewConnectHandler(
	budgerigar *stuber.Budgerigar,
	reg *descriptors.Registry,
	recorder history.Recorder,
) *ConnectHandler {
	return &ConnectHandler{
		budgerigar:  budgerigar,
		descriptors: reg,
		recorder:    recorder,
	}
}

// ServeHTTP handles a Connect protocol unary request.
//
//nolint:cyclop,funlen
func (h *ConnectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	ct := r.Header.Get("Content-Type")

	// application/connect+* and application/grpc* use envelope framing — not supported yet.
	if strings.HasPrefix(ct, "application/connect+") || strings.HasPrefix(ct, "application/grpc") {
		h.writeError(w, codes.Unimplemented, "streaming is not supported")

		return
	}

	zerolog.Ctx(r.Context()).Info().Str("method", r.Method).Str("path", r.URL.Path).Msg("connect: handling request")

	serviceName, methodName, ok := parseConnectPath(r.URL.Path)
	if !ok {
		h.writeError(w, codes.Unimplemented, "invalid path: "+r.URL.Path)

		return
	}

	methodDesc, err := h.findMethodDescriptor(serviceName, methodName)
	if err != nil {
		// No proto descriptor found. Fall back to descriptor-less stub matching:
		// passes an empty input map which matches any stub with input.equals: {}.
		h.serveWithoutDescriptor(w, r, ct, serviceName, methodName)

		return
	}

	body, err := readConnectBody(r)
	if err != nil {
		h.writeError(w, codes.Internal, "failed to read request body")

		return
	}

	inputMsg := dynamicpb.NewMessage(methodDesc.Input())

	if isJSONConnect(ct) {
		if err := protojson.Unmarshal(body, inputMsg); err != nil {
			h.writeError(w, codes.InvalidArgument, "failed to decode JSON request: "+err.Error())

			return
		}
	} else {
		if err := proto.Unmarshal(body, inputMsg); err != nil {
			h.writeError(w, codes.InvalidArgument, "failed to decode proto request: "+err.Error())

			return
		}
	}

	requestTime := time.Now()
	requestData := convertToMap(inputMsg)

	query := stuber.Query{
		Service: serviceName,
		Method:  methodName,
		Input:   []map[string]any{requestData},
		Headers: extractConnectHeaders(r.Header),
		Session: strings.TrimSpace(r.Header.Get("X-Gripmock-Session")),
	}

	result, findErr := h.budgerigar.FindByQuery(query)
	if findErr != nil || result == nil || result.Found() == nil {
		if result == nil {
			result = &stuber.Result{}
		}

		notFoundMsg := NewErrorFormatter().FormatStubNotFoundError(query, result).Error()
		h.record(r, serviceName, methodName, query.Session, uuid.Nil, uint32(codes.NotFound), requestTime, []map[string]any{requestData}, nil, notFoundMsg)
		h.writeError(w, codes.NotFound, notFoundMsg)

		return
	}

	found := result.Found()

	if err := delayResponse(r.Context(), found.Output.Delay); err != nil {
		st, _ := status.FromError(err)
		h.record(r, serviceName, methodName, query.Session, found.ID, uint32(st.Code()), requestTime, []map[string]any{requestData}, nil, st.Message())
		h.writeError(w, st.Code(), st.Message())

		return
	}

	engine := template.New(r.Context(), nil)
	headers := extractConnectHeaders(r.Header)

	templateData := template.Data{
		Request:      requestData,
		Headers:      headers,
		MessageIndex: 0,
		RequestTime:  requestTime,
		Timestamp:    requestTime,
		State:        make(map[string]any),
		Requests:     []any{requestData},
		StubID:       found.ID.String(),
		RequestID:    found.ID.String(),
	}

	outputDataCopy := deepCopyMapAny(found.Output.Data)
	if err := engine.ProcessMap(outputDataCopy, templateData); err != nil {
		zerolog.Ctx(r.Context()).Err(err).Msg("connect: failed to process output templates")
		h.writeError(w, codes.Internal, "failed to process templates: "+err.Error())

		return
	}

	outputToUse := found.Output

	if template.HasTemplatesInHeaders(outputToUse.Headers) {
		headersCopy := deepCopyStringMap(outputToUse.Headers)
		if err := engine.ProcessHeaders(headersCopy, templateData); err != nil {
			h.writeError(w, codes.Internal, "failed to process header templates: "+err.Error())

			return
		}

		outputToUse.Headers = headersCopy
	}

	if outputToUse.Error != "" && template.IsTemplateString(outputToUse.Error) {
		rendered, err := engine.ProcessError(outputToUse.Error, templateData)
		if err != nil {
			h.writeError(w, codes.Internal, "failed to process error template: "+err.Error())

			return
		}

		outputToUse.Error = rendered
	}

	// If the stub declares an error, translate it to a Connect error response.
	if st := outputStatusBase(outputToUse); st != nil {
		h.record(r, serviceName, methodName, query.Session, found.ID, uint32(st.Code()), requestTime, []map[string]any{requestData}, nil, st.Message())
		h.writeError(w, st.Code(), st.Message())

		return
	}

	outputMsg, err := newOutputMessageFromDescriptor(methodDesc.Output(), outputDataCopy)
	if err != nil {
		h.writeError(w, codes.Internal, "failed to build response message: "+err.Error())

		return
	}

	var respBytes []byte

	if isJSONConnect(ct) {
		respBytes, err = protojson.Marshal(outputMsg)
		w.Header().Set("Content-Type", "application/json")
	} else {
		respBytes, err = proto.Marshal(outputMsg)
		w.Header().Set("Content-Type", "application/proto")
	}

	if err != nil {
		h.writeError(w, codes.Internal, "failed to encode response: "+err.Error())

		return
	}

	for k, v := range outputToUse.Headers {
		w.Header().Set(k, v)
	}

	h.record(r, serviceName, methodName, query.Session, found.ID, uint32(codes.OK), requestTime, []map[string]any{requestData}, []map[string]any{outputDataCopy}, "")

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

// serveWithoutDescriptor handles a Connect request when no proto method descriptor is
// available (e.g. no .proto file supplied at startup). It tries stub matching using an
// empty input map, which succeeds for any stub whose input is "input.equals: {}".
// For stubs with a non-empty output.data the caller still needs a descriptor to encode
// the response; those requests receive a clear 501 rather than a confusing "unknown
// service/method" message.
//
//nolint:cyclop
func (h *ConnectHandler) serveWithoutDescriptor(
	w http.ResponseWriter, r *http.Request,
	ct, serviceName, methodName string,
) {
	// Drain the body so the TCP connection can be reused.
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

		notFoundMsg := NewErrorFormatter().FormatStubNotFoundError(query, result).Error()
		h.record(r, serviceName, methodName, query.Session, uuid.Nil, uint32(codes.NotFound),
			requestTime, []map[string]any{emptyInput}, nil, notFoundMsg)
		h.writeError(w, codes.NotFound, notFoundMsg)

		return
	}

	found := result.Found()

	if err := delayResponse(r.Context(), found.Output.Delay); err != nil {
		st, _ := status.FromError(err)
		h.record(r, serviceName, methodName, query.Session, found.ID, uint32(st.Code()),
			requestTime, []map[string]any{emptyInput}, nil, st.Message())
		h.writeError(w, st.Code(), st.Message())

		return
	}

	outputToUse := found.Output

	if st := outputStatusBase(outputToUse); st != nil {
		h.record(r, serviceName, methodName, query.Session, found.ID, uint32(st.Code()),
			requestTime, []map[string]any{emptyInput}, nil, st.Message())
		h.writeError(w, st.Code(), st.Message())

		return
	}

	if len(outputToUse.Data) > 0 {
		// Non-empty output requires the proto descriptor to encode the response fields.
		h.writeError(w, codes.Unimplemented,
			"proto descriptor required to encode non-empty output for "+serviceName+"/"+methodName)

		return
	}

	// Empty output: return empty proto bytes (valid encoding for any all-default message).
	for k, v := range outputToUse.Headers {
		w.Header().Set(k, v)
	}

	if isJSONConnect(ct) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	} else {
		w.Header().Set("Content-Type", "application/proto")
		w.WriteHeader(http.StatusOK)
		// Empty body == empty proto message (zero bytes is the canonical encoding).
	}

	h.record(r, serviceName, methodName, query.Session, found.ID, uint32(codes.OK),
		requestTime, []map[string]any{emptyInput}, []map[string]any{{}}, "")
}

// writeError sends a Connect protocol error response (JSON body with code + message).
func (h *ConnectHandler) writeError(w http.ResponseWriter, code codes.Code, msg string) {
	codeStr, ok := connectErrorCodeStr[code]
	if !ok {
		codeStr = "internal"
		code = codes.Internal
	}

	httpStatus, ok := connectHTTPStatus[code]
	if !ok {
		httpStatus = http.StatusInternalServerError
	}

	body, _ := json.Marshal(map[string]string{
		"code":    codeStr,
		"message": msg,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_, _ = w.Write(body)
}

// record writes a call to the history recorder; no-op when recorder is nil.
func (h *ConnectHandler) record(
	r *http.Request,
	service, method, session string,
	stubID uuid.UUID,
	code uint32,
	ts time.Time,
	requests, responses []map[string]any,
	errMsg string,
) {
	if h.recorder == nil {
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

	h.recorder.Record(rec)
}

// findMethodDescriptor looks up the method in protoregistry.GlobalFiles, then the dynamic registry.
func (h *ConnectHandler) findMethodDescriptor(serviceName, methodName string) (protoreflect.MethodDescriptor, error) { //nolint:ireturn
	if md := findMethodInGlobalFiles(serviceName, methodName); md != nil {
		return md, nil
	}

	var found protoreflect.MethodDescriptor

	h.descriptors.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		svcs := file.Services()

		for i := range svcs.Len() {
			svc := svcs.Get(i)
			if string(svc.FullName()) != serviceName {
				continue
			}

			methods := svc.Methods()

			for j := range methods.Len() {
				m := methods.Get(j)
				if string(m.Name()) != methodName {
					continue
				}

				found = m

				return false
			}
		}

		return true
	})

	if found == nil {
		return nil, &connectMethodNotFoundError{service: serviceName, method: methodName}
	}

	return found, nil
}

type connectMethodNotFoundError struct {
	service string
	method  string
}

func (e *connectMethodNotFoundError) Error() string {
	return "unknown service/method: " + e.service + "/" + e.method
}

// parseConnectPath splits a Connect URL path into (service, method).
// Connect paths have the form /{package.ServiceName}/{MethodName}.
func parseConnectPath(path string) (service, method string, ok bool) {
	path = strings.TrimPrefix(path, "/")
	idx := strings.LastIndex(path, "/")

	if idx <= 0 || idx == len(path)-1 {
		return "", "", false
	}

	return path[:idx], path[idx+1:], true
}

// readConnectBody reads the request body, transparently decompressing gzip if indicated.
func readConnectBody(r *http.Request) ([]byte, error) {
	src := r.Body

	if strings.EqualFold(r.Header.Get("Content-Encoding"), "gzip") {
		gr, err := gzip.NewReader(src)
		if err != nil {
			return nil, err
		}

		defer gr.Close()

		src = gr
	}

	return io.ReadAll(src)
}

// isJSONConnect reports whether the Content-Type indicates JSON encoding.
func isJSONConnect(contentType string) bool {
	return strings.HasPrefix(contentType, "application/json")
}

// extractConnectHeaders converts HTTP headers to the map format used by stub matching,
// excluding Connect-protocol and transport-level headers.
func extractConnectHeaders(hdr http.Header) map[string]any {
	if len(hdr) == 0 {
		return nil
	}

	result := make(map[string]any, len(hdr))

	for k, vals := range hdr {
		lower := strings.ToLower(k)
		if _, excluded := connectExcludedHeaders[lower]; excluded {
			continue
		}

		result[lower] = strings.Join(vals, ";")
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

// newOutputMessageFromDescriptor converts a stub output map to a dynamicpb.Message.
// It reuses the package-level jsonBufferPool from grpc_server.go.
func newOutputMessageFromDescriptor(desc protoreflect.MessageDescriptor, data map[string]any) (*dynamicpb.Message, error) {
	pooled, _ := jsonBufferPool.Get().(*bytes.Buffer)
	if pooled == nil {
		pooled = bytes.NewBuffer(make([]byte, 0, jsonBufferInitialCap))
	}

	pooled.Reset()

	defer func() {
		pooled.Reset()
		jsonBufferPool.Put(pooled)
	}()

	converted := convertMapNumericToStringNumber(data)

	if err := json.NewEncoder(pooled).Encode(converted); err != nil {
		return nil, err
	}

	msg := dynamicpb.NewMessage(desc)

	if err := protojson.Unmarshal(pooled.Bytes(), msg); err != nil {
		return nil, err
	}

	return msg, nil
}

// Compile-time interface check.
var _ http.Handler = (*ConnectHandler)(nil)
