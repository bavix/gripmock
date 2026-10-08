package app

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	protosetinfra "github.com/bavix/gripmock/v3/internal/infra/protoset"
)

// logUnaryInterceptor logs unary gRPC calls.
func (s *GRPCServer) logUnaryInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	return logUnaryCall(ctx, req, info, handler, s.logOptions)
}

func logUnaryCall(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
	opts LogOptions,
) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)

	grpcPeer, _ := peer.FromContext(ctx)
	service, method := splitMethodName(info.FullMethod)

	level := zerolog.InfoLevel
	if service == serviceReflection {
		level = zerolog.DebugLevel
	}

	event := zerolog.Ctx(ctx).WithLevel(level).
		Str("grpc.component", "server").
		Str("grpc.method", method).
		Str("grpc.method_type", "unary").
		Str("grpc.service", service).
		Str("grpc.code", status.Code(err).String()).
		Dur("grpc.time_ms", time.Since(start)).
		Str("peer.address", getPeerAddress(grpcPeer)).
		Str("protocol", "grpc")

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		event.Interface("grpc.metadata", opts.redactMetadata(md))
	}

	opts.logMessageContent(event, "grpc.request.content", req)
	opts.logMessageContent(event, "grpc.response.content", resp)

	event.Msg("gRPC call completed")

	return resp, err
}

// logStreamInterceptor logs streaming gRPC calls.
func (s *GRPCServer) logStreamInterceptor(
	srv any,
	stream grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	return logStreamCall(stream.Context(), srv, stream, info, handler, s.logOptions)
}

func logStreamCall(
	ctx context.Context,
	srv any,
	stream grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
	opts LogOptions,
) error {
	start := time.Now()
	grpcPeer, _ := peer.FromContext(ctx)
	service, method := splitMethodName(info.FullMethod)

	wrapped := newLoggingStream(stream)
	err := handler(srv, wrapped)

	requests, responses := wrapped.snapshot()

	level := zerolog.InfoLevel
	if service == serviceReflection {
		level = zerolog.DebugLevel
	}

	event := zerolog.Ctx(ctx).WithLevel(level).
		Str("grpc.component", "server").
		Str("grpc.method", method).
		Str("grpc.method_type", "stream").
		Str("grpc.service", service).
		Str("grpc.code", status.Code(err).String()).
		Dur("grpc.time_ms", time.Since(start)).
		Str("peer.address", getPeerAddress(grpcPeer)).
		Str("protocol", "grpc")

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		event.Interface("grpc.metadata", opts.redactMetadata(md))
	}

	if opts.MessageContent {
		event.Array("grpc.request.content", toLogArray(requests...)).
			Array("grpc.response.content", toLogArray(responses...))
	}

	event.Msg("gRPC call completed")

	return err
}

func splitMethodName(fullMethod string) (string, string) {
	const (
		slash = "/"
	)

	parts := strings.Split(fullMethod, slash)
	if len(parts) != 3 { //nolint:mnd
		return unknownValue, unknownValue
	}

	return parts[1], parts[2]
}

func getPeerAddress(p *peer.Peer) string {
	if p != nil && p.Addr != nil {
		return p.Addr.String()
	}

	return unknownValue
}

func protoToJSON(msg any) []byte {
	if msg == nil || isNilInterface(msg) {
		return nil
	}

	message, ok := msg.(proto.Message)
	if !ok || message == nil {
		return nil
	}

	data, err := protosetinfra.GlobalTypeResolver().MarshalProtoNames(message)
	if err != nil {
		return nil
	}

	return data
}

func protoToMap(msg any) map[string]any {
	data := protoToJSON(msg)
	if data == nil {
		return nil
	}

	var result map[string]any

	err := json.Unmarshal(data, &result)
	if err != nil {
		return nil
	}

	return result
}

func isNilInterface(v any) bool {
	if v == nil {
		return true
	}

	rv := reflect.ValueOf(v)
	//nolint:exhaustive
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return rv.IsNil()
	default:
		return false
	}
}

func toLogArray(items ...any) *zerolog.Array {
	arr := zerolog.Arr()

	for _, item := range items {
		if item == nil || isNilInterface(item) {
			continue
		}

		if value := protoToJSON(item); value != nil {
			arr = arr.RawJSON(value)
		} else {
			arr = arr.Str(fmt.Sprintf("%v", item))
		}
	}

	return arr
}

type loggingStream struct {
	grpc.ServerStream

	mu        sync.Mutex
	requests  []any
	responses []any
}

func newLoggingStream(stream grpc.ServerStream) *loggingStream {
	return &loggingStream{
		ServerStream: stream,
		requests:     []any{},
		responses:    []any{},
	}
}

func (s *loggingStream) SendMsg(m any) error {
	s.appendResponse(m)

	return s.ServerStream.SendMsg(m)
}

func (s *loggingStream) RecvMsg(m any) error {
	s.appendRequest(m)

	return s.ServerStream.RecvMsg(m)
}

func (s *loggingStream) snapshot() ([]any, []any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Clone(s.requests), slices.Clone(s.responses)
}

func (s *loggingStream) appendRequest(m any) {
	if m == nil || isNilInterface(m) {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.requests) < maxLoggingStreamMsgs {
		s.requests = append(s.requests, m)
	}
}

func (s *loggingStream) appendResponse(m any) {
	if m == nil || isNilInterface(m) {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.responses) < maxLoggingStreamMsgs {
		s.responses = append(s.responses, m)
	}
}

const redactedValue = "[REDACTED]"

// DefaultRedactKeys lists the metadata keys hidden from the call log unless the
// configuration replaces them.
func DefaultRedactKeys() []string {
	return []string{
		"authorization",
		"proxy-authorization",
		"cookie",
		"set-cookie",
		"x-api-key",
		"api-key",
		"x-auth-token",
	}
}

// LogOptions controls how much of a call reaches the log. Build it with
// DefaultLogOptions: the zero value neither redacts metadata nor logs bodies.
type LogOptions struct {
	RedactMetadata bool
	RedactKeys     []string
	MessageContent bool
}

// DefaultLogOptions redacts the well-known credential headers and logs bodies.
func DefaultLogOptions() LogOptions {
	return LogOptions{RedactMetadata: true, RedactKeys: DefaultRedactKeys(), MessageContent: true}
}

func (o LogOptions) redactMetadata(md metadata.MD) metadata.MD {
	if !o.RedactMetadata {
		return md
	}

	keys := o.RedactKeys
	if keys == nil {
		keys = DefaultRedactKeys()
	}

	sensitive := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		sensitive[strings.ToLower(strings.TrimSpace(key))] = struct{}{}
	}

	redacted := make(metadata.MD, len(md))

	for key, values := range md {
		if _, hide := sensitive[strings.ToLower(key)]; hide {
			redacted[key] = []string{redactedValue}

			continue
		}

		redacted[key] = values
	}

	return redacted
}

func (o LogOptions) logMessageContent(event *zerolog.Event, key string, msg any) {
	if !o.MessageContent {
		return
	}

	content := protoToJSON(msg)
	if content == nil {
		return
	}

	if len(content) > maxLoggedBodyBytes {
		event.Str(key, fmt.Sprintf("[truncated: %d bytes]", len(content)))

		return
	}

	event.RawJSON(key, content)
}
