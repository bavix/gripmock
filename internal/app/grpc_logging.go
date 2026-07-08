package app

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// LogUnaryInterceptor logs unary gRPC calls.
func LogUnaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
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
		event.Interface("grpc.metadata", md)
	}

	if content := protoToJSON(req); content != nil {
		event.RawJSON("grpc.request.content", content)
	}

	if content := protoToJSON(resp); content != nil {
		event.RawJSON("grpc.response.content", content)
	}

	event.Msg("gRPC call completed")

	return resp, err
}

// LogStreamInterceptor logs streaming gRPC calls.
func LogStreamInterceptor(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	start := time.Now()
	grpcPeer, _ := peer.FromContext(stream.Context())
	service, method := splitMethodName(info.FullMethod)

	wrapped := &loggingStream{stream, []any{}, []any{}}
	err := handler(srv, wrapped)

	level := zerolog.InfoLevel
	if service == serviceReflection {
		level = zerolog.DebugLevel
	}

	zerolog.Ctx(stream.Context()).WithLevel(level).
		Str("grpc.component", "server").
		Str("grpc.method", method).
		Str("grpc.method_type", "stream").
		Str("grpc.service", service).
		Str("grpc.code", status.Code(err).String()).
		Dur("grpc.time_ms", time.Since(start)).
		Str("peer.address", getPeerAddress(grpcPeer)).
		Array("grpc.request.content", toLogArray(wrapped.requests...)).
		Array("grpc.response.content", toLogArray(wrapped.responses...)).
		Str("protocol", "grpc").
		Msg("gRPC call completed")

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

	marshaller := protojson.MarshalOptions{
		EmitUnpopulated: false,
		UseProtoNames:   true,
		Indent:          "",
	}

	data, err := marshaller.Marshal(message)
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
	if err := json.Unmarshal(data, &result); err != nil {
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

	requests  []any
	responses []any
}

func (s *loggingStream) SendMsg(m any) error {
	s.appendResponse(m)

	return s.ServerStream.SendMsg(m)
}

func (s *loggingStream) RecvMsg(m any) error {
	s.appendRequest(m)

	return s.ServerStream.RecvMsg(m)
}

func (s *loggingStream) appendRequest(m any) {
	if m == nil || isNilInterface(m) {
		return
	}

	if len(s.requests) < maxLoggingStreamMsgs {
		s.requests = append(s.requests, m)
	}
}

func (s *loggingStream) appendResponse(m any) {
	if m == nil || isNilInterface(m) {
		return
	}

	if len(s.responses) < maxLoggingStreamMsgs {
		s.responses = append(s.responses, m)
	}
}
