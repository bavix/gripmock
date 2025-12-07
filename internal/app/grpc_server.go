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
	"github.com/bavix/gripmock/v3/internal/domain/types"
	"github.com/bavix/gripmock/v3/internal/infra/grpccontext"
	"github.com/bavix/gripmock/v3/internal/infra/grpcservice"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	localtypes "github.com/bavix/gripmock/v3/internal/infra/types"
	"github.com/bavix/gripmock/v3/pkg/plugins"
)

// excludedHeaders contains headers that should be excluded from stub matching.
//
//nolint:gochecknoglobals
var excludedHeaders = []string{
	":authority",
	"content-type",
	"grpc-accept-encoding",
	"user-agent",
	"accept-encoding",
}

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

const serviceReflection = "grpc.reflection.v1.ServerReflection"

type GRPCServer struct {
	network        string
	address        string
	params         *protoloc.Arguments
	budgerigar     *stuber.Budgerigar
	serviceManager *grpcservice.Manager
	waiter         Extender
	healthcheck    *health.Server
	errorFormatter *ErrorFormatter
	pluginRegistry plugins.Registry
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

	errorFormatter *ErrorFormatter
	pluginRegistry plugins.Registry
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

func (m *grpcMocker) delay(ctx context.Context, delayDur localtypes.Duration) {
	if delayDur == 0 {
		return
	}

	timer := time.NewTimer(time.Duration(delayDur))
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		return
	}
}

//nolint:nestif,cyclop,funlen
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

	// Process dynamic templates if the output contains them
	if found.Output.HasTemplates() {
		requestData := convertToMap(inputMsg)

		headers := make(map[string]any)
		if md, ok := metadata.FromIncomingContext(stream.Context()); ok {
			headers = processHeaders(md)
		}

		if err := found.Output.ProcessDynamicOutput(
			requestData,
			headers,
			0,
			nil,
			1,
			found.Times,
			found.ID.String(),
			m.pluginRegistry,
		); err != nil {
			return status.Error(codes.Internal, fmt.Sprintf("failed to process dynamic templates: %v", err))
		}
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

	// For server streaming, if Stream is not empty, send it first, then throw error if specified
	if found.IsServerStream() {
		if len(found.Output.Stream) > 0 {
			if err := m.handleArrayStreamData(stream, found); err != nil {
				return err
			}

			// After sending the stream, if output.error is set, return it now
			if err := m.handleOutputError(stream.Context(), stream, found.Output); err != nil { //nolint:wrapcheck
				return err
			}

			return nil
		}

		// If stream is empty and error is specified, return it immediately
		if err := m.handleOutputError(stream.Context(), stream, found.Output); err != nil { //nolint:wrapcheck
			return err
		}

		// Fallback: no stream and no error – treat as single message
		return m.handleNonArrayStreamData(stream, found)
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

		// Check if this stub is v4 format (has outputs field)
		if m.isV4Stub(found) {
			if streamStep, err := m.parseAsV4StreamStep(streamData); err == nil {
				if err := m.processV4StreamStep(stream, streamStep); err != nil {
					return err
				}

				continue
			}
		}

		// Fallback to legacy handling for backward compatibility
		outputData, ok := streamData.(map[string]any)
		if !ok {
			return status.Errorf(
				codes.Internal,
				"invalid data format in stream array at index %d: got %T, expected map[string]any",
				i, streamData,
			)
		}

		if err := m.processLegacyStreamStep(stream, outputData, found); err != nil {
			return err
		}
	}

	return nil
}

// parseAsV4StreamStep attempts to parse stream data as v4 StreamStep.
// Returns error if the data doesn't match v4 StreamStep structure.
//
//nolint:cyclop
func (m *grpcMocker) parseAsV4StreamStep(streamData any) (*types.StreamStep, error) {
	// Try direct type assertion first
	if streamStep, ok := streamData.(*types.StreamStep); ok {
		return streamStep, nil
	}

	// Try to convert from map[string]any if it has v4 StreamStep structure
	//nolint:nestif
	if dataMap, ok := streamData.(map[string]any); ok {
		streamStep := &types.StreamStep{}

		// Check for send field
		if sendData, hasSend := dataMap["send"]; hasSend {
			if sendMap, ok := sendData.(map[string]any); ok {
				streamStep.Send = sendMap
			}
		}

		// Check for delay field
		if delayData, hasDelay := dataMap["delay"]; hasDelay {
			if delayStr, ok := delayData.(string); ok {
				streamStep.Delay = delayStr
			}
		}

		// Check for end field
		if endData, hasEnd := dataMap["end"]; hasEnd {
			if endMap, ok := endData.(map[string]any); ok {
				streamStep.End = &types.GrpcStatus{}
				if code, ok := endMap["code"].(string); ok {
					streamStep.End.Code = code
				}

				if message, ok := endMap["message"].(string); ok {
					streamStep.End.Message = message
				}
			}
		}

		// Validate that at least one v4 field is present
		if streamStep.Send == nil && streamStep.Delay == "" && streamStep.End == nil {
			return nil, errors.New("no v4 StreamStep fields found")
		}

		return streamStep, nil
	}

	return nil, errors.New("data is not a v4 StreamStep")
}

// isV4Stub checks if the stub is v4 format by checking if it has OutputsRawV4.
func (m *grpcMocker) isV4Stub(stub *stuber.Stub) bool {
	return len(stub.OutputsRawV4) > 0
}

// processV4StreamStep handles v4 StreamStep structure using proper types.
func (m *grpcMocker) processV4StreamStep(stream grpc.ServerStream, streamStep *types.StreamStep) error {
	// Handle delay first if present
	if streamStep.Delay != "" {
		if d, err := time.ParseDuration(streamStep.Delay); err == nil {
			m.delay(stream.Context(), localtypes.Duration(d))
		}
	}

	// Handle send if present
	if streamStep.Send != nil {
		outputMsg, err := m.newOutputMessage(streamStep.Send)
		if err != nil {
			return errors.Wrap(err, "failed to convert v4 send response to dynamic message")
		}

		if err := sendStreamMessage(stream, outputMsg); err != nil {
			return err
		}
	}

	// Handle end status if present
	if streamStep.End != nil {
		// Process end status - this would typically end the stream
		// For now, we'll just log it as the current implementation doesn't handle stream termination
		_ = streamStep.End // Stream termination not yet implemented
	}

	return nil
}

// processLegacyStreamStep handles legacy stream structure for backward compatibility.
//
//nolint:cyclop
func (m *grpcMocker) processLegacyStreamStep(stream grpc.ServerStream, outputData map[string]any, found *stuber.Stub) error {
	// Per-step delay
	if rawDelay, hasDelay := outputData["delay"]; hasDelay {
		if s, ok := rawDelay.(string); ok && s != "" {
			if d, err := time.ParseDuration(s); err == nil {
				m.delay(stream.Context(), localtypes.Duration(d))
			}
		}
		// If this element has only delay, skip sending a message
		if _, hasSend := outputData["send"]; !hasSend && len(outputData) == 1 {
			return nil
		}
	}

	// Determine payload to send
	payload := outputData
	if rawSend, hasSend := outputData["send"]; hasSend {
		if sendMap, ok := rawSend.(map[string]any); ok {
			payload = sendMap
		}
	}

	// Also honor top-level per-message delay for backward compatibility
	m.delay(stream.Context(), found.Output.Delay)

	outputMsg, err := m.newOutputMessage(payload)
	if err != nil {
		return errors.Wrap(err, "failed to convert response to dynamic message")
	}

	if err := sendStreamMessage(stream, outputMsg); err != nil { //nolint:wrapcheck
		return err
	}

	return nil
}

func (m *grpcMocker) handleNonArrayStreamData(stream grpc.ServerStream, found *stuber.Stub) error {
	// Check for output error before attempting to send data
	if err := m.handleOutputError(stream.Context(), stream, found.Output); err != nil {
		return err
	}

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

		if err := sendStreamMessage(stream, outputMsg); err != nil { //nolint:wrapcheck
			return err
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
	// Prevent marshaling to JSON null which breaks protojson.Unmarshal
	if data == nil {
		data = map[string]any{}
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal map to JSON: %w", err)
	}

	msg := dynamicpb.NewMessage(m.outputDesc)

	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}

	err = unmarshaler.Unmarshal(jsonData, msg)
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

//nolint:gocognit,cyclop,funlen
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
		// V2 API didn't find a stub, try V1 API as fallback
		query := m.newQuery(ctx, req)

		result, err = m.budgerigar.FindByQuery(query)
		if err != nil {
			return nil, err //nolint:wrapcheck
		}

		found = result.Found()
		if found == nil {
			// Use appropriate error function based on which API was used
			if queryV2.Service != "" {
				return nil, status.Error(codes.NotFound, m.errorFormatter.FormatStubNotFoundErrorV2(queryV2, result).Error())
			}

			// Fallback to V1 error format
			return nil, status.Error(codes.NotFound, stubNotFoundError(query, result).Error())
		}
	}

	m.delay(ctx, found.Output.Delay)

	// Prepare output copy and merge v4 response headers before potential template processing
	outputToUse := found.Output

	outputToUse.Headers = deepCopyStringMap(found.Output.Headers)
	if len(found.ResponseHeaders) > 0 {
		if outputToUse.Headers == nil {
			outputToUse.Headers = make(map[string]string, len(found.ResponseHeaders))
		}

		for k, v := range found.ResponseHeaders {
			if _, exists := outputToUse.Headers[k]; !exists {
				outputToUse.Headers[k] = v
			}
		}
	}

	// Process dynamic templates on a deep copy to avoid mutating the stub
	if found.Output.HasTemplates() || found.Output.Error != "" || len(found.ResponseHeaders) > 0 {
		requestData := convertToMap(req)

		headers := make(map[string]any)
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			headers = processHeaders(md)
		}

		// deep copy Output fields (headers already prepared)
		outputToUse.Data = deepCopyMapAny(found.Output.Data)
		outputToUse.Stream = deepCopySliceAny(found.Output.Stream)

		if err := outputToUse.ProcessDynamicOutput(
			requestData,
			headers,
			0,
			nil,
			1,
			found.Times,
			found.ID.String(),
			m.pluginRegistry,
		); err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to process dynamic templates: %v", err))
		}
	}

	// Always send headers first (both for success and error cases)
	// Merge top-level ResponseHeaders (v4) with Output.Headers (legacy/execution path)
	if len(found.ResponseHeaders) > 0 {
		if outputToUse.Headers == nil {
			outputToUse.Headers = make(map[string]string, len(found.ResponseHeaders))
		}

		for k, v := range found.ResponseHeaders {
			if _, exists := outputToUse.Headers[k]; !exists {
				outputToUse.Headers[k] = v
			}
		}
	}

	if err := m.setResponseHeadersAny(ctx, nil, outputToUse.Headers); err != nil {
		return nil, err //nolint:wrapcheck
	}

	if err := m.handleOutputError(ctx, nil, outputToUse); err != nil {
		return nil, err //nolint:wrapcheck
	}

	outputMsg, err := m.newOutputMessage(outputToUse.Data)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	return outputMsg, nil
}

// buildResponseMetadata builds gRPC metadata from headers map.
func buildResponseMetadata(headers map[string]string) (metadata.MD, bool) {
	if len(headers) == 0 {
		return nil, false
	}

	mdResp := make(metadata.MD, len(headers))
	for k, v := range headers {
		mdResp.Append(k, strings.Split(v, ";")...)
	}

	return mdResp, true
}

// setResponseHeadersAny sets headers for success responses.
func (m *grpcMocker) setResponseHeadersAny(ctx context.Context, stream grpc.ServerStream, headers map[string]string) error {
	mdResp, ok := buildResponseMetadata(headers)
	if !ok {
		return nil
	}

	if stream != nil {
		return stream.SetHeader(mdResp)
	}

	// For unary calls ensure headers are flushed to the client
	return grpc.SendHeader(ctx, mdResp)
}

func (m *grpcMocker) handleOutputError(_ context.Context, _ grpc.ServerStream, output stuber.Output) error {
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
			f := result.Found()
			// Skip stubs that have no concrete output
			if f.Output.Data == nil && len(f.Output.Stream) == 0 && f.Output.Error == "" && f.Output.Code == nil {
				continue
			}

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
	return m.sendClientStreamResponse(stream, found, messages)
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
//

//nolint:gocognit,cyclop,funlen
func (m *grpcMocker) sendClientStreamResponse(stream grpc.ServerStream, found *stuber.Stub, messages []map[string]any) error {
	m.delay(stream.Context(), found.Output.Delay)

	// Process dynamic templates if the output contains them
	//nolint:nestif
	if found.Output.HasTemplates() {
		// For client streaming, we can use different strategies:
		// 1. Use the last message (most common case)
		// 2. Use all messages as an array
		// 3. Use a combined approach
		var requestData map[string]any

		if len(messages) > 0 {
			// Use the last non-empty message as primary data
			for i := len(messages) - 1; i >= 0; i-- {
				if len(messages[i]) > 0 {
					requestData = messages[i]

					break
				}
			}

			if requestData == nil {
				requestData = make(map[string]any)
			}
		} else {
			requestData = make(map[string]any)
		}

		headers := make(map[string]any)
		if md, ok := metadata.FromIncomingContext(stream.Context()); ok {
			headers = processHeaders(md)
		}

		// Provide only non-empty client messages to template context via .Requests
		allMessages := make([]any, 0, len(messages))
		for _, m := range messages {
			if len(m) == 0 {
				continue
			}

			allMessages = append(allMessages, m)
		}

		if err := found.Output.ProcessDynamicOutput(
			requestData,
			headers,
			0,
			allMessages,
			1,
			found.Times,
			found.ID.String(),
			m.pluginRegistry,
		); err != nil {
			return status.Error(codes.Internal, fmt.Sprintf("failed to process dynamic templates: %v", err))
		}
	}

	// If matched stub has no concrete output, try legacy fallback to find a usable response
	if found.Output.Data == nil && len(found.Output.Stream) == 0 && found.Output.Error == "" && found.Output.Code == nil {
		if md, ok := metadata.FromIncomingContext(stream.Context()); ok {
			if res, err := m.tryV1APIFallback(messages, md); err == nil && res != nil && res.Found() != nil {
				found = res.Found()
			}
		}
	}

	// Handle headers
	// If the output specifies an error, return it instead of sending a message
	if err := m.handleOutputError(stream.Context(), stream, found.Output); err != nil { //nolint:wrapcheck
		return err
	}

	if err := m.setResponseHeadersAny(stream.Context(), stream, found.Output.Headers); err != nil {
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

		// Make a deep copy of the output and process dynamic templates per message
		outputToUse := stub.Output
		if stub.Output.HasTemplates() {
			requestData := convertToMap(inputMsg)
			// Add message index for bidirectional streaming
			requestData["_message_index"] = bidiResult.GetMessageIndex()

			headers := make(map[string]any)
			if md, ok := metadata.FromIncomingContext(stream.Context()); ok {
				headers = processHeaders(md)
			}

			// Deep copy Output fields so each message is rendered independently
			outputToUse.Data = deepCopyMapAny(stub.Output.Data)
			outputToUse.Stream = deepCopySliceAny(stub.Output.Stream)
			outputToUse.Headers = deepCopyStringMap(stub.Output.Headers)

			if err := outputToUse.ProcessDynamicOutput(
				requestData, headers, bidiResult.GetMessageIndex(), nil, 1, stub.Times, stub.ID.String(), m.pluginRegistry,
			); err != nil {
				return status.Error(codes.Internal, fmt.Sprintf("failed to process dynamic templates: %v", err))
			}
		}

		// Send headers only once at the beginning of the stream
		if bidiResult.GetMessageIndex() == 0 {
			if err := m.setResponseHeadersAny(stream.Context(), stream, outputToUse.Headers); err != nil {
				return errors.Wrap(err, "failed to set headers")
			}
		}

		// If the output specifies an error, return it instead of sending a message
		if err := m.handleOutputError(stream.Context(), stream, outputToUse); err != nil { //nolint:wrapcheck
			return err
		}

		// Send response(s) based on output configuration
		if err := m.sendBidiResponses(stream, outputToUse, stub, bidiResult.GetMessageIndex()); err != nil {
			return err
		}
	}
}

func NewGRPCServer(
	network, address string,
	params *protoloc.Arguments,
	budgerigar *stuber.Budgerigar,
	serviceManager *grpcservice.Manager,
	waiter Extender,
	errorFormatter *ErrorFormatter,
	pluginRegistry plugins.Registry,
) *GRPCServer {
	return &GRPCServer{
		network:        network,
		address:        address,
		params:         params,
		budgerigar:     budgerigar,
		serviceManager: serviceManager,
		waiter:         waiter,
		errorFormatter: errorFormatter,
		pluginRegistry: pluginRegistry,
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
			grpccontext.PanicRecoveryUnaryInterceptor,
			grpccontext.UnaryInterceptor(logger),
			LogUnaryInterceptor,
		),
		grpc.ChainStreamInterceptor(
			grpccontext.PanicRecoveryStreamInterceptor,
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

	s.healthcheck = healthcheck
}

func (s *GRPCServer) registerServices(ctx context.Context, server *grpc.Server, descriptors []*descriptorpb.FileDescriptorSet) {
	logger := zerolog.Ctx(ctx)

	// First register all services in the service manager
	s.serviceManager.RegisterFromDescriptor(descriptors)

	// Then register with gRPC server
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
		inputType := protoreflect.FullName(strings.TrimPrefix(method.GetInputType(), "."))

		inputDesc, err := findMessageDescriptor(inputType)
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to get input message descriptor")
		}

		outputType := protoreflect.FullName(strings.TrimPrefix(method.GetOutputType(), "."))

		outputDesc, err := findMessageDescriptor(outputType)
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to get output message descriptor")
		}

		// Register method in the service manager
		s.serviceManager.GetMethodRegistry().RegisterMethod(
			serviceDesc.ServiceName,
			method.GetName(),
			method.GetClientStreaming(),
			method.GetServerStreaming(),
		)

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

func findMessageDescriptor(name protoreflect.FullName) (protoreflect.MessageDescriptor, error) {
	// Prefer GlobalTypes if populated.
	if msg, err := protoregistry.GlobalTypes.FindMessageByName(name); err == nil {
		return msg.Descriptor(), nil
	}

	// Fallback to files registry.
	desc, err := protoregistry.GlobalFiles.FindDescriptorByName(name)
	if err != nil {
		return nil, err
	}

	msgDesc, ok := desc.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, errors.Newf("descriptor %s is not a message", name)
	}

	return msgDesc, nil
}

func (s *GRPCServer) createGrpcMocker(
	serviceDesc *grpc.ServiceDesc,
	svc *descriptorpb.ServiceDescriptorProto,
	method *descriptorpb.MethodDescriptorProto,
	inputDesc, outputDesc protoreflect.MessageDescriptor,
) *grpcMocker {
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

		errorFormatter: s.errorFormatter,
		pluginRegistry: s.pluginRegistry,
	}
}

func (s *GRPCServer) startHealthCheckRoutine(ctx context.Context) {
	logger := zerolog.Ctx(ctx)

	// Start background routine to handle waiter completion
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error().
					Interface("panic", r).
					Msg("Panic recovered in health check routine")
			}
		}()

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

	// Use more robust marshalling options for better JSON output
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

func isNilInterface(v any) bool {
	if v == nil {
		return true
	}

	rv := reflect.ValueOf(v)
	//nolint:exhaustive
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return rv.IsNil()
	default:
		return false
	}
}

func toLogArray(items ...any) *zerolog.Array {
	arr := zerolog.Arr()

	for _, item := range items {
		// Skip nil items (they shouldn't be in the array anymore, but just in case)
		if item == nil || isNilInterface(item) {
			continue
		}

		if value := protoToJSON(item); value != nil {
			arr = arr.RawJSON(value)
		} else {
			// Fallback to string representation for non-proto messages
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
	// Only log non-nil messages
	if m != nil && !isNilInterface(m) {
		s.responses = append(s.responses, m)
	}

	return s.ServerStream.SendMsg(m)
}

func (s *loggingStream) RecvMsg(m any) error {
	// Only log non-nil messages
	if m != nil && !isNilInterface(m) {
		s.requests = append(s.requests, m)
	}

	return s.ServerStream.RecvMsg(m)
}

// sendBidiResponses sends response(s) for bidirectional streaming.
//

func (m *grpcMocker) sendBidiResponses(stream grpc.ServerStream, output stuber.Output, stub *stuber.Stub, messageIndex int) error {
	// For bidirectional streaming, send all elements from Stream if available.
	if len(output.Stream) > 0 {
		return m.sendStreamResponses(stream, output, stub, messageIndex)
	}

	// Fallback to Data if no Stream available.
	outputMsg, err := m.newOutputMessage(output.Data)
	if err != nil {
		return errors.Wrap(err, "failed to convert response to dynamic message")
	}

	return sendStreamMessage(stream, outputMsg)
}

// sendStreamResponses sends responses from output stream.
//
//nolint:cyclop,nestif
func (m *grpcMocker) sendStreamResponses(stream grpc.ServerStream, output stuber.Output, stub *stuber.Stub, messageIndex int) error {
	// For stubs with Inputs (multiple input messages), send one response per input message
	if stub.IsClientStream() {
		// If only one element is provided in the stream, treat it as a template to be used for every message
		// The MessageIndex is already applied in handleBidiStream during template processing
		var idx int

		if len(output.Stream) == 0 {
			return nil
		}

		if len(output.Stream) == 1 {
			idx = 0
		} else {
			if messageIndex < 0 || messageIndex >= len(output.Stream) {
				return nil
			}

			idx = messageIndex
		}

		streamData, ok := output.Stream[idx].(map[string]any)
		if !ok {
			return nil
		}

		outputMsg, err := m.newOutputMessage(streamData)
		if err != nil {
			return errors.Wrap(err, "failed to convert response to dynamic message")
		}

		return sendStreamMessage(stream, outputMsg)
	}

	// For stubs with Input (single input message), send all elements from the stream array
	for _, streamElement := range output.Stream {
		if streamData, ok := streamElement.(map[string]any); ok {
			outputMsg, err := m.newOutputMessage(streamData)
			if err != nil {
				return errors.Wrap(err, "failed to convert response to dynamic message")
			}

			if err := sendStreamMessage(stream, outputMsg); err != nil {
				return err //nolint:wrapcheck
			}
		}
	}

	return nil
}
