package app

import (
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/infra/proxyroutes"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
)

const (
	grpcwebContentTypeProto = "application/grpc-web+proto"
	grpcwebContentTypeJSON  = "application/grpc-web+json"

	// gRPC-Web uses bit 7 for the trailers flag.
	grpcwebEnvelopeFlagTrailers = 0b10000000
)

// GRPCWebGateway proxies gRPC-Web HTTP requests to the gRPC mocker.
// It translates between gRPC-Web framing (length-prefixed messages +
// trailers with grpc-status/grpc-message) and the shared mocker.
type GRPCWebGateway struct {
	budgerigar     *stuber.Budgerigar
	descriptors    *descriptors.Registry
	recorder       history.Recorder
	proxies        *proxyroutes.Registry
	validator      *validator.Validate
	errorFormatter *ErrorFormatter
}

func NewGRPCWebGateway(
	budgerigar *stuber.Budgerigar,
	descriptorRegistry *descriptors.Registry,
	recorder history.Recorder,
	proxies *proxyroutes.Registry,
	validator *validator.Validate,
	errorFormatter *ErrorFormatter,
) *GRPCWebGateway {
	e := errorFormatter
	if e == nil {
		e = NewErrorFormatter()
	}

	return &GRPCWebGateway{
		budgerigar:     budgerigar,
		descriptors:    descriptorRegistry,
		recorder:       recorder,
		proxies:        proxies,
		validator:      validator,
		errorFormatter: e,
	}
}

//nolint:funlen
func (g *GRPCWebGateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
		Str("protocol", "grpc-web").
		Str("service", service).
		Str("method", method).
		Msg("gateway: handling grpc-web request")

	methodDesc, err := findMethodDescriptor(g.descriptors, service, method)
	if err != nil {
		if g.descriptors == nil && g.budgerigar != nil {
			g.handleWithoutDescriptor(w, r, service, method)

			return
		}

		writeGRPCWebError(w, codes.NotFound, "method not found")

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

	adapter := newGRPCWebAdapter(r, w, mocker)

	if !mocker.serverStream && !mocker.clientStream {
		g.handleUnary(mocker, adapter)

		return
	}

	if err := mocker.streamHandler(adapter.ctx, adapter); err != nil { //nolint:contextcheck
		st, _ := status.FromError(err)
		adapter.writeError(st.Code(), st.Message())
	} else {
		adapter.writeTrailers(codes.OK, "")
	}
}

func (g *GRPCWebGateway) handleUnary(mocker *grpcMocker, a *grpcwebAdapter) {
	raw, err := io.ReadAll(a.req.Body)
	if err != nil {
		a.writeError(codes.Internal, "failed to read body")

		return
	}

	data, err := extractPayload(raw)
	if err != nil {
		a.writeError(codes.InvalidArgument, err.Error())

		return
	}

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)
	if isGRPCWebJSONContentType(a.req.Header.Get("Content-Type")) {
		if err := protojson.Unmarshal(data, inputMsg); err != nil {
			a.writeError(codes.InvalidArgument, "failed to unmarshal: "+err.Error())

			return
		}
	} else {
		if err := proto.Unmarshal(data, inputMsg); err != nil {
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
		zerolog.Ctx(a.ctx).Debug().Err(err).Msg("grpcweb.gateway: send unary response")

		return
	}

	a.writeTrailers(codes.OK, "")
}

// extractPayload strips the gRPC-Web length-prefixed frame header
// (5-byte envelope) when present. Strict gRPC-Web clients always frame
// messages; simpler tools may send raw protobuf/JSON bytes.
//
//   - flag 0x00 (uncompressed data): header stripped, payload returned
//   - flag 0x01 (compressed data):   clear error — not supported
//   - no valid frame detected:       raw body returned as-is
func extractPayload(raw []byte) ([]byte, error) {
	if len(raw) < connectEnvelopeHeaderSize {
		return raw, nil
	}

	declared := binary.BigEndian.Uint32(raw[1:5])
	if int(declared)+connectEnvelopeHeaderSize != len(raw) {
		return raw, nil
	}

	switch raw[0] {
	case 0x00: //nolint:mnd
		return raw[connectEnvelopeHeaderSize:], nil
	case 0x01: //nolint:mnd
		return nil, status.Error(codes.Unimplemented,
			"grpc frame compression (flag 0x01) is not supported; use Content-Encoding: gzip on the HTTP body instead")
	default:
		return raw, nil
	}
}

// handleWithoutDescriptor matches stubs by service/method name alone,
// without requiring proto descriptors. Results are returned as empty
// length-prefixed frames with gRPC-Web trailers.
//
//nolint:funlen
func (g *GRPCWebGateway) handleWithoutDescriptor(w http.ResponseWriter, r *http.Request, serviceName, methodName string) {
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
		recordCall(g.recorder, serviceName, methodName, query.Session, uuid.Nil, uint32(codes.NotFound),
			requestTime, []map[string]any{emptyInput}, nil, notFoundMsg)
		setGRPCWebContentType(w, r)
		w.WriteHeader(http.StatusOK)
		writeGRPCWebTrailers(w, codes.NotFound, notFoundMsg)

		return
	}

	found := result.Found()

	if err := delayResponse(r.Context(), found.Output.Delay); err != nil {
		st, _ := status.FromError(err)
		recordCall(g.recorder, serviceName, methodName, query.Session, found.ID, uint32(st.Code()),
			requestTime, []map[string]any{emptyInput}, nil, st.Message())
		setGRPCWebContentType(w, r)
		w.WriteHeader(http.StatusOK)
		writeGRPCWebTrailers(w, st.Code(), st.Message())

		return
	}

	outputToUse := found.Output

	if st := outputStatusBase(outputToUse); st != nil {
		recordCall(g.recorder, serviceName, methodName, query.Session, found.ID, uint32(st.Code()),
			requestTime, []map[string]any{emptyInput}, nil, st.Message())
		setGRPCWebContentType(w, r)
		w.WriteHeader(http.StatusOK)
		writeGRPCWebTrailers(w, st.Code(), st.Message())

		return
	}

	if outputToUse.Data != nil {
		recordCall(g.recorder, serviceName, methodName, query.Session, found.ID, uint32(codes.Unimplemented),
			requestTime, []map[string]any{emptyInput}, nil,
			"proto descriptor required to encode non-empty output for "+serviceName+"/"+methodName)
		setGRPCWebContentType(w, r)
		w.WriteHeader(http.StatusOK)
		writeGRPCWebTrailers(w, codes.Unimplemented,
			"proto descriptor required to encode non-empty output for "+serviceName+"/"+methodName)

		return
	}

	for k, v := range outputToUse.Headers {
		w.Header().Set(k, v)
	}

	setGRPCWebContentType(w, r)
	w.WriteHeader(http.StatusOK)

	_ = writeConnectFrame(w, nil, false)
	writeGRPCWebTrailers(w, codes.OK, "")

	recordCall(g.recorder, serviceName, methodName, query.Session, found.ID, uint32(codes.OK),
		requestTime, []map[string]any{emptyInput}, []map[string]any{{}}, "")
}

func isGRPCWebJSONContentType(ct string) bool {
	return ct == "application/json" || ct == grpcwebContentTypeJSON
}

func setGRPCWebContentType(w http.ResponseWriter, r *http.Request) {
	if isGRPCWebJSONContentType(r.Header.Get("Content-Type")) {
		w.Header().Set("Content-Type", grpcwebContentTypeJSON)
	} else {
		w.Header().Set("Content-Type", grpcwebContentTypeProto)
	}
}

func writeGRPCWebError(w http.ResponseWriter, code codes.Code, msg string) {
	w.Header().Set("Content-Type", grpcwebContentTypeProto)
	w.WriteHeader(http.StatusOK)
	writeGRPCWebTrailers(w, code, msg)
}

// writeGRPCWebTrailers writes a gRPC-Web trailers frame containing
// grpc-status and optionally grpc-message (percent-encoded).
// This frame uses the gRPC-Web trailers flag (0x80) which is distinct
// from the Connect end-stream flag (0x02).
func writeGRPCWebTrailers(w http.ResponseWriter, code codes.Code, msg string) {
	var data string
	if msg == "" {
		data = fmt.Sprintf("grpc-status: %d\r\n", code)
	} else {
		data = fmt.Sprintf("grpc-status: %d\r\ngrpc-message: %s\r\n",
			code, percentEncode(msg))
	}

	var header [connectEnvelopeHeaderSize]byte

	header[0] = grpcwebEnvelopeFlagTrailers
	binary.BigEndian.PutUint32(header[1:5], uint32(len(data))) //nolint:gosec

	if _, err := w.Write(header[:]); err != nil {
		return
	}

	if _, err := io.WriteString(w, data); err != nil {
		return
	}
}

// percentEncode encodes s per RFC 3986 Section 2.1 for use in
// grpc-message trailer values. Spaces become %20 (not +).
func percentEncode(s string) string {
	var buf strings.Builder

	for _, b := range []byte(s) {
		if shouldEscape(b) {
			fmt.Fprintf(&buf, "%%%02X", b)
		} else {
			buf.WriteByte(b)
		}
	}

	return buf.String()
}

func shouldEscape(b byte) bool {
	return b <= 0x20 || b > 0x7E || b == '%'
}

// grpcwebAdapter implements grpc.ServerStream and translates between
// gRPC-Web framing and the in-process mocker. Outgoing messages are
// written as length-prefixed frames; the caller must finish with a
// trailers frame via writeTrailers or writeError.
type grpcwebAdapter struct {
	baseStreamAdapter
}

func newGRPCWebAdapter(r *http.Request, w http.ResponseWriter, _ *grpcMocker) *grpcwebAdapter {
	ctx := httpHeadersToGRPCContext(r.Context(), r.Header)

	return &grpcwebAdapter{
		baseStreamAdapter: baseStreamAdapter{
			ctx: ctx,
			req: r,
			w:   w,
		},
	}
}

func (a *grpcwebAdapter) SendMsg(m any) error {
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

	if err := writeConnectFrame(a.w, data, false); err != nil {
		return err
	}

	if flusher, ok := a.w.(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}

func (a *grpcwebAdapter) RecvMsg(m any) error {
	msg, ok := m.(proto.Message)
	if !ok {
		return nil
	}

	ct := a.req.Header.Get("Content-Type")

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

func (a *grpcwebAdapter) sendHeader() {
	a.sendHeaderOnce.Do(func() {
		setGRPCWebContentType(a.w, a.req)
		a.w.WriteHeader(http.StatusOK)
	})
}

func (a *grpcwebAdapter) decodeMessage(data []byte, msg proto.Message, ct string) error {
	if isGRPCWebJSONContentType(ct) {
		return protojson.Unmarshal(data, msg)
	}

	return proto.Unmarshal(data, msg)
}

func (a *grpcwebAdapter) encodeMessage(msg proto.Message, ct string) ([]byte, error) {
	if isGRPCWebJSONContentType(ct) {
		return protojson.Marshal(msg)
	}

	return proto.Marshal(msg)
}

func (a *grpcwebAdapter) writeError(code codes.Code, msg string) {
	a.sendHeader()

	writeGRPCWebTrailers(a.w, code, msg)
}

func (a *grpcwebAdapter) writeTrailers(code codes.Code, msg string) {
	writeGRPCWebTrailers(a.w, code, msg)
}

// Compile-time check that grpcwebAdapter satisfies grpc.ServerStream.
var _ grpc.ServerStream = (*grpcwebAdapter)(nil)
