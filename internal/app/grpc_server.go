package app

//nolint:revive
import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/gripmock/stuber"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"

	protoloc "github.com/bavix/gripmock/internal/domain/proto"
	"github.com/bavix/gripmock/internal/domain/protoset"
	"github.com/bavix/gripmock/pkg/grpccontext"
)

const serviceReflection = "grpc.reflection.v1.ServerReflection"

type GRPCServer struct {
	network    string
	address    string
	params     *protoloc.Arguments
	budgerigar *stuber.Budgerigar
	waiter     Extender
}

type grpcMocker struct {
	budgerigar *stuber.Budgerigar

	inputDesc  protoreflect.MessageDescriptor
	outputDesc protoreflect.MessageDescriptor

	packageName string
	serviceName string
	methodName  string
	fullMethod  string

	serverStream bool
	clientStream bool
}

func (m *grpcMocker) streamHandler(srv any, stream grpc.ServerStream) error {
	info := &grpc.StreamServerInfo{
		FullMethod:     m.fullMethod,
		IsClientStream: m.clientStream,
		IsServerStream: m.serverStream,
	}

	handler := func(_ any, stream grpc.ServerStream) error {
		switch {
		case m.serverStream && !m.clientStream:
			return m.handleServerStream(stream)
		case !m.serverStream && m.clientStream:
			return m.handleClientStream(stream)
		case m.serverStream && m.clientStream:
			return m.handleBidiStream(stream)
		default:
			return status.Errorf(codes.Unimplemented, "Unknown stream type")
		}
	}

	return grpc.StreamServerInterceptor(func(srv any, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, ss)
	})(srv, stream, info, handler)
}

func (m *grpcMocker) newQuery(ctx context.Context, msg *dynamicpb.Message) stuber.Query {
	query := stuber.Query{
		Service: m.serviceName,
		Method:  m.methodName,
		Data:    convertToMap(msg),
	}

	excludes := []string{":authority", "content-type", "grpc-accept-encoding", "user-agent", "accept-encoding"}

	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		query.Headers = make(map[string]any, len(md))

		for key, values := range md {
			if slices.Contains(excludes, key) {
				continue
			}

			query.Headers[key] = strings.Join(values, ";")
		}
	}

	zerolog.Ctx(ctx).Debug().Interface("data", query).Msg("New query")

	return query
}

func convertToMap(msg proto.Message) map[string]any {
	if msg == nil {
		return nil
	}

	result := make(map[string]any)
	message := msg.ProtoReflect()

	message.Range(func(fd protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if !message.Has(fd) {
			return true
		}

		fieldName := string(fd.Name())
		result[fieldName] = convertValue(fd, value)

		return true
	})

	return result
}

func convertValue(fd protoreflect.FieldDescriptor, value protoreflect.Value) any {
	switch {
	case fd.IsList():
		return convertList(fd, value.List())
	case fd.IsMap():
		return convertMap(fd, value.Map())
	default:
		return convertScalar(fd, value)
	}
}

func convertList(fd protoreflect.FieldDescriptor, list protoreflect.List) []any {
	result := make([]any, list.Len())
	elemType := fd.Message()

	for i := range list.Len() {
		elem := list.Get(i)

		if elemType != nil {
			result[i] = convertToMap(elem.Message().Interface())
		} else {
			result[i] = convertScalar(fd, elem)
		}
	}

	return result
}

func convertMap(fd protoreflect.FieldDescriptor, m protoreflect.Map) map[string]any {
	result := make(map[string]any)
	keyType := fd.MapKey()
	valType := fd.MapValue().Message()

	m.Range(func(key protoreflect.MapKey, val protoreflect.Value) bool {
		convertedKey, ok := convertScalar(keyType, key.Value()).(string)
		if !ok {
			return true
		}

		if valType != nil {
			result[convertedKey] = convertToMap(val.Message().Interface())
		} else {
			result[convertedKey] = convertScalar(fd.MapValue(), val)
		}

		return true
	})

	return result
}

//nolint:cyclop
func convertScalar(fd protoreflect.FieldDescriptor, value protoreflect.Value) any {
	const nullValue = "google.protobuf.NullValue"

	switch fd.Kind() {
	case protoreflect.BoolKind:
		return value.Bool()
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return json.Number(value.String())
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return json.Number(value.String())
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return json.Number(value.String())
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return json.Number(value.String())
	case protoreflect.FloatKind:
		return json.Number(value.String())
	case protoreflect.DoubleKind:
		return json.Number(value.String())
	case protoreflect.StringKind:
		return value.String()
	case protoreflect.BytesKind:
		return base64.StdEncoding.EncodeToString(value.Bytes())
	case protoreflect.EnumKind:
		if fd.Enum().FullName() == nullValue {
			return nil
		}

		desc := fd.Enum().Values().ByNumber(value.Enum())
		if desc != nil {
			return string(desc.Name())
		}

		return ""
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return convertToMap(value.Message().Interface())
	default:
		return nil
	}
}

//nolint:cyclop
func (m *grpcMocker) handleServerStream(stream grpc.ServerStream) error {
	for {
		inputMsg := dynamicpb.NewMessage(m.inputDesc)

		err := stream.RecvMsg(inputMsg)
		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return errors.Wrap(err, "failed to receive message")
		}

		query := m.newQuery(stream.Context(), inputMsg)

		result, err := m.budgerigar.FindByQuery(query)
		if err != nil {
			return errors.Wrap(err, "failed to find response")
		}

		found := result.Found()
		if found == nil {
			return status.Errorf(codes.NotFound, "No response found: %v", result.Similar())
		}

		if found.Output.Headers != nil {
			mdResp := make(metadata.MD, len(found.Output.Headers))
			for k, v := range found.Output.Headers {
				mdResp.Append(k, strings.Split(v, ";")...)
			}

			if err := stream.SetHeader(mdResp); err != nil {
				return errors.Wrap(err, "failed to set headers")
			}
		}

		outputMsg, err := m.newOutputMessage(found.Output.Data)
		if err != nil {
			return errors.Wrap(err, "failed to convert response to dynamic message")
		}

		if err := stream.SendMsg(outputMsg); err != nil {
			return errors.Wrap(err, "failed to send response")
		}
	}
}

func (m *grpcMocker) newOutputMessage(data map[string]any) (*dynamicpb.Message, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal map to JSON: %w", err)
	}

	msg := dynamicpb.NewMessage(m.outputDesc)

	err = protojson.Unmarshal(jsonData, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON into dynamic message: %w", err)
	}

	return msg, nil
}

func (m *grpcMocker) unaryHandler() grpc.MethodHandler {
	return func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
		req := dynamicpb.NewMessage(m.inputDesc)
		if err := dec(req); err != nil {
			return nil, err
		}

		if interceptor != nil {
			return interceptor(ctx, req, &grpc.UnaryServerInfo{
				Server:     srv,
				FullMethod: m.fullMethod,
			}, func(ctx context.Context, req any) (any, error) {
				if msg, ok := req.(*dynamicpb.Message); ok {
					return m.handleUnary(ctx, msg)
				}

				return nil, status.Errorf(codes.InvalidArgument, "expected *dynamicpb.Message, got %T", req)
			})
		}

		return m.handleUnary(ctx, req)
	}
}

//nolint:cyclop
func (m *grpcMocker) handleUnary(ctx context.Context, req *dynamicpb.Message) (*dynamicpb.Message, error) {
	query := m.newQuery(ctx, req)

	result, err := m.budgerigar.FindByQuery(query)
	if err != nil {
		return nil, err
	}

	found := result.Found()

	if found == nil {
		return nil, stubNotFoundError(query, result)
	}

	if found.Output.Error != "" || found.Output.Code != nil {
		if found.Output.Code == nil {
			return nil, status.Error(codes.Aborted, found.Output.Error)
		}

		if *found.Output.Code != codes.OK {
			return nil, status.Error(*found.Output.Code, found.Output.Error)
		}
	}

	outputMsg, err := m.newOutputMessage(found.Output.Data)
	if err != nil {
		return nil, err
	}

	if found.Output.Headers != nil {
		mdResp := make(metadata.MD, len(found.Output.Headers))
		for k, v := range found.Output.Headers {
			mdResp.Append(k, strings.Split(v, ";")...)
		}

		if err := grpc.SetHeader(ctx, mdResp); err != nil {
			return nil, errors.Wrap(err, "failed to set headers")
		}
	}

	return outputMsg, nil
}

//nolint:cyclop
func (m *grpcMocker) handleClientStream(stream grpc.ServerStream) error {
	var (
		allResponses []map[string]any
		lastHeaders  metadata.MD
	)

	for {
		inputMsg := dynamicpb.NewMessage(m.inputDesc)

		err := stream.RecvMsg(inputMsg)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return errors.Wrap(err, "failed to receive message")
		}

		query := m.newQuery(stream.Context(), inputMsg)

		result, err := m.budgerigar.FindByQuery(query)
		if err != nil {
			return errors.Wrap(err, "failed to find response")
		}

		found := result.Found()
		if found == nil {
			return status.Errorf(codes.NotFound, "No response found: %v", result.Similar())
		}

		allResponses = append(allResponses, found.Output.Data)

		if found.Output.Headers != nil {
			lastHeaders = make(metadata.MD, len(found.Output.Headers))
			for k, v := range found.Output.Headers {
				lastHeaders.Append(k, strings.Split(v, ";")...)
			}
		}
	}

	if lastHeaders != nil {
		if err := stream.SetHeader(lastHeaders); err != nil {
			return errors.Wrap(err, "failed to set headers")
		}
	}

	outputMsg, err := m.newOutputMessage(allResponses[len(allResponses)-1])
	if err != nil {
		return errors.Wrap(err, "failed to convert response to dynamic message")
	}

	return stream.SendMsg(outputMsg)
}

//nolint:cyclop
func (m *grpcMocker) handleBidiStream(stream grpc.ServerStream) error {
	for {
		inputMsg := dynamicpb.NewMessage(m.inputDesc)

		err := stream.RecvMsg(inputMsg)
		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return errors.Wrap(err, "failed to receive message")
		}

		query := m.newQuery(stream.Context(), inputMsg)

		result, err := m.budgerigar.FindByQuery(query)
		if err != nil {
			return errors.Wrap(err, "failed to find response")
		}

		found := result.Found()
		if found == nil {
			return status.Errorf(codes.NotFound, "No response found: %v", result.Similar())
		}

		outputMsg, err := m.newOutputMessage(found.Output.Data)
		if err != nil {
			return errors.Wrap(err, "failed to convert response to dynamic message")
		}

		if found.Output.Headers != nil {
			mdResp := make(metadata.MD, len(found.Output.Headers))
			for k, v := range found.Output.Headers {
				mdResp.Append(k, strings.Split(v, ";")...)
			}

			if err := stream.SetHeader(mdResp); err != nil {
				return errors.Wrap(err, "failed to set headers")
			}
		}

		if err := stream.SendMsg(outputMsg); err != nil {
			return errors.Wrap(err, "failed to send response")
		}
	}
}

func NewGRPCServer(
	network, address string,
	params *protoloc.Arguments,
	budgerigar *stuber.Budgerigar,
	waiter Extender,
) *GRPCServer {
	return &GRPCServer{
		network:    network,
		address:    address,
		params:     params,
		budgerigar: budgerigar,
		waiter:     waiter,
	}
}

//nolint:cyclop,funlen
func (s *GRPCServer) Build(ctx context.Context) (*grpc.Server, error) {
	descriptors, err := protoset.Build(ctx, s.params.Imports(), s.params.ProtoPath())
	if err != nil {
		return nil, errors.Wrap(err, "failed to build descriptors")
	}

	logger := zerolog.Ctx(ctx)

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpccontext.UnaryInterceptor(logger),
			LogUnaryInterceptor,
		),
		grpc.ChainStreamInterceptor(
			grpccontext.StreamInterceptor(logger),
			LogStreamInterceptor,
		),
	)

	healthcheck := health.NewServer()

	healthcheck.SetServingStatus("gripmock", healthgrpc.HealthCheckResponse_NOT_SERVING)

	healthgrpc.RegisterHealthServer(server, healthcheck)
	reflection.Register(server)

	for _, descriptor := range descriptors {
		for _, file := range descriptor.GetFile() {
			for _, svc := range file.GetService() {
				serviceDesc := grpc.ServiceDesc{
					ServiceName: fmt.Sprintf("%s.%s", file.GetPackage(), svc.GetName()),
					HandlerType: (*any)(nil),
				}

				for _, method := range svc.GetMethod() {
					inputDesc, err := getMessageDescriptor(method.GetInputType())
					if err != nil {
						logger.Fatal().Err(err).Msg("Failed to get input message descriptor")
					}

					outputDesc, err := getMessageDescriptor(method.GetOutputType())
					if err != nil {
						logger.Fatal().Err(err).Msg("Failed to get output message descriptor")
					}

					m := &grpcMocker{
						budgerigar: s.budgerigar,

						inputDesc:  inputDesc,
						outputDesc: outputDesc,

						packageName: file.GetPackage(),
						serviceName: svc.GetName(),
						methodName:  method.GetName(),
						fullMethod:  fmt.Sprintf("/%s.%s/%s", file.GetPackage(), svc.GetName(), method.GetName()),

						serverStream: method.GetServerStreaming(),
						clientStream: method.GetClientStreaming(),
					}

					if method.GetServerStreaming() || method.GetClientStreaming() {
						serviceDesc.Streams = append(serviceDesc.Streams, grpc.StreamDesc{
							StreamName:    method.GetName(),
							Handler:       m.streamHandler,
							ServerStreams: m.serverStream,
							ClientStreams: m.clientStream,
						})
					} else {
						serviceDesc.Methods = append(serviceDesc.Methods, grpc.MethodDesc{
							MethodName: method.GetName(),
							Handler:    m.unaryHandler(),
						})
					}
				}

				server.RegisterService(&serviceDesc, nil)

				zerolog.Ctx(ctx).Info().Str("service", serviceDesc.ServiceName).Msg("Registered gRPC service")
			}
		}
	}

	go func() {
		s.waiter.Wait(ctx)

		select {
		case <-ctx.Done():
			return
		default:
			healthcheck.SetServingStatus("gripmock", healthgrpc.HealthCheckResponse_SERVING)
			logger.Info().Msg("gRPC server is ready to accept requests")
		}
	}()

	return server, nil
}

func getMessageDescriptor(messageType string) (protoreflect.MessageDescriptor, error) { //nolint:ireturn
	msgName := protoreflect.FullName(strings.TrimPrefix(messageType, "."))

	desc, err := protoregistry.GlobalFiles.FindDescriptorByName(msgName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Message descriptor not found: %v", err)
	}

	msgDesc, ok := desc.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, status.Errorf(codes.Internal, "Not a message descriptor: %s", msgName)
	}

	return msgDesc, nil
}

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
		slash   = "/"
		unknown = "unknown"
	)

	parts := strings.Split(fullMethod, slash)
	if len(parts) != 3 { //nolint:mnd
		return unknown, unknown
	}

	return parts[1], parts[2]
}

func getPeerAddress(p *peer.Peer) string {
	if p != nil && p.Addr != nil {
		return p.Addr.String()
	}

	return "unknown"
}

func protoToJSON(msg any) []byte {
	if msg == nil || isNilInterface(msg) {
		return nil
	}

	message, ok := msg.(proto.Message)
	if !ok {
		return nil
	}

	marshaller := protojson.MarshalOptions{EmitUnpopulated: false}
	data, _ := marshaller.Marshal(message)

	return data
}

func isNilInterface(v any) bool {
	return v == nil || (reflect.ValueOf(v).Kind() == reflect.Ptr && reflect.ValueOf(v).IsNil())
}

func toLogArray(items ...any) *zerolog.Array {
	arr := zerolog.Arr()

	for _, item := range items {
		if value := protoToJSON(item); value != nil {
			arr = arr.RawJSON(value)
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
	s.responses = append(s.responses, m)

	return s.ServerStream.SendMsg(m)
}

func (s *loggingStream) RecvMsg(m any) error {
	s.requests = append(s.requests, m)

	return s.ServerStream.RecvMsg(m)
}
