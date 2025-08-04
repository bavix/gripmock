package app

//nolint:revive
import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/goccy/go-json"
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
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	protoloc "github.com/bavix/gripmock/v3/internal/domain/proto"
	"github.com/bavix/gripmock/v3/internal/domain/protoset"
	"github.com/bavix/gripmock/v3/internal/infra/grpccontext"
)

// excludedHeaders contains headers that should be excluded from stub matching.
var excludedHeaders = []string{":authority", "content-type", "grpc-accept-encoding", "user-agent", "accept-encoding"}

// processHeaders converts metadata to headers map, excluding specified headers.
func processHeaders(md metadata.MD) map[string]any {
	if len(md) == 0 {
		return nil
	}

	headers := make(map[string]any)

	for k, v := range md {
		if !slices.Contains(excludedHeaders, k) {
			headers[k] = strings.Join(v, ";")
		}
	}

	return headers
}

// setStreamHeaders sets response headers on a gRPC stream.
func setStreamHeaders(stream grpc.ServerStream, headers map[string]string) error {
	if headers == nil {
		return nil
	}

	mdResp := make(metadata.MD, len(headers))

	for k, v := range headers {
		mdResp.Append(k, strings.Split(v, ";")...)
	}

	return stream.SetHeader(mdResp)
}

// sendStreamMessage sends a message on a gRPC stream with error handling.
func sendStreamMessage(stream grpc.ServerStream, msg *dynamicpb.Message) error {
	if err := stream.SendMsg(msg); err != nil {
		return errors.Wrap(err, "failed to send response")
	}

	return nil
}

// receiveStreamMessage receives a message from a gRPC stream with error handling.
func receiveStreamMessage(stream grpc.ServerStream, msg *dynamicpb.Message) error {
	err := stream.RecvMsg(msg)
	if err != nil {
		return errors.Wrap(err, "failed to receive message")
	}

	return nil
}

// shouldUseStreamOutput checks if stream output should be used based on stub configuration and message index.
func shouldUseStreamOutput(stream []any, messageIndex int) bool {
	return len(stream) > 0 && messageIndex >= 0 && messageIndex < len(stream)
}

const serviceReflection = "grpc.reflection.v1.ServerReflection"

type GRPCServer struct {
	network     string
	address     string
	params      *protoloc.Arguments
	budgerigar  *stuber.Budgerigar
	waiter      Extender
	healthcheck *health.Server
}

type grpcMocker struct {
	budgerigar *stuber.Budgerigar

	inputDesc  protoreflect.MessageDescriptor
	outputDesc protoreflect.MessageDescriptor

	fullServiceName string
	serviceName     string
	methodName      string
	fullMethod      string

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
		Service: m.fullServiceName,
		Method:  m.methodName,
		Data:    convertToMap(msg),
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		query.Headers = processHeaders(md)
	}

	return query
}

// newQueryV2 creates a new V2 query for improved performance.
func (m *grpcMocker) newQueryV2(ctx context.Context, msg *dynamicpb.Message) stuber.QueryV2 {
	query := stuber.QueryV2{
		Service: m.fullServiceName,
		Method:  m.methodName,
		Input:   []map[string]any{convertToMap(msg)},
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		query.Headers = processHeaders(md)
	}

	return query
}

// newQueryBidi creates a new bidirectional streaming query.
func (m *grpcMocker) newQueryBidi(ctx context.Context) stuber.QueryBidi {
	query := stuber.QueryBidi{
		Service: m.fullServiceName,
		Method:  m.methodName,
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		query.Headers = processHeaders(md)
	}

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

func (m *grpcMocker) delay(ctx context.Context, delayDur time.Duration) {
	if delayDur == 0 {
		return
	}

	timer := time.NewTimer(delayDur)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		return
	}
}

func (m *grpcMocker) handleServerStream(stream grpc.ServerStream) error {
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

	// Set headers once at the beginning
	if found.Output.Headers != nil {
		mdResp := make(metadata.MD, len(found.Output.Headers))
		for k, v := range found.Output.Headers {
			mdResp.Append(k, strings.Split(v, ";")...)
		}

		if err := stream.SetHeader(mdResp); err != nil {
			return errors.Wrap(err, "failed to set headers")
		}
	}

	// For server streaming, prioritize Stream field if it is not empty; fallback to Data if Stream is empty
	if len(found.Output.Stream) > 0 {
		return m.handleArrayStreamData(stream, found)
	}

	// Fallback to Data for single message streaming
	return m.handleNonArrayStreamData(stream, found)
}

func (m *grpcMocker) handleArrayStreamData(stream grpc.ServerStream, found *stuber.Stub) error {
	// Store context done channel outside the loop for clarity; context.Done() is already cached
	done := stream.Context().Done()

	// Send all messages, validating each element incrementally
	for i, streamData := range found.Output.Stream {
		select {
		case <-done:
			return stream.Context().Err()
		default:
		}

		// Validate type of each streamData just before sending
		outputData, ok := streamData.(map[string]any)
		if !ok {
			return status.Errorf(
				codes.Internal,
				"invalid data format in stream array at index %d: got %T, expected map[string]any",
				i, streamData,
			)
		}

		// Apply delay before sending each message
		m.delay(stream.Context(), found.Output.Delay)

		outputMsg, err := m.newOutputMessage(outputData)
		if err != nil {
			return errors.Wrap(err, "failed to convert response to dynamic message")
		}

		if err := sendStreamMessage(stream, outputMsg); err != nil {
			return err
		}
	}

	return nil
}

func (m *grpcMocker) handleNonArrayStreamData(stream grpc.ServerStream, found *stuber.Stub) error {
	// Original behavior for non-array data, with context cancellation check
	done := stream.Context().Done()

	for {
		select {
		case <-done:
			return stream.Context().Err()
		default:
		}

		m.delay(stream.Context(), found.Output.Delay)

		outputMsg, err := m.newOutputMessage(found.Output.Data)
		if err != nil {
			return errors.Wrap(err, "failed to convert response to dynamic message")
		}

		if err := sendStreamMessage(stream, outputMsg); err != nil {
			return err //nolint:wrapcheck
		}

		// In server streaming, do not receive further messages from the client after the initial request.
		// The server should only send messages to the client.
		// Check for EOF to determine if client has closed the stream
		if err := stream.RecvMsg(nil); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			return errors.Wrap(err, "failed to receive message")
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
			return nil, err //nolint:wrapcheck
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

func (m *grpcMocker) handleUnary(ctx context.Context, req *dynamicpb.Message) (*dynamicpb.Message, error) {
	// Try V2 API first for better performance
	queryV2 := m.newQueryV2(ctx, req)

	result, err := m.budgerigar.FindByQueryV2(queryV2)
	if err != nil {
		// Fallback to V1 API for backward compatibility
		query := m.newQuery(ctx, req)

		result, err = m.budgerigar.FindByQuery(query)
		if err != nil {
			return nil, err //nolint:wrapcheck
		}
	}

	found := result.Found()
	if found == nil {
		// Use appropriate error function based on which API was used
		if queryV2.Service != "" {
			errorFormatter := NewErrorFormatter()

			return nil, status.Error(codes.NotFound, errorFormatter.FormatStubNotFoundErrorV2(queryV2, result).Error())
		}

		// Fallback to V1 error format
		query := m.newQuery(ctx, req)

		return nil, status.Error(codes.NotFound, stubNotFoundError(query, result).Error())
	}

	m.delay(ctx, found.Output.Delay)

	if err := m.handleOutputError(found.Output); err != nil {
		return nil, err //nolint:wrapcheck
	}

	outputMsg, err := m.newOutputMessage(found.Output.Data)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	if err := m.setResponseHeaders(ctx, found.Output.Headers); err != nil {
		return nil, err //nolint:wrapcheck
	}

	return outputMsg, nil
}

func (m *grpcMocker) handleOutputError(output stuber.Output) error {
	if output.Error != "" || output.Code != nil {
		if output.Code == nil {
			return status.Error(codes.Aborted, output.Error)
		}

		if *output.Code != codes.OK {
			return status.Error(*output.Code, output.Error)
		}
	}

	return nil
}

func (m *grpcMocker) setResponseHeaders(ctx context.Context, headers map[string]string) error {
	if headers == nil {
		return nil
	}

	mdResp := make(metadata.MD, len(headers))
	for k, v := range headers {
		mdResp.Append(k, strings.Split(v, ";")...)
	}

	return grpc.SetHeader(ctx, mdResp)
}

// tryV2API attempts to find a stub using V2 API.
func (m *grpcMocker) tryV2API(messages []map[string]any, md metadata.MD) (*stuber.Result, error) {
	queryV2 := stuber.QueryV2{
		Service: m.fullServiceName,
		Method:  m.methodName,
		Input:   messages,
	}

	// Add headers to V2 query
	if len(md) > 0 {
		queryV2.Headers = processHeaders(md)
	}

	return m.budgerigar.FindByQueryV2(queryV2)
}

// tryV1APIFallback attempts to find a stub using V1 API as fallback.
func (m *grpcMocker) tryV1APIFallback(messages []map[string]any, md metadata.MD) (*stuber.Result, error) {
	// Try each message individually (from last to first for better matching) using V1 API
	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]

		query := stuber.Query{
			Service: m.fullServiceName,
			Method:  m.methodName,
			Data:    message,
		}

		// Add headers to V1 query
		if len(md) > 0 {
			query.Headers = processHeaders(md)
		}

		result, foundErr := m.budgerigar.FindByQuery(query)
		if foundErr == nil && result != nil && result.Found() != nil {
			return result, nil
		}
	}

	return nil, status.Errorf(codes.NotFound, "failed to find response for client stream")
}

func (m *grpcMocker) handleClientStream(stream grpc.ServerStream) error {
	// Collect all messages from client
	messages, err := m.collectClientMessages(stream)
	if err != nil {
		return err
	}

	// Try to find stub
	found, err := m.tryFindStub(stream, messages)
	if err != nil {
		return err
	}

	// Send response
	return m.sendClientStreamResponse(stream, found)
}

// collectClientMessages collects all messages from the client stream.
func (m *grpcMocker) collectClientMessages(stream grpc.ServerStream) ([]map[string]any, error) {
	var messages []map[string]any

	for {
		inputMsg := dynamicpb.NewMessage(m.inputDesc)

		err := receiveStreamMessage(stream, inputMsg)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, err //nolint:wrapcheck
		}

		messages = append(messages, convertToMap(inputMsg))
	}

	return messages, nil
}

// tryFindStub attempts to find a matching stub using V2 API first, then falls back to V1 API.
func (m *grpcMocker) tryFindStub(stream grpc.ServerStream, messages []map[string]any) (*stuber.Stub, error) {
	// Add headers
	md, _ := metadata.FromIncomingContext(stream.Context())

	// Try V2 API first
	result, foundErr := m.tryV2API(messages, md)

	// If V2 API fails, try V1 API for backward compatibility
	if foundErr != nil || result == nil || result.Found() == nil {
		result, foundErr = m.tryV1APIFallback(messages, md)
	}

	if foundErr != nil || result == nil || result.Found() == nil {
		// Return an error message with service and method context to aid debugging
		errorMsg := fmt.Sprintf("Failed to find response for client stream (service: %s, method: %s)", m.serviceName, m.methodName)
		if foundErr != nil {
			errorMsg += fmt.Sprintf(" - Error: %v", foundErr)
		}

		return nil, status.Error(codes.NotFound, errorMsg)
	}

	found := result.Found()
	if found == nil {
		return nil, status.Errorf(codes.NotFound, "No response found for client stream: %v", result.Similar())
	}

	return found, nil
}

// sendClientStreamResponse sends the response for client streaming.
func (m *grpcMocker) sendClientStreamResponse(stream grpc.ServerStream, found *stuber.Stub) error {
	m.delay(stream.Context(), found.Output.Delay)

	// Handle headers
	if err := setStreamHeaders(stream, found.Output.Headers); err != nil {
		return errors.Wrap(err, "failed to set headers")
	}

	// Send response
	outputMsg, err := m.newOutputMessage(found.Output.Data)
	if err != nil {
		return errors.Wrap(err, "failed to convert response to dynamic message")
	}

	return stream.SendMsg(outputMsg)
}

//nolint:cyclop
func (m *grpcMocker) handleBidiStream(stream grpc.ServerStream) error {
	// Initialize bidirectional streaming session
	queryBidi := m.newQueryBidi(stream.Context())

	bidiResult, err := m.budgerigar.FindByQueryBidi(queryBidi)
	if err != nil {
		return errors.Wrapf(err, "failed to initialize bidirectional streaming session: %v", err)
	}

	for {
		inputMsg := dynamicpb.NewMessage(m.inputDesc)

		err := receiveStreamMessage(stream, inputMsg)
		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return err //nolint:wrapcheck
		}

		// Process message through bidirectional streaming
		stub, err := bidiResult.Next(convertToMap(inputMsg))
		if err != nil {
			return errors.Wrap(err, "failed to process bidirectional message")
		}

		m.delay(stream.Context(), stub.Output.Delay)

		// For bidirectional streaming, use Stream output if available
		var outputData map[string]any
		if shouldUseStreamOutput(stub.Output.Stream, bidiResult.GetMessageIndex()) {
			// Get the corresponding response from the stream
			if streamData, ok := stub.Output.Stream[bidiResult.GetMessageIndex()].(map[string]any); ok {
				outputData = streamData
			} else {
				// Fallback to Data if Stream element is not a map
				outputData = stub.Output.Data
			}
		} else {
			// Fallback to Data if no Stream available
			outputData = stub.Output.Data
		}

		outputMsg, err := m.newOutputMessage(outputData)
		if err != nil {
			return errors.Wrap(err, "failed to convert response to dynamic message")
		}

		if err := setStreamHeaders(stream, stub.Output.Headers); err != nil {
			return errors.Wrap(err, "failed to set headers")
		}

		if err := sendStreamMessage(stream, outputMsg); err != nil {
			return err //nolint:wrapcheck
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

func (s *GRPCServer) Build(ctx context.Context) (*grpc.Server, error) {
	descriptors, err := protoset.Build(ctx, s.params.Imports(), s.params.ProtoPath())
	if err != nil {
		return nil, errors.Wrap(err, "failed to build descriptors")
	}

	server := s.createServer(ctx)
	s.setupHealthCheck(server)
	s.registerServices(ctx, server, descriptors)
	s.startHealthCheckRoutine(ctx)

	return server, nil
}

func (s *GRPCServer) createServer(ctx context.Context) *grpc.Server {
	logger := zerolog.Ctx(ctx)

	return grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpccontext.UnaryInterceptor(logger),
			LogUnaryInterceptor,
		),
		grpc.ChainStreamInterceptor(
			grpccontext.StreamInterceptor(logger),
			LogStreamInterceptor,
		),
	)
}

func (s *GRPCServer) setupHealthCheck(server *grpc.Server) {
	healthcheck := health.NewServer()
	healthcheck.SetServingStatus("gripmock", healthgrpc.HealthCheckResponse_NOT_SERVING)
	healthgrpc.RegisterHealthServer(server, healthcheck)
	reflection.Register(server)

	// Store healthcheck server for later status updates
	s.healthcheck = healthcheck
}

func (s *GRPCServer) registerServices(ctx context.Context, server *grpc.Server, descriptors []*descriptorpb.FileDescriptorSet) {
	logger := zerolog.Ctx(ctx)

	for _, descriptor := range descriptors {
		for _, file := range descriptor.GetFile() {
			for _, svc := range file.GetService() {
				serviceDesc := s.createServiceDesc(file, svc)
				s.registerServiceMethods(ctx, &serviceDesc, svc)
				server.RegisterService(&serviceDesc, nil)
				logger.Info().Str("service", serviceDesc.ServiceName).Msg("Registered gRPC service")
			}
		}
	}
}

func (s *GRPCServer) createServiceDesc(file *descriptorpb.FileDescriptorProto, svc *descriptorpb.ServiceDescriptorProto) grpc.ServiceDesc {
	return grpc.ServiceDesc{
		ServiceName: getServiceName(file, svc),
		HandlerType: (*any)(nil),
	}
}

func (s *GRPCServer) registerServiceMethods(ctx context.Context, serviceDesc *grpc.ServiceDesc, svc *descriptorpb.ServiceDescriptorProto) {
	logger := zerolog.Ctx(ctx)

	for _, method := range svc.GetMethod() {
		inputDesc, err := getMessageDescriptor(method.GetInputType())
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to get input message descriptor")
		}

		outputDesc, err := getMessageDescriptor(method.GetOutputType())
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to get output message descriptor")
		}

		m := s.createGrpcMocker(serviceDesc, svc, method, inputDesc, outputDesc)

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
}

func (s *GRPCServer) createGrpcMocker(serviceDesc *grpc.ServiceDesc, svc *descriptorpb.ServiceDescriptorProto, method *descriptorpb.MethodDescriptorProto, inputDesc, outputDesc protoreflect.MessageDescriptor) *grpcMocker {
	return &grpcMocker{
		budgerigar: s.budgerigar,

		inputDesc:  inputDesc,
		outputDesc: outputDesc,

		fullServiceName: serviceDesc.ServiceName,
		serviceName:     svc.GetName(),
		methodName:      method.GetName(),
		fullMethod:      fmt.Sprintf("/%s/%s", serviceDesc.ServiceName, method.GetName()),

		serverStream: method.GetServerStreaming(),
		clientStream: method.GetClientStreaming(),
	}
}

func (s *GRPCServer) startHealthCheckRoutine(ctx context.Context) {
	logger := zerolog.Ctx(ctx)

	go func() {
		s.waiter.Wait(ctx)

		select {
		case <-ctx.Done():
			return
		default:
			logger.Info().Msg("gRPC server is ready to accept requests")
			s.healthcheck.SetServingStatus("gripmock", healthgrpc.HealthCheckResponse_SERVING)
		}
	}()
}

// getServiceName constructs the fully qualified service name by combining the package name
// and the service name. If the package name is empty, it returns only the service name,
// avoiding a leading dot in the result.
func getServiceName(file *descriptorpb.FileDescriptorProto, svc *descriptorpb.ServiceDescriptorProto) string {
	if file.GetPackage() != "" {
		return fmt.Sprintf("%s.%s", file.GetPackage(), svc.GetName())
	}

	return svc.GetName()
}

func getMessageDescriptor(messageType string) (protoreflect.MessageDescriptor, error) { //nolint:ireturn // Returns protobuf interface which is required for compatibility
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
	if !ok || message == nil {
		return nil
	}

	marshaller := protojson.MarshalOptions{EmitUnpopulated: false}

	data, err := marshaller.Marshal(message)
	if err != nil {
		return nil
	}

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
