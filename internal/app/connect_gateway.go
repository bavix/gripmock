package app

import (
	"io"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-json"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/infra/proxyroutes"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

type ConnectRPCGateway struct {
	gatewayHandler
}

func NewConnectRPCGateway(
	budgerigar *stuber.Budgerigar,
	descriptorRegistry *descriptors.Registry,
	recorder history.Recorder,
	proxies *proxyroutes.Registry,
	validator *validator.Validate,
	errorFormatter *ErrorFormatter,
) *ConnectRPCGateway {
	return &ConnectRPCGateway{
		gatewayHandler: newGatewayHandler(budgerigar, descriptorRegistry, recorder, proxies, validator, errorFormatter),
	}
}

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
	logger.Debug().
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("protocol", "connectrpc").
		Str("service", service).
		Str("method", method).
		Msg("gateway: handling connectrpc request")

	methodDesc, err := findMethodDescriptor(g.descriptors, service, method)
	if err != nil {
		if g.descriptors == nil && g.budgerigar != nil {
			g.handleWithoutDescriptor(w, r, service, method, connectResponse{})

			return
		}

		g.writeError(w, codes.NotFound, "method not found")

		return
	}

	mocker := g.buildMocker(r, service, method, fullMethod, methodDesc)

	adapter := &httpStreamAdapter{
		baseStreamAdapter: baseStreamAdapter{
			ctx: r.Context(),
			req: r,
			w:   w,
		},
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

// connectResponse implements withoutDescriptorResponse for the ConnectRPC protocol.
type connectResponse struct{}

func (connectResponse) WriteError(w http.ResponseWriter, r *http.Request, code codes.Code, msg string) {
	body, _ := json.Marshal(connectError{
		Code:    ErrorCodeToString(code),
		Message: msg,
		Details: []map[string]any{},
	})

	w.Header().Set("Content-Type", "application/connect+json")
	w.WriteHeader(ErrorCodeToHTTPStatus(code))
	_, _ = w.Write(body)
}

func (connectResponse) WriteSuccess(w http.ResponseWriter, r *http.Request) {
	ct := r.Header.Get("Content-Type")
	if isJSONContentType(ct) {
		w.Header().Set("Content-Type", "application/connect+json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	} else {
		w.Header().Set("Content-Type", "application/connect+proto")
		w.WriteHeader(http.StatusOK)
	}
}

func isJSONContentType(ct string) bool {
	return ct == "application/json" || ct == "application/connect+json"
}

type httpStreamAdapter struct {
	baseStreamAdapter

	streaming bool
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
	return decodeMessageData(data, msg, ct, isJSONContentType)
}

func (a *httpStreamAdapter) encodeMessage(msg proto.Message, ct string) ([]byte, error) {
	return encodeMessageData(msg, ct, isJSONContentType)
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
