package app

import (
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	"github.com/bavix/gripmock/v3/internal/domain/history"
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
	proxies *proxyroutes.Registry,
	validator *validator.Validate,
	errorFormatter *ErrorFormatter,
) *GRPCWebGateway {
	return &GRPCWebGateway{
		gatewayHandler: newGatewayHandler(budgerigar, descriptorRegistry, recorder, proxies, validator, errorFormatter),
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
			g.handleWithoutDescriptor(w, r, service, method, grpcwebResponse{})

			return
		}

		writeGRPCWebError(w, codes.NotFound, "method not found")

		return
	}

	mocker := g.buildMocker(r, service, method, fullMethod, methodDesc)

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

	resp, err := handleUnaryCore(a.ctx, data, mocker,
		a.req.Header.Get("Content-Type"),
		isGRPCWebJSONContentType,
		a.writeError,
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
	return decodeMessageData(data, msg, ct, isGRPCWebJSONContentType)
}

func (a *grpcwebAdapter) encodeMessage(msg proto.Message, ct string) ([]byte, error) {
	return encodeMessageData(msg, ct, isGRPCWebJSONContentType)
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
