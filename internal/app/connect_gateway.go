package app

import (
	"context"
	"io"
	"net/http"
	"strings"
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
		ctx: r.Context(),
		req: r,
		w:   w,
	}

	if mocker.serverStream || mocker.clientStream {
		if err := mocker.streamHandler(adapter.ctx, adapter); err != nil { //nolint:contextcheck
			st, _ := status.FromError(err)
			adapter.writeError(st.Code(), st.Message())
		}
	} else {
		g.handleUnary(mocker, adapter)
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

	_ = a.SendMsg(resp)
}

//nolint:ireturn
func (g *ConnectRPCGateway) findMethodDescriptor(serviceName, methodName string) (protoreflect.MethodDescriptor, error) {
	if method := findMethodInGlobalFiles(serviceName, methodName); method != nil {
		return method, nil
	}

	if g.descriptors == nil {
		return nil, &connectMethodNotFoundError{service: serviceName, method: methodName}
	}

	var found protoreflect.MethodDescriptor

	g.descriptors.RangeFiles(func(file protoreflect.FileDescriptor) bool {
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

	if len(outputToUse.Data) > 0 {
		g.writeError(w, codes.Unimplemented,
			"proto descriptor required to encode non-empty output for "+serviceName+"/"+methodName)

		return
	}

	for k, v := range outputToUse.Headers {
		w.Header().Set(k, v)
	}

	ct := r.Header.Get("Content-Type")
	if isJSONContentType(ct) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	} else {
		w.Header().Set("Content-Type", "application/proto")
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
	body, _ := json.Marshal(map[string]string{
		"code":    ErrorCodeToString(code),
		"message": msg,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(ErrorCodeToHTTPStatus(code))
	_, _ = w.Write(body)
}

func isJSONContentType(ct string) bool {
	return ct == "application/json" || ct == "application/connect+json"
}

type httpStreamAdapter struct {
	req *http.Request
	w   http.ResponseWriter

	sentHeader bool

	ctx context.Context //nolint:containedctx
}

func (a *httpStreamAdapter) Context() context.Context {
	return a.ctx
}

func (a *httpStreamAdapter) SetHeader(md metadata.MD) error {
	if a.sentHeader {
		return nil
	}

	for k, v := range md {
		for _, val := range v {
			a.w.Header().Add(k, val)
		}
	}

	return nil
}

func (a *httpStreamAdapter) SendHeader(md metadata.MD) error {
	a.sentHeader = true

	return a.SetHeader(md)
}

func (a *httpStreamAdapter) SetTrailer(md metadata.MD) {
	_ = a.SetHeader(md)
}

func (a *httpStreamAdapter) SendMsg(m any) error {
	if !a.sentHeader {
		a.w.WriteHeader(http.StatusOK)
		a.sentHeader = true
	}

	msg, ok := m.(proto.Message)
	if !ok {
		return nil
	}

	ct := a.req.Header.Get("Content-Type")

	data, err := protojson.Marshal(msg)
	if err != nil {
		return err
	}

	if !isJSONContentType(ct) {
		data, err = proto.Marshal(msg)
		if err != nil {
			return err
		}
	}

	_, err = a.w.Write(data)
	if err != nil {
		return err
	}

	if flusher, ok := a.w.(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}

func (a *httpStreamAdapter) RecvMsg(m any) error {
	buf := make([]byte, 4096) //nolint:mnd

	var body []byte

	for {
		n, err := a.req.Body.Read(buf)

		if n > 0 {
			body = append(body, buf[:n]...)
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}
	}

	msg, ok := m.(proto.Message)
	if !ok {
		return nil
	}

	ct := a.req.Header.Get("Content-Type")
	if isJSONContentType(ct) {
		return protojson.Unmarshal(body, msg)
	}

	return proto.Unmarshal(body, msg)
}

func (a *httpStreamAdapter) writeError(code codes.Code, msg string) {
	body, _ := json.Marshal(map[string]string{
		"code":    ErrorCodeToString(code),
		"message": msg,
	})

	a.w.Header().Set("Content-Type", "application/json")
	a.w.WriteHeader(ErrorCodeToHTTPStatus(code))
	_, _ = a.w.Write(body)
}

var _ grpc.ServerStream = (*httpStreamAdapter)(nil)
