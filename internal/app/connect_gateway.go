package app

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-json"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/infra/proxyroutes"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
)

type ConnectRPCGateway struct {
	gatewayHandler

	requireProtocolVersion bool
}

const (
	headerConnectProtocolVersion = "Connect-Protocol-Version"
	connectProtocolVersion       = "1"
	connectGetVersionValue       = "v1"
)

func NewConnectRPCGateway(
	ctx context.Context,
	budgerigar *stuber.Budgerigar,
	descriptorRegistry *descriptors.Registry,
	recorder history.Recorder,
	proxyRoutesRef *atomic.Pointer[proxyroutes.Registry],
	validator *validator.Validate,
	errorFormatter *ErrorFormatter,
	engines ...*template.Engine,
) *ConnectRPCGateway {
	return &ConnectRPCGateway{
		gatewayHandler: newGatewayHandler(ctx, budgerigar, descriptorRegistry, recorder,
			proxyRoutesRef, validator, errorFormatter, engines...),
	}
}

// isConnectStreamContentType reports whether a streaming Connect request
// carries one of the enveloped codecs. The protocol requires
// application/connect+{codec} for every streaming RPC; the plain
// application/{codec} forms are unary-only.
func acceptableContentType(streaming bool, ct string) bool {
	if streaming {
		return isConnectStreamContentType(ct)
	}

	if ct == "" || isConnectStreamContentType(ct) {
		return true
	}

	switch normalizeContentType(ct) {
	case contentTypeJSON, contentTypeProto:
		return true
	default:
		return false
	}
}

func unsupportedStreamEncoding(header http.Header) (string, bool) {
	encoding := strings.ToLower(strings.TrimSpace(header.Get("Connect-Content-Encoding")))
	if encoding == "" || encoding == "identity" {
		return "", true
	}

	return encoding, false
}

func normalizeContentType(ct string) string {
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}

	return strings.ToLower(strings.TrimSpace(ct))
}

func isConnectStreamContentType(ct string) bool {
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}

	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(ct)), "application/connect+")
}

// RequireProtocolVersion turns on the rejection the protocol permits: "clients
// should send this header. Servers and proxies may reject traffic without this
// header with 400 Bad Request".
func (g *ConnectRPCGateway) RequireProtocolVersion(require bool) {
	g.requireProtocolVersion = require
}

func (g *ConnectRPCGateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !g.acceptRequest(w, r) {
		return
	}

	vars := mux.Vars(r)
	service := vars["service"]
	method := vars["method"]

	zerolog.Ctx(r.Context()).Debug().
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("protocol", "connectrpc").
		Str("service", service).
		Str("method", method).
		Msg("gateway: handling connectrpc request")

	if isReflectionMethod(service, method) {
		g.serveReflection(w, r, service)

		return
	}

	methodDesc, ok := g.resolveMethod(w, r, service, method)
	if !ok {
		return
	}

	g.serveMethod(w, r, service, method, methodDesc)
}

func (g *ConnectRPCGateway) acceptRequest(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return false
	}

	if g.requireProtocolVersion && !g.hasProtocolVersion(r) {
		g.writeError(w, codes.InvalidArgument,
			`missing required header: set Connect-Protocol-Version to "1"`)

		return false
	}

	return true
}

// resolveMethod finds the descriptor and, for GET, checks the method is
// side-effect free and rewrites the content type from the query encoding.
//
//nolint:ireturn
func (g *ConnectRPCGateway) resolveMethod(
	w http.ResponseWriter, r *http.Request, service, method string,
) (protoreflect.MethodDescriptor, bool) {
	methodDesc, err := findMethodDescriptor(g.descriptors, service, method)
	if err != nil {
		if g.descriptors == nil && g.budgerigar != nil {
			g.handleWithoutDescriptor(w, r, service, method, connectResponse{})

			return nil, false
		}

		g.writeError(w, codes.NotFound, "method not found")

		return nil, false
	}

	if r.Method == http.MethodGet {
		if !methodAllowsGET(methodDesc) {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

			return nil, false
		}

		r.Header.Set(headerContentType, connectGetContentType(r))
	}

	return methodDesc, true
}

func (g *ConnectRPCGateway) acceptRequestEncoding(
	w http.ResponseWriter, r *http.Request, adapter *httpStreamAdapter,
) bool {
	if !acceptableContentType(adapter.streaming, r.Header.Get(headerContentType)) {
		http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)

		return false
	}

	if !adapter.streaming {
		return true
	}

	if encoding, ok := unsupportedStreamEncoding(r.Header); !ok {
		adapter.writeError(codes.Unimplemented,
			"stream compression "+encoding+" is not supported; send identity")

		return false
	}

	return true
}

func (g *ConnectRPCGateway) serveMethod(
	w http.ResponseWriter, r *http.Request, service, method string, methodDesc protoreflect.MethodDescriptor,
) {
	mocker := g.buildMocker(r, service, method, "/"+service+"/"+method, methodDesc)

	adapter := &httpStreamAdapter{
		baseStreamAdapter: baseStreamAdapter{
			ctx:           httpHeadersToGRPCContext(r.Context(), r.Header),
			req:           r,
			w:             w,
			typeResolver:  mocker.typeResolver,
			frameEncoding: responseFrameEncoding(w, r),
		},
		streaming: mocker.serverStream || mocker.clientStream,
	}

	if !g.acceptRequestEncoding(w, r, adapter) {
		return
	}

	timeout, ok, err := requestTimeout(r.Header)
	if err != nil {
		g.writeError(w, codes.InvalidArgument, "invalid connect-timeout-ms")

		return
	}

	if ok && timeout > 0 {
		timedCtx, cancel := context.WithTimeout(adapter.ctx, timeout)
		defer cancel()

		adapter.ctx = timedCtx
	}

	if !adapter.streaming {
		g.handleUnary(mocker, adapter, methodDesc)

		return
	}

	if err := mocker.streamHandler(adapter.ctx, adapter); err != nil { //nolint:contextcheck
		st, _ := status.FromError(err)
		adapter.writeErrorStatus(normalizeHealthError(st, service))

		return
	}

	adapter.sendHeader()

	body, _ := json.Marshal(connectEndStream{Metadata: adapter.takeTrailerMetadata()})
	if err := writeConnectFrameEncoded(adapter.w, body, true, adapter.frameEncoding); err != nil {
		zerolog.Ctx(r.Context()).Debug().Err(err).Msg("connect.gateway: send end stream")
	}
}

func (g *ConnectRPCGateway) hasProtocolVersion(r *http.Request) bool {
	if r.Method == http.MethodGet {
		return r.URL.Query().Get(connectQueryVersion) == connectGetVersionValue
	}

	return r.Header.Get(headerConnectProtocolVersion) == connectProtocolVersion
}

// ServerReflectionInfo is bidi-streaming, so the Connect protocol requires the
// enveloped application/connect+{codec} form and answers with 200 plus a final
// EndStreamResponse frame -- the same shape as any other streaming method.
func (g *ConnectRPCGateway) serveReflection(w http.ResponseWriter, r *http.Request, service string) {
	if !isConnectStreamContentType(r.Header.Get(headerContentType)) {
		http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)

		return
	}

	adapter := &httpStreamAdapter{
		baseStreamAdapter: baseStreamAdapter{
			ctx:           httpHeadersToGRPCContext(r.Context(), r.Header),
			req:           r,
			w:             w,
			typeResolver:  g.reflection.resolver,
			frameEncoding: responseFrameEncoding(w, r),
		},
		streaming: true,
	}

	if err := g.reflection.serve(service, adapter); err != nil && !errors.Is(err, io.EOF) {
		st, _ := status.FromError(err)
		adapter.writeErrorStatus(st)

		return
	}

	adapter.sendHeader()

	body, _ := json.Marshal(connectEndStream{Metadata: adapter.takeTrailerMetadata()})
	if err := writeConnectFrameEncoded(adapter.w, body, true, adapter.frameEncoding); err != nil {
		zerolog.Ctx(r.Context()).Debug().Err(err).Msg("connect.gateway: send reflection end stream")
	}
}

func (g *ConnectRPCGateway) handleUnary(mocker *grpcMocker, a *httpStreamAdapter, methodDesc protoreflect.MethodDescriptor) {
	body, contentType, err := g.unaryRequestBody(a, methodDesc)
	if err != nil {
		return
	}

	resp, err := handleUnaryCore(a.ctx, a, body, mocker,
		contentType,
		isJSONContentType,
		func(st *status.Status) {
			a.writeErrorStatus(normalizeHealthError(st, mocker.serviceName))
		},
	)
	if err != nil {
		return
	}

	if err := a.SendMsg(resp); err != nil {
		zerolog.Ctx(a.ctx).Debug().Err(err).Msg("connect.gateway: send unary response")
	}
}

func (g *ConnectRPCGateway) unaryRequestBody(a *httpStreamAdapter, methodDesc protoreflect.MethodDescriptor) ([]byte, string, error) {
	if a.req.Method == http.MethodGet {
		body, err := connectGetRequest(a.req, methodDesc)
		if err != nil {
			a.writeError(codes.InvalidArgument, err.Error())

			return nil, "", err
		}

		return body, a.req.Header.Get(headerContentType), nil
	}

	body, err := io.ReadAll(a.req.Body)
	if err != nil {
		a.writeError(codes.Internal, "failed to read body")

		return nil, "", err
	}

	return body, a.req.Header.Get(headerContentType), nil
}

func (g *ConnectRPCGateway) writeError(w http.ResponseWriter, code codes.Code, msg string) {
	body, _ := json.Marshal(connectError{
		Code:    ErrorCodeToString(code),
		Message: msg,
	})

	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(ErrorCodeToHTTPStatus(code))
	_, _ = w.Write(body)
}

// connectResponse implements withoutDescriptorResponse for the ConnectRPC protocol.
type connectResponse struct{}

func (connectResponse) WriteError(w http.ResponseWriter, r *http.Request, code codes.Code, msg string) {
	body, _ := json.Marshal(connectError{
		Code:    ErrorCodeToString(code),
		Message: msg,
	})

	// The protocol pins error bodies to application/json whatever the request
	// asked for, so a client that cannot read the codec can still read the error.
	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(ErrorCodeToHTTPStatus(code))
	_, _ = w.Write(body)
}

func (connectResponse) WriteSuccess(w http.ResponseWriter, r *http.Request) {
	ct := r.Header.Get(headerContentType)
	if isJSONContentType(ct) {
		w.Header().Set(headerContentType, contentTypeConnectJSON)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	} else {
		w.Header().Set(headerContentType, contentTypeConnectProto)
		w.WriteHeader(http.StatusOK)
	}
}

func isJSONContentType(ct string) bool {
	switch normalizeContentType(ct) {
	case contentTypeJSON, contentTypeConnectJSON:
		return true
	default:
		return false
	}
}

type httpStreamAdapter struct {
	baseStreamAdapter

	streaming bool
	trailerMD metadata.MD
}

func (a *httpStreamAdapter) SetTrailer(md metadata.MD) {
	if len(md) == 0 {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.streaming {
		a.trailerMD = metadata.Join(a.trailerMD, md)

		return
	}

	for k, values := range md {
		key := k
		if !strings.HasPrefix(strings.ToLower(key), "trailer-") {
			key = "Trailer-" + key
		}

		for _, v := range values {
			a.w.Header().Add(key, v)
		}
	}
}

func (a *httpStreamAdapter) SendMsg(m any) error {
	a.sendHeader()

	msg, ok := m.(proto.Message)
	if !ok {
		return nil
	}

	ct := a.req.Header.Get(headerContentType)

	data, err := a.encodeMessage(msg, ct)
	if err != nil {
		return err
	}

	if a.streaming {
		if err := writeConnectFrameEncoded(a.w, data, false, a.frameEncoding); err != nil {
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
	// If the peer already signalled end-of-stream (via an end-stream
	// envelope or a single plain-body read), return EOF immediately.
	if a.endOfStream.Load() {
		return io.EOF
	}

	msg, ok := m.(proto.Message)
	if !ok {
		// nil message = end-of-stream check. The body is consumed
		// after the first read, so signal EOF.
		return io.EOF
	}

	ct := a.req.Header.Get(headerContentType)

	if a.streaming {
		return a.recvStreamingMessage(msg, ct)
	}

	return a.recvUnaryMessage(msg, ct)
}

func (a *httpStreamAdapter) takeTrailerMetadata() map[string][]string {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.trailerMD) == 0 {
		return nil
	}

	return a.trailerMD
}

func (a *httpStreamAdapter) sendHeader() {
	a.sendHeaderOnce.Do(func() {
		a.mu.Lock()
		defer a.mu.Unlock()

		ct := a.req.Header.Get(headerContentType)
		switch {
		case a.streaming && isJSONContentType(ct):
			a.w.Header().Set(headerContentType, contentTypeConnectJSON)
		case a.streaming:
			a.w.Header().Set(headerContentType, contentTypeConnectProto)
		case isJSONContentType(ct):
			a.w.Header().Set(headerContentType, contentTypeJSON)
		default:
			a.w.Header().Set(headerContentType, contentTypeProto)
		}

		if a.streaming && a.frameEncoding == encodingGzip {
			a.w.Header().Set(headerConnectContentEncoding, encodingGzip)
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
	// Plain application/json (without connect+ envelope) on a streaming
	// endpoint: treat the entire body as a single stream message.
	// This matches gRPC-Web behaviour and improves interop with clients
	// that do not frame every message when they only send one.
	if unaryContentType := normalizeContentType(ct); unaryContentType == contentTypeJSON ||
		unaryContentType == contentTypeProto {
		data, err := io.ReadAll(a.req.Body)
		if err != nil {
			return err
		}

		if len(data) == 0 {
			return io.EOF
		}

		a.endOfStream.Store(true)

		return a.decodeMessage(data, msg, ct)
	}

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

	payload, err := decodeFramePayload(frame.flags, frame.data, a.req.Header)
	if err != nil {
		return err
	}

	return a.decodeMessage(payload, msg, ct)
}

func (a *httpStreamAdapter) decodeMessage(data []byte, msg proto.Message, ct string) error {
	return decodeMessageData(data, msg, ct, isJSONContentType, a.typeResolver)
}

func (a *httpStreamAdapter) encodeMessage(msg proto.Message, ct string) ([]byte, error) {
	return encodeMessageData(msg, ct, isJSONContentType, a.typeResolver)
}

func (a *httpStreamAdapter) writeError(code codes.Code, msg string) {
	body, _ := json.Marshal(connectError{
		Code:    ErrorCodeToString(code),
		Message: msg,
	})
	a.writeBody(code, body)
}

func (a *httpStreamAdapter) writeErrorStatus(st *status.Status) {
	connErr := serializeErrorStatus(st)

	if !a.streaming {
		body, _ := json.Marshal(connErr)
		a.writeBody(st.Code(), body)

		return
	}

	body, _ := json.Marshal(connectEndStream{Error: &connErr, Metadata: a.takeTrailerMetadata()})
	a.writeBody(st.Code(), body)
}

func (a *httpStreamAdapter) writeBody(code codes.Code, body []byte) {
	if a.streaming {
		a.sendHeader()

		_ = writeConnectFrameEncoded(a.w, body, true, a.frameEncoding)
	} else {
		a.w.Header().Set(headerContentType, contentTypeJSON)
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

const statusClientClosedRequest = 499

//nolint:cyclop,exhaustive
func ErrorCodeToHTTPStatus(code codes.Code) int {
	switch code {
	case codes.Canceled:
		return statusClientClosedRequest
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
