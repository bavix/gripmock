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
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
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

type ConnectRPCGateway struct {
	budgerigar     *stuber.Budgerigar
	descriptors    *descriptors.Registry
	recorder       history.Recorder
	proxies        *proxyroutes.Registry
	validator      *validator.Validate
	errorFormatter *ErrorFormatter
}

func NewConnectRPCGateway(
	budgerigar *stuber.Budgerigar,
	descriptorRegistry *descriptors.Registry,
	recorder history.Recorder,
	proxies *proxyroutes.Registry,
	validator *validator.Validate,
	errorFormatter *ErrorFormatter,
) *ConnectRPCGateway {
	e := errorFormatter
	if e == nil {
		e = NewErrorFormatter()
	}

	return &ConnectRPCGateway{
		budgerigar:     budgerigar,
		descriptors:    descriptorRegistry,
		recorder:       recorder,
		proxies:        proxies,
		validator:      validator,
		errorFormatter: e,
	}
}

//nolint:funlen
func (g *ConnectRPCGateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	vars := mux.Vars(r)
	service := vars["service"]
	method := vars["method"]
	fullMethod := "/" + service + "/" + method

	logger := zerolog.Ctx(r.Context())
	logger.Info().
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("service", service).
		Str("method", method).
		Msg("connect.gateway: handling request")

	methodDesc, err := g.findMethodDescriptor(service, method)
	if err != nil {
		if g.descriptors == nil && g.budgerigar != nil {
			g.handleWithoutDescriptor(w, r, service, method)

			return
		}

		g.writeError(w, codes.NotFound, "method not found")

		return
	}

	mocker := &grpcMocker{
		budgerigar:     g.budgerigar,
		templateEngine: template.New(r.Context(), nil),
		errorFormatter: g.errorFormatter,
		recorder:       g.recorder,
		descriptorResolver: &dynamicDescriptorResolver{
			static:  protoregistry.GlobalFiles,
			dynamic: g.descriptors,
		},
		proxies:            g.proxies,
		validator:          g.validator,
		fullServiceName:    service,
		serviceName:        service,
		methodName:         method,
		fullMethod:         fullMethod,
		inputDesc:          methodDesc.Input(),
		outputDesc:         methodDesc.Output(),
		serverStream:       methodDesc.IsStreamingServer(),
		clientStream:       methodDesc.IsStreamingClient(),
		strictServiceMatch: g.proxies != nil && g.proxies.RouteByMethod(fullMethod) != nil,
	}

	adapter := &httpStreamAdapter{
		ctx:       r.Context(),
		req:       r,
		w:         w,
		streaming: mocker.serverStream || mocker.clientStream,
	}

	adapter.ctx = httpHeadersToGRPCContext(r.Context(), r.Header)

	if !adapter.streaming {
		g.handleUnary(mocker, adapter)

		return
	}

	if err := mocker.streamHandler(adapter.ctx, adapter); err != nil { //nolint:contextcheck
		st, _ := status.FromError(err)
		adapter.writeError(st.Code(), st.Message())
	} else {
		// Per Connect RPC protocol, the server signals end of stream
		// by sending an empty envelope with the endStream flag set.
		if err := writeConnectFrame(adapter.w, nil, true); err != nil {
			logger.Debug().Err(err).Msg("connect.gateway: send end stream")
		}
	}
}

//nolint:nlreturn
func (g *ConnectRPCGateway) handleUnary(mocker *grpcMocker, a *httpStreamAdapter) {
	body, err := io.ReadAll(a.req.Body)
	if err != nil {
		a.writeError(codes.Internal, "failed to read body")
		return
	}

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)
	if isJSONContentType(a.req.Header.Get("Content-Type")) {
		if err := protojson.Unmarshal(body, inputMsg); err != nil {
			a.writeError(codes.InvalidArgument, "failed to unmarshal: "+err.Error())
			return
		}
	} else {
		if err := proto.Unmarshal(body, inputMsg); err != nil {
			a.writeError(codes.InvalidArgument, "failed to unmarshal: "+err.Error())
			return
		}
	}

	resp, err := mocker.handleUnary(a.ctx, inputMsg)
	if err != nil {
		st, _ := status.FromError(err)
		a.writeError(st.Code(), st.Message())

		return
	}

	if err := a.SendMsg(resp); err != nil {
		// The client may have disconnected before we could write the
		// response (e.g. context cancelled, keep-alive timeout). The
		// stub was matched and a response was produced, but the
		// transport write failed. We cannot change the response status
		// at this point (headers already flushed) so the only safe
		// action is to log the failure for observability.
		zerolog.Ctx(a.ctx).Debug().Err(err).Msg("connect.gateway: send unary response")
	}
}

//nolint:ireturn
func (g *ConnectRPCGateway) findMethodDescriptor(serviceName, methodName string) (protoreflect.MethodDescriptor, error) {
	if method := findMethodInGlobalFiles(serviceName, methodName); method != nil {
		return method, nil
	}

	if g.descriptors == nil {
		return nil, &connectMethodNotFoundError{service: serviceName, method: methodName}
	}

	if method := findMethodInFiles(g.descriptors, serviceName, methodName); method != nil {
		return method, nil
	}

	return nil, &connectMethodNotFoundError{service: serviceName, method: methodName}
}

//nolint:funlen
func (g *ConnectRPCGateway) handleWithoutDescriptor(w http.ResponseWriter, r *http.Request, serviceName, methodName string) {
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

	result, findErr := g.budgerigar.FindByQuery(query)
	if findErr != nil || result == nil || result.Found() == nil {
		if result == nil {
			result = &stuber.Result{}
		}

		notFoundMsg := g.errorFormatter.FormatStubNotFoundError(query, result).Error()
		g.record(serviceName, methodName, query.Session, uuid.Nil, uint32(codes.NotFound),
			requestTime, []map[string]any{emptyInput}, nil, notFoundMsg)
		g.writeError(w, codes.NotFound, notFoundMsg)

		return
	}

	found := result.Found()

	if err := delayResponse(r.Context(), found.Output.Delay); err != nil {
		st, _ := status.FromError(err)
		g.record(serviceName, methodName, query.Session, found.ID, uint32(st.Code()),
			requestTime, []map[string]any{emptyInput}, nil, st.Message())
		g.writeError(w, st.Code(), st.Message())

		return
	}

	outputToUse := found.Output

	if st := outputStatusBase(outputToUse); st != nil {
		g.record(serviceName, methodName, query.Session, found.ID, uint32(st.Code()),
			requestTime, []map[string]any{emptyInput}, nil, st.Message())
		g.writeError(w, st.Code(), st.Message())

		return
	}

	if outputToUse.Data != nil {
		g.record(serviceName, methodName, query.Session, found.ID, uint32(codes.Unimplemented),
			requestTime, []map[string]any{emptyInput}, nil,
			"proto descriptor required to encode non-empty output for "+serviceName+"/"+methodName)
		g.writeError(w, codes.Unimplemented,
			"proto descriptor required to encode non-empty output for "+serviceName+"/"+methodName)

		return
	}

	for k, v := range outputToUse.Headers {
		w.Header().Set(k, v)
	}

	ct := r.Header.Get("Content-Type")
	if isJSONContentType(ct) {
		w.Header().Set("Content-Type", "application/connect+json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	} else {
		w.Header().Set("Content-Type", "application/connect+proto")
		w.WriteHeader(http.StatusOK)
	}

	g.record(serviceName, methodName, query.Session, found.ID, uint32(codes.OK),
		requestTime, []map[string]any{emptyInput}, []map[string]any{{}}, "")
}

func (g *ConnectRPCGateway) record(
	service, method, session string,
	stubID uuid.UUID,
	code uint32,
	ts time.Time,
	requests, responses []map[string]any,
	errMsg string,
) {
	if g.recorder == nil {
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

	g.recorder.Record(rec)
}

func (g *ConnectRPCGateway) writeError(w http.ResponseWriter, code codes.Code, msg string) {
	body, _ := json.Marshal(connectError{
		Code:    ErrorCodeToString(code),
		Message: msg,
		Details: []map[string]any{},
	})

	w.Header().Set("Content-Type", "application/connect+json")
	w.WriteHeader(ErrorCodeToHTTPStatus(code))
	_, _ = w.Write(body)
}

func isJSONContentType(ct string) bool {
	return ct == "application/json" || ct == "application/connect+json"
}

type httpStreamAdapter struct {
	req *http.Request
	w   http.ResponseWriter

	mu             sync.Mutex
	sendHeaderOnce sync.Once
	endOfStream    atomic.Bool

	streaming bool

	ctx context.Context //nolint:containedctx
}

func (a *httpStreamAdapter) Context() context.Context {
	return a.ctx
}

func (a *httpStreamAdapter) SetHeader(md metadata.MD) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for k, v := range md {
		for _, val := range v {
			a.w.Header().Add(k, val)
		}
	}

	return nil
}

func (a *httpStreamAdapter) SendHeader(md metadata.MD) error {
	return a.SetHeader(md)
}

func (a *httpStreamAdapter) SetTrailer(md metadata.MD) {
	_ = a.SetHeader(md)
}

func (a *httpStreamAdapter) SendMsg(m any) error {
	a.sendHeader()

	msg, ok := m.(proto.Message)
	if !ok {
		return nil
	}

	ct := a.req.Header.Get("Content-Type")

	data, err := a.encodeMessage(msg, ct)
	if err != nil {
		return err
	}

	if a.streaming {
		if err := writeConnectFrame(a.w, data, false); err != nil {
			return err
		}
	} else {
		if _, err = a.w.Write(data); err != nil {
			return err
		}
	}

	if flusher, ok := a.w.(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}

// encodeMessage serializes msg using JSON or binary proto based on the
// request Content-Type. For unary calls, the choice matches the request.
// For streaming, the response uses the same family (json or proto) as
// negotiated via the request content type.
func (a *httpStreamAdapter) RecvMsg(m any) error {
	msg, ok := m.(proto.Message)
	if !ok {
		return nil
	}

	ct := a.req.Header.Get("Content-Type")

	if a.streaming {
		return a.recvStreamingMessage(msg, ct)
	}

	return a.recvUnaryMessage(msg, ct)
}

func (a *httpStreamAdapter) sendHeader() {
	a.sendHeaderOnce.Do(func() {
		a.mu.Lock()
		defer a.mu.Unlock()

		ct := a.req.Header.Get("Content-Type")
		switch {
		case a.streaming && isJSONContentType(ct):
			a.w.Header().Set("Content-Type", "application/connect+json")
		case a.streaming:
			a.w.Header().Set("Content-Type", "application/connect+proto")
		case isJSONContentType(ct):
			a.w.Header().Set("Content-Type", "application/json")
		default:
			a.w.Header().Set("Content-Type", "application/proto")
		}

		a.w.WriteHeader(http.StatusOK)
	})
}

func (a *httpStreamAdapter) recvUnaryMessage(msg proto.Message, ct string) error {
	data, err := io.ReadAll(a.req.Body)
	if err != nil {
		return err
	}

	return a.decodeMessage(data, msg, ct)
}

func (a *httpStreamAdapter) recvStreamingMessage(msg proto.Message, ct string) error {
	frame, err := readConnectFrame(a.req.Body)
	if err != nil {
		return err
	}

	if frame.flags&connectEnvelopeFlagEndStream != 0 {
		if len(frame.data) == 0 {
			return io.EOF
		}

		a.endOfStream.Store(true)
	}

	return a.decodeMessage(frame.data, msg, ct)
}

func (a *httpStreamAdapter) decodeMessage(data []byte, msg proto.Message, ct string) error {
	if isJSONContentType(ct) {
		return protojson.Unmarshal(data, msg)
	}

	return proto.Unmarshal(data, msg)
}

func (a *httpStreamAdapter) encodeMessage(msg proto.Message, ct string) ([]byte, error) {
	if isJSONContentType(ct) {
		return protojson.Marshal(msg)
	}

	return proto.Marshal(msg)
}

func (a *httpStreamAdapter) writeError(code codes.Code, msg string) {
	body, _ := json.Marshal(connectError{
		Code:    ErrorCodeToString(code),
		Message: msg,
		Details: []map[string]any{},
	})

	if a.streaming {
		a.sendHeader()

		_ = writeConnectFrame(a.w, body, true)
	} else {
		a.w.Header().Set("Content-Type", "application/connect+json")
		a.w.WriteHeader(ErrorCodeToHTTPStatus(code))
		_, _ = a.w.Write(body)
	}
}

var _ grpc.ServerStream = (*httpStreamAdapter)(nil)

//nolint:cyclop
func ErrorCodeToString(code codes.Code) string {
	switch code {
	case codes.OK:
		return "ok"
	case codes.Canceled:
		return "canceled"
	case codes.Unknown:
		return "unknown"
	case codes.InvalidArgument:
		return "invalid_argument"
	case codes.DeadlineExceeded:
		return "deadline_exceeded"
	case codes.NotFound:
		return "not_found"
	case codes.AlreadyExists:
		return "already_exists"
	case codes.PermissionDenied:
		return "permission_denied"
	case codes.ResourceExhausted:
		return "resource_exhausted"
	case codes.FailedPrecondition:
		return "failed_precondition"
	case codes.Aborted:
		return "aborted"
	case codes.OutOfRange:
		return "out_of_range"
	case codes.Unimplemented:
		return "unimplemented"
	case codes.Internal:
		return "internal"
	case codes.Unavailable:
		return "unavailable"
	case codes.DataLoss:
		return "data_loss"
	case codes.Unauthenticated:
		return "unauthenticated"
	default:
		return "internal"
	}
}

//nolint:cyclop,exhaustive
func ErrorCodeToHTTPStatus(code codes.Code) int {
	switch code {
	case codes.Canceled:
		return http.StatusRequestTimeout
	case codes.Unknown:
		return http.StatusInternalServerError
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.FailedPrecondition:
		return http.StatusBadRequest
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}
