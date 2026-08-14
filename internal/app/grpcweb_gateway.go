package app

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/go-playground/validator/v10"
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
	protosetinfra "github.com/bavix/gripmock/v3/internal/infra/protoset"
	"github.com/bavix/gripmock/v3/internal/infra/proxyroutes"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
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
	gatewayHandler
}

func NewGRPCWebGateway(
	budgerigar *stuber.Budgerigar,
	descriptorRegistry *descriptors.Registry,
	recorder history.Recorder,
	proxyRoutesRef *atomic.Pointer[proxyroutes.Registry],
	validator *validator.Validate,
	errorFormatter *ErrorFormatter,
) *GRPCWebGateway {
	return &GRPCWebGateway{
		gatewayHandler: newGatewayHandler(budgerigar, descriptorRegistry, recorder, proxyRoutesRef, validator, errorFormatter),
	}
}

func (g *GRPCWebGateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	vars := mux.Vars(r)
	service := vars["service"]
	method := vars["method"]
	fullMethod := "/" + service + "/" + method

	zerolog.Ctx(r.Context()).Debug().
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("protocol", "grpc-web").
		Str("service", service).
		Str("method", method).
		Msg("gateway: handling grpc-web request")

	if isReflectionMethod(service, method) {
		g.serveReflection(w, r, service)

		return
	}

	methodDesc, err := findMethodDescriptor(g.descriptors, service, method)
	if err != nil {
		if g.descriptors == nil && g.budgerigar != nil {
			g.handleWithoutDescriptor(w, r, service, method, grpcwebResponse{})

			return
		}

		writeGRPCWebError(w, codes.NotFound, "method not found")

		return
	}

	g.serveMethod(w, r, service, method, fullMethod, methodDesc) //nolint:contextcheck
}

func (g *GRPCWebGateway) serveMethod(
	w http.ResponseWriter, r *http.Request, service, method, fullMethod string,
	methodDesc protoreflect.MethodDescriptor,
) {
	mocker := g.buildMocker(r, service, method, fullMethod, methodDesc)

	if isGRPCWebTextContentType(r.Header.Get(headerContentType)) {
		textWriter := newBase64StreamWriter(w)
		defer func() { _ = textWriter.Close() }()

		w = textWriter
	}

	adapter := newGRPCWebAdapter(r, w, mocker)

	timeout, ok, err := requestTimeout(r.Header)
	if err != nil {
		writeGRPCWebError(w, codes.InvalidArgument, "invalid grpc-timeout")

		return
	}

	if ok && timeout > 0 {
		timedCtx, cancel := context.WithTimeout(adapter.ctx, timeout)
		defer cancel()

		adapter.ctx = timedCtx
	}

	if !mocker.serverStream && !mocker.clientStream {
		g.handleUnary(mocker, adapter)

		return
	}

	if err := mocker.streamHandler(adapter.ctx, adapter); err != nil {
		st, _ := status.FromError(err)
		adapter.writeErrorStatus(normalizeHealthError(st, mocker.serviceName))

		return
	}

	adapter.writeTrailers(codes.OK, "")
}

// gRPC-Web frames every mode identically, so reflection needs no special
// casing beyond the base64 text variant and the usual trailers frame.
func (g *GRPCWebGateway) serveReflection(w http.ResponseWriter, r *http.Request, service string) {
	if isGRPCWebTextContentType(r.Header.Get(headerContentType)) {
		textWriter := newBase64StreamWriter(w)
		defer func() { _ = textWriter.Close() }()

		w = textWriter
	}

	adapter := &grpcwebAdapter{
		baseStreamAdapter: baseStreamAdapter{
			ctx:           httpHeadersToGRPCContext(r.Context(), r.Header),
			req:           r,
			w:             w,
			typeResolver:  g.reflection.resolver,
			frameEncoding: responseFrameEncoding(w, r),
		},
	}

	if err := g.reflection.serve(service, adapter); err != nil && !errors.Is(err, io.EOF) {
		st, _ := status.FromError(err)
		adapter.writeErrorStatus(st)

		return
	}

	adapter.writeTrailers(codes.OK, "")
}

func (g *GRPCWebGateway) handleUnary(mocker *grpcMocker, a *grpcwebAdapter) {
	raw, err := io.ReadAll(a.req.Body)
	if err != nil {
		a.writeError(codes.Internal, "failed to read body")

		return
	}

	if isGRPCWebTextContentType(a.req.Header.Get(headerContentType)) {
		raw, err = decodeGRPCWebText(raw)
		if err != nil {
			a.writeError(codes.InvalidArgument, "malformed grpc-web-text body")

			return
		}
	}

	data, err := extractPayload(raw, a.req.Header)
	if err != nil {
		a.writeError(codes.InvalidArgument, err.Error())

		return
	}

	resp, err := handleUnaryCore(a.ctx, a, data, mocker,
		a.req.Header.Get(headerContentType),
		isGRPCWebJSONContentType,
		func(st *status.Status) {
			a.writeErrorStatus(normalizeHealthError(st, mocker.serviceName))
		},
	)
	if err != nil {
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
func extractPayload(raw []byte, hdr http.Header) ([]byte, error) {
	if len(raw) < ConnectEnvelopeHeaderSize {
		return raw, nil
	}

	declared := binary.BigEndian.Uint32(raw[1:5])
	if int(declared)+ConnectEnvelopeHeaderSize != len(raw) {
		return raw, nil
	}

	switch raw[0] {
	case 0x00: //nolint:mnd
		return raw[ConnectEnvelopeHeaderSize:], nil
	case connectEnvelopeFlagCompressed:
		return decompressFrame(raw[ConnectEnvelopeHeaderSize:], frameEncoding(hdr))
	default:
		return raw, nil
	}
}

// grpcwebResponse implements withoutDescriptorResponse for the gRPC-Web protocol.
type grpcwebResponse struct{}

func (grpcwebResponse) WriteError(w http.ResponseWriter, r *http.Request, code codes.Code, msg string) {
	setGRPCWebContentType(w, r)
	w.WriteHeader(http.StatusOK)
	writeGRPCWebTrailers(w, code, msg)
}

func (grpcwebResponse) WriteSuccess(w http.ResponseWriter, r *http.Request) {
	setGRPCWebContentType(w, r)
	w.WriteHeader(http.StatusOK)
	_ = writeConnectFrame(w, nil, false)
	writeGRPCWebTrailers(w, codes.OK, "")
}

func isGRPCWebJSONContentType(ct string) bool {
	return ct == contentTypeJSON || ct == grpcwebContentTypeJSON || ct == grpcwebContentTypeTextJSON
}

func setGRPCWebContentType(w http.ResponseWriter, r *http.Request) {
	ct := r.Header.Get(headerContentType)

	switch {
	case isGRPCWebTextContentType(ct):
		w.Header().Set(headerContentType, grpcwebTextResponseContentType(ct))
	case isGRPCWebJSONContentType(ct):
		w.Header().Set(headerContentType, grpcwebContentTypeJSON)
	default:
		w.Header().Set(headerContentType, grpcwebContentTypeProto)
	}
}

func writeGRPCWebError(w http.ResponseWriter, code codes.Code, msg string) {
	w.Header().Set(headerContentType, grpcwebContentTypeProto)
	w.WriteHeader(http.StatusOK)
	writeGRPCWebTrailers(w, code, msg)
}

// writeDataFrame writes a gRPC-Web data frame (flag 0x00) to w.
// The data is written as a 5-byte header (flag + big-endian length) followed by payload.
func writeDataFrame(w http.ResponseWriter, data []byte) {
	var header [ConnectEnvelopeHeaderSize]byte

	header[0] = 0x00                                           // data frame flag
	binary.BigEndian.PutUint32(header[1:5], uint32(len(data))) //nolint:gosec
	_, _ = w.Write(header[:])
	_, _ = w.Write(data)
}

// writeGRPCWebTrailers writes a gRPC-Web trailers frame containing
// grpc-status and optionally grpc-message (percent-encoded), plus any
// additional trailer lines from extra.
func writeGRPCWebTrailers(w http.ResponseWriter, code codes.Code, msg string, extra ...string) {
	var data string
	if msg == "" {
		data = fmt.Sprintf("grpc-status: %d\r\n", code)
	} else {
		data = fmt.Sprintf("grpc-status: %d\r\ngrpc-message: %s\r\n",
			code, percentEncode(msg))
	}

	for _, e := range extra {
		data += e + "\r\n" //nolint:perfsprint
	}

	var header [ConnectEnvelopeHeaderSize]byte

	header[0] = grpcwebEnvelopeFlagTrailers
	binary.BigEndian.PutUint32(header[1:5], uint32(len(data))) //nolint:gosec

	if _, err := w.Write(header[:]); err != nil {
		return
	}

	if _, err := io.WriteString(w, data); err != nil {
		return
	}

	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
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

	trailerExtra []string
}

func newGRPCWebAdapter(r *http.Request, w http.ResponseWriter, mocker *grpcMocker) *grpcwebAdapter {
	ctx := httpHeadersToGRPCContext(r.Context(), r.Header)

	return &grpcwebAdapter{
		baseStreamAdapter: baseStreamAdapter{
			ctx:           ctx,
			req:           r,
			w:             w,
			typeResolver:  mocker.typeResolver,
			frameEncoding: responseFrameEncoding(w, r),
		},
	}
}

func (a *grpcwebAdapter) SendMsg(m any) error {
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

	if err := writeConnectFrameEncoded(a.w, data, false, a.frameEncoding); err != nil {
		return err
	}

	if flusher, ok := a.w.(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}

func (a *grpcwebAdapter) RecvMsg(m any) error {
	if a.endOfStream.Load() {
		return io.EOF
	}

	msg, ok := m.(proto.Message)
	if !ok {
		// nil message = end-of-stream check. Body is already consumed
		// after the first read, so signal EOF.
		return io.EOF
	}

	ct := a.req.Header.Get(headerContentType)

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

func (a *grpcwebAdapter) SetTrailer(md metadata.MD) {
	if len(md) == 0 {
		return
	}

	lines := make([]string, 0, len(md))

	for k, values := range md {
		if len(values) == 0 {
			continue
		}

		lines = append(lines, sanitizeTrailerLine(k)+": "+sanitizeTrailerLine(strings.Join(values, ",")))
	}

	a.setTrailerExtra(lines...)
}

func sanitizeTrailerLine(s string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(s)
}

func (a *grpcwebAdapter) sendHeader() {
	a.sendHeaderOnce.Do(func() {
		setGRPCWebContentType(a.w, a.req)

		if a.frameEncoding == encodingGzip {
			a.w.Header().Set(headerGRPCEncoding, encodingGzip)
		}

		a.w.WriteHeader(http.StatusOK)
	})
}

func (a *grpcwebAdapter) decodeMessage(data []byte, msg proto.Message, ct string) error {
	return decodeMessageData(data, msg, ct, isGRPCWebJSONContentType, a.typeResolver)
}

func (a *grpcwebAdapter) encodeMessage(msg proto.Message, ct string) ([]byte, error) {
	return encodeMessageData(msg, ct, isGRPCWebJSONContentType, a.typeResolver)
}

func (a *grpcwebAdapter) setTrailerExtra(lines ...string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.trailerExtra = append(a.trailerExtra, lines...)
}

func (a *grpcwebAdapter) writeError(code codes.Code, msg string) {
	a.sendHeader()

	writeGRPCWebTrailers(a.w, code, msg, a.trailerExtra...)
}

func (a *grpcwebAdapter) writeErrorStatus(st *status.Status) {
	a.sendHeader()

	// gRPC-Web unary errors: write a data frame with the full google.rpc.Status
	// (including @type-annotated details), then a trailers frame.
	statusJSON, _ := protosetinfra.GlobalTypeResolver().Marshal(st.Proto())
	if len(statusJSON) > 0 {
		writeDataFrame(a.w, statusJSON)
	}

	writeGRPCWebTrailers(a.w, st.Code(), st.Message(), a.trailerExtra...)
}

func (a *grpcwebAdapter) writeTrailers(code codes.Code, msg string) {
	writeGRPCWebTrailers(a.w, code, msg, a.trailerExtra...)
}

// Compile-time check that grpcwebAdapter satisfies grpc.ServerStream.
var _ grpc.ServerStream = (*grpcwebAdapter)(nil)
