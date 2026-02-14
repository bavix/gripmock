package app

//nolint:revive
import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/goccy/go-json"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
	reflectiongrpc "google.golang.org/grpc/reflection/grpc_reflection_v1"
	reflectiongrpcv1alpha "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/domain/history"
	protoloc "github.com/bavix/gripmock/v3/internal/domain/proto"
	"github.com/bavix/gripmock/v3/internal/domain/protoset"
	"github.com/bavix/gripmock/v3/internal/infra/grpccontext"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
	"github.com/bavix/gripmock/v3/internal/infra/types"
)

// excludedHeaders contains headers that should be excluded from stub matching.
// Map for O(1) lookup in hot path.
//
//nolint:gochecknoglobals
var excludedHeaders = map[string]struct{}{
	":authority":           {},
	"content-type":         {},
	"grpc-accept-encoding": {},
	"user-agent":           {},
	"accept-encoding":      {},
}

const (
	sessionHeaderKey = "x-gripmock-session" // gRPC metadata keys are lowercase

	// High-load gRPC server tuning.
	keepaliveMaxIdle     = 5 * time.Minute
	keepaliveMaxAge      = 30 * time.Minute
	keepaliveMaxAgeGrace = 5 * time.Second
	keepaliveTime        = 30 * time.Second
	keepaliveTimeout     = 10 * time.Second
	keepaliveMinTime     = 10 * time.Second
	maxConcurrentStreams = 100
	maxLoggingStreamMsgs = 32
	minStreamWorkers     = 4
)

const jsonBufferInitialCap = 4096

var (
	//nolint:gochecknoglobals
	runtimeNumStreamWorkers = max(runtime.NumCPU(), minStreamWorkers)
	//nolint:gochecknoglobals
	jsonBufferPool = sync.Pool{
		New: func() any {
			return bytes.NewBuffer(make([]byte, 0, jsonBufferInitialCap))
		},
	}
)

func sessionFromMetadata(md metadata.MD) string {
	if v := md.Get(sessionHeaderKey); len(v) > 0 && v[0] != "" {
		return v[0]
	}

	return ""
}

// processHeaders converts metadata to headers map, excluding specified headers.
func processHeaders(md metadata.MD) map[string]any {
	if len(md) == 0 {
		return nil
	}

	headers := make(map[string]any, len(md))

	for k, v := range md {
		if _, excluded := excludedHeaders[k]; !excluded {
			headers[k] = strings.Join(v, ";")
		}
	}

	return headers
}

func sendStreamMessage(stream grpc.ServerStream, msg *dynamicpb.Message) error {
	if err := stream.SendMsg(msg); err != nil {
		return errors.Wrap(err, "failed to send response")
	}

	return nil
}

func receiveStreamMessage(stream grpc.ServerStream, msg *dynamicpb.Message) error {
	err := stream.RecvMsg(msg)
	if err != nil {
		return errors.Wrap(err, "failed to receive message")
	}

	return nil
}

const serviceReflection = "grpc.reflection.v1.ServerReflection"

type GRPCServer struct {
	network     string
	address     string
	params      *protoloc.Arguments
	budgerigar  *stuber.Budgerigar
	waiter      Extender
	recorder    history.Recorder
	healthcheck *health.Server
}

type grpcMocker struct {
	budgerigar     *stuber.Budgerigar
	templateEngine *template.Engine
	recorder       history.Recorder

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
		Input:   []map[string]any{convertToMap(msg)},
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		query.Headers = processHeaders(md)
		query.Session = sessionFromMetadata(md)
	}

	return query
}

func (m *grpcMocker) newQueryBidi(ctx context.Context) stuber.QueryBidi {
	query := stuber.QueryBidi{
		Service: m.fullServiceName,
		Method:  m.methodName,
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		query.Headers = processHeaders(md)
		query.Session = sessionFromMetadata(md)
	}

	return query
}

func convertToMap(msg proto.Message) map[string]any {
	if msg == nil {
		return nil
	}

	message := msg.ProtoReflect()
	desc := message.Descriptor()
	result := make(map[string]any, desc.Fields().Len())

	// Iterate over descriptor fields, not message.Range: Range only visits populated fields.
	// In proto3, scalars with default values (e.g. 0.0) are not "populated", so Range skips them.
	// We need all fields including defaults for stub matching.
	for i := range desc.Fields().Len() {
		fd := desc.Fields().Get(i)

		if fd.Cardinality() == protoreflect.Repeated && !message.Has(fd) {
			continue
		}

		fieldName := string(fd.Name())
		result[fieldName] = convertValue(fd, message.Get(fd))
	}

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
		return float64(value.Float())
	case protoreflect.DoubleKind:
		return value.Float()
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

func (m *grpcMocker) delay(ctx context.Context, delayDur types.Duration) {
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

//nolint:nestif,cyclop
func (m *grpcMocker) handleServerStream(stream grpc.ServerStream) error {
	inputMsg := dynamicpb.NewMessage(m.inputDesc)

	err := stream.RecvMsg(inputMsg)
	if errors.Is(err, io.EOF) {
		return nil
	}

	if err != nil {
		return errors.Wrap(err, "failed to receive message")
	}

	requestTime := time.Now()

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

	if found.IsServerStream() {
		if len(found.Output.Stream) > 0 {
			if err := m.handleArrayStreamData(stream, found, inputMsg, requestTime); err != nil {
				return err
			}

			if err := m.handleOutputError(stream.Context(), stream, found.Output); err != nil { //nolint:wrapcheck
				return err
			}

			return nil
		}

		if err := m.handleOutputError(stream.Context(), stream, found.Output); err != nil { //nolint:wrapcheck
			return err
		}
	}

	return m.handleNonArrayStreamData(stream, found)
}

func (m *grpcMocker) handleArrayStreamData(
	stream grpc.ServerStream,
	found *stuber.Stub,
	inputMsg *dynamicpb.Message,
	requestTime time.Time,
) error {
	done := stream.Context().Done()

	for i, streamData := range found.Output.Stream {
		select {
		case <-done:
			return stream.Context().Err()
		default:
		}

		outputData, ok := streamData.(map[string]any)
		if !ok {
			return status.Errorf(
				codes.Internal,
				"invalid data format in stream array at index %d: got %T, expected map[string]any",
				i, streamData,
			)
		}

		m.delay(stream.Context(), found.Output.Delay)

		outputDataCopy := deepCopyMapAny(outputData)
		requestData := convertToMap(inputMsg)

		headers := make(map[string]any)
		if md, ok := metadata.FromIncomingContext(stream.Context()); ok {
			headers = processHeaders(md)
		}

		templateData := template.Data{
			Request:      requestData,
			Headers:      headers,
			MessageIndex: i,
			RequestTime:  requestTime,
			Timestamp:    requestTime,
			State:        make(map[string]any),
			Requests:     []any{requestData},
			StubID:       found.ID.String(),
			RequestID:    found.ID.String(),
		}
		if err := m.templateEngine.ProcessMap(outputDataCopy, templateData); err != nil {
			return errors.Wrap(err, "failed to process dynamic templates")
		}

		outputMsg, err := m.newOutputMessage(outputDataCopy)
		if err != nil {
			return errors.Wrap(err, "failed to convert response to dynamic message")
		}

		if err := sendStreamMessage(stream, outputMsg); err != nil {
			return err
		}
	}

	return nil
}

//nolint:cyclop
func (m *grpcMocker) handleNonArrayStreamData(stream grpc.ServerStream, found *stuber.Stub) error {
	if err := m.handleOutputError(stream.Context(), stream, found.Output); err != nil {
		return err
	}

	done := stream.Context().Done()

	for {
		select {
		case <-done:
			return stream.Context().Err()
		default:
		}

		m.delay(stream.Context(), found.Output.Delay)

		outputDataCopy := deepCopyMapAny(found.Output.Data)

		inputMsg := dynamicpb.NewMessage(m.inputDesc)
		if err := stream.RecvMsg(inputMsg); err == nil {
			requestTime := time.Now()
			requestData := convertToMap(inputMsg)

			headers := make(map[string]any)
			if md, ok := metadata.FromIncomingContext(stream.Context()); ok {
				headers = processHeaders(md)
			}

			templateData := template.Data{
				Request:      requestData,
				Headers:      headers,
				MessageIndex: 0,
				RequestTime:  requestTime,
				Timestamp:    requestTime,
				State:        make(map[string]any),
				Requests:     []any{requestData},
				StubID:       found.ID.String(),
				RequestID:    found.ID.String(),
			}
			if err := m.templateEngine.ProcessMap(outputDataCopy, templateData); err != nil {
				return errors.Wrap(err, "failed to process dynamic templates")
			}
		}

		outputMsg, err := m.newOutputMessage(outputDataCopy)
		if err != nil {
			return errors.Wrap(err, "failed to convert response to dynamic message")
		}

		if err := sendStreamMessage(stream, outputMsg); err != nil {
			return err //nolint:wrapcheck
		}

		if err := stream.RecvMsg(nil); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			return errors.Wrap(err, "failed to receive message")
		}
	}
}

func (m *grpcMocker) newOutputMessage(data map[string]any) (*dynamicpb.Message, error) {
	pooled, _ := jsonBufferPool.Get().(*bytes.Buffer)
	if pooled == nil {
		pooled = bytes.NewBuffer(make([]byte, 0, jsonBufferInitialCap))
	}

	pooled.Reset()

	defer func() {
		pooled.Reset()
		jsonBufferPool.Put(pooled)
	}()

	enc := json.NewEncoder(pooled)
	if err := enc.Encode(data); err != nil {
		return nil, fmt.Errorf("failed to marshal map to JSON: %w", err)
	}

	msg := dynamicpb.NewMessage(m.outputDesc)

	err := protojson.Unmarshal(pooled.Bytes(), msg)
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

//nolint:cyclop,funlen
func (m *grpcMocker) handleUnary(ctx context.Context, req *dynamicpb.Message) (*dynamicpb.Message, error) {
	requestTime := time.Now()

	query := m.newQuery(ctx, req)

	result, err := m.budgerigar.FindByQuery(query)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	found := result.Found()
	if found == nil {
		errorFormatter := NewErrorFormatter()

		return nil, status.Error(codes.NotFound, errorFormatter.FormatStubNotFoundError(query, result).Error())
	}

	m.delay(ctx, found.Output.Delay)

	outputToUse := found.Output
	requestData := convertToMap(req)

	headers := make(map[string]any)
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		headers = processHeaders(md)
	}

	templateData := template.Data{
		Request:      requestData,
		Headers:      headers,
		MessageIndex: 0,
		RequestTime:  requestTime,
		Timestamp:    requestTime,
		State:        make(map[string]any),
		Requests:     []any{requestData},
		StubID:       found.ID.String(),
		RequestID:    found.ID.String(),
	}

	outputDataCopy := deepCopyMapAny(outputToUse.Data)

	if err := m.templateEngine.ProcessMap(outputDataCopy, templateData); err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("failed to process dynamic templates")

		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to process dynamic templates: %v", err))
	}

	outputToUse.Data = outputDataCopy

	if template.HasTemplatesInHeaders(outputToUse.Headers) {
		headersCopy := deepCopyStringMap(outputToUse.Headers)
		if err := m.templateEngine.ProcessHeaders(headersCopy, templateData); err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to process header templates: %v", err))
		}

		outputToUse.Headers = headersCopy
	}

	if outputToUse.Error != "" && template.IsTemplateString(outputToUse.Error) {
		errorStr, err := m.templateEngine.ProcessError(outputToUse.Error, templateData)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to process error template: %v", err))
		}

		outputToUse.Error = errorStr
	}

	if err := m.setResponseHeadersAny(ctx, nil, outputToUse.Headers); err != nil {
		return nil, err //nolint:wrapcheck
	}

	var sess string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		sess = sessionFromMetadata(md)
	}

	if err := m.handleOutputError(ctx, nil, outputToUse); err != nil {
		if m.recorder != nil {
			m.recorder.Record(history.CallRecord{
				Service:   m.fullServiceName,
				Method:    m.methodName,
				Session:   sess,
				Request:   requestData,
				Error:     outputToUse.Error,
				StubID:    found.ID.String(),
				Timestamp: requestTime,
			})
		}

		return nil, err //nolint:wrapcheck
	}

	outputMsg, err := m.newOutputMessage(outputToUse.Data)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	if m.recorder != nil {
		m.recorder.Record(history.CallRecord{
			Service:   m.fullServiceName,
			Method:    m.methodName,
			Session:   sess,
			Request:   requestData,
			Response:  outputToUse.Data,
			StubID:    found.ID.String(),
			Timestamp: requestTime,
		})
	}

	return outputMsg, nil
}

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

func (m *grpcMocker) setResponseHeadersAny(ctx context.Context, stream grpc.ServerStream, headers map[string]string) error {
	mdResp, ok := buildResponseMetadata(headers)
	if !ok {
		return nil
	}

	if stream != nil {
		return stream.SetHeader(mdResp)
	}

	return grpc.SetHeader(ctx, mdResp)
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

func (m *grpcMocker) tryV2API(messages []map[string]any, md metadata.MD) (*stuber.Result, error) {
	query := stuber.Query{
		Service: m.fullServiceName,
		Method:  m.methodName,
		Input:   messages,
	}

	if len(md) > 0 {
		query.Headers = processHeaders(md)
		query.Session = sessionFromMetadata(md)
	}

	return m.budgerigar.FindByQuery(query)
}

func (m *grpcMocker) tryV1APIFallback(messages []map[string]any, md metadata.MD) (*stuber.Result, error) {
	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]

		query := stuber.Query{
			Service: m.fullServiceName,
			Method:  m.methodName,
			Input:   []map[string]any{message},
		}

		if len(md) > 0 {
			query.Headers = processHeaders(md)
			query.Session = sessionFromMetadata(md)
		}

		result, foundErr := m.budgerigar.FindByQuery(query)
		if foundErr == nil && result != nil && result.Found() != nil {
			return result, nil
		}
	}

	return nil, status.Errorf(codes.NotFound, "failed to find response for client stream")
}

func (m *grpcMocker) handleClientStream(stream grpc.ServerStream) error {
	requestTime := time.Now()

	messages, err := m.collectClientMessages(stream)
	if err != nil {
		return err
	}

	found, err := m.tryFindStub(stream, messages)
	if err != nil {
		return err
	}

	return m.sendClientStreamResponse(stream, found, messages, requestTime)
}

const clientMessagesInitCap = 16

func (m *grpcMocker) collectClientMessages(stream grpc.ServerStream) ([]map[string]any, error) {
	messages := make([]map[string]any, 0, clientMessagesInitCap)

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

func (m *grpcMocker) tryFindStub(stream grpc.ServerStream, messages []map[string]any) (*stuber.Stub, error) {
	md, _ := metadata.FromIncomingContext(stream.Context())

	result, foundErr := m.tryV2API(messages, md)

	if foundErr != nil || result == nil || result.Found() == nil {
		result, foundErr = m.tryV1APIFallback(messages, md)
	}

	if foundErr != nil || result == nil || result.Found() == nil {
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

func (m *grpcMocker) sendClientStreamResponse(
	stream grpc.ServerStream,
	found *stuber.Stub,
	messages []map[string]any,
	requestTime time.Time,
) error {
	m.delay(stream.Context(), found.Output.Delay)

	if err := m.handleOutputError(stream.Context(), stream, found.Output); err != nil { //nolint:wrapcheck
		return err
	}

	if err := m.setResponseHeadersAny(stream.Context(), stream, found.Output.Headers); err != nil {
		return errors.Wrap(err, "failed to set headers")
	}

	outputDataCopy := deepCopyMapAny(found.Output.Data)

	headers := make(map[string]any)
	if md, ok := metadata.FromIncomingContext(stream.Context()); ok {
		headers = processHeaders(md)
	}

	requestsAny := make([]any, len(messages))
	for i, msg := range messages {
		requestsAny[i] = msg
	}

	templateData := template.Data{
		Request:      nil,
		Headers:      headers,
		MessageIndex: 0,
		RequestTime:  requestTime,
		Timestamp:    requestTime,
		State:        make(map[string]any),
		Requests:     requestsAny,
		StubID:       found.ID.String(),
		RequestID:    found.ID.String(),
	}
	if err := m.templateEngine.ProcessMap(outputDataCopy, templateData); err != nil {
		return errors.Wrap(err, "failed to process dynamic templates")
	}

	outputMsg, err := m.newOutputMessage(outputDataCopy)
	if err != nil {
		return errors.Wrap(err, "failed to convert response to dynamic message")
	}

	return stream.SendMsg(outputMsg)
}

func (m *grpcMocker) handleBidiStream(stream grpc.ServerStream) error {
	queryBidi := m.newQueryBidi(stream.Context())

	bidiResult, err := m.budgerigar.FindByQueryBidi(queryBidi)
	if err != nil {
		return errors.Wrap(err, "failed to initialize bidirectional streaming session")
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

		if err := m.processBidiStreamMessage(stream, bidiResult, inputMsg); err != nil {
			return err
		}
	}
}

func (m *grpcMocker) processBidiStreamMessage(
	stream grpc.ServerStream,
	bidiResult *stuber.BidiResult,
	inputMsg *dynamicpb.Message,
) error {
	requestTime := time.Now()

	stub, err := bidiResult.Next(convertToMap(inputMsg))
	if err != nil {
		return errors.Wrap(err, "failed to process bidirectional message")
	}

	m.delay(stream.Context(), stub.Output.Delay)

	requestData := convertToMap(inputMsg)

	headers := make(map[string]any)
	if md, ok := metadata.FromIncomingContext(stream.Context()); ok {
		headers = processHeaders(md)
	}

	templateData := template.Data{
		Request:      requestData,
		Headers:      headers,
		MessageIndex: bidiResult.GetMessageIndex(),
		RequestTime:  requestTime,
		Timestamp:    requestTime,
		State:        make(map[string]any),
		Requests:     []any{requestData},
		StubID:       stub.ID.String(),
		RequestID:    stub.ID.String(),
	}

	outputToUse, err := m.prepareBidiOutput(stub, templateData)
	if err != nil {
		return err
	}

	if bidiResult.GetMessageIndex() == 0 {
		if err := m.setResponseHeadersAny(stream.Context(), stream, outputToUse.Headers); err != nil {
			return errors.Wrap(err, "failed to set headers")
		}
	}

	if err := m.handleOutputError(stream.Context(), stream, outputToUse); err != nil { //nolint:wrapcheck
		return err
	}

	return m.sendBidiResponses(stream, outputToUse, stub, bidiResult.GetMessageIndex(), requestTime)
}

func (m *grpcMocker) prepareBidiOutput(stub *stuber.Stub, templateData template.Data) (stuber.Output, error) {
	outputDataCopy := deepCopyMapAny(stub.Output.Data)
	if err := m.templateEngine.ProcessMap(outputDataCopy, templateData); err != nil {
		return stuber.Output{}, errors.Wrap(err, "failed to process dynamic templates")
	}

	headersCopy := deepCopyStringMap(stub.Output.Headers)
	if template.HasTemplatesInHeaders(headersCopy) {
		if err := m.templateEngine.ProcessHeaders(headersCopy, templateData); err != nil {
			return stuber.Output{}, errors.Wrap(err, "failed to process header templates")
		}
	}

	streamCopy := make([]any, len(stub.Output.Stream))
	for i, item := range stub.Output.Stream {
		if itemMap, ok := item.(map[string]any); ok {
			itemCopy := deepCopyMapAny(itemMap)
			if err := m.templateEngine.ProcessMap(itemCopy, templateData); err != nil {
				return stuber.Output{}, errors.Wrap(err, "failed to process stream template")
			}

			streamCopy[i] = itemCopy
		} else {
			streamCopy[i] = item
		}
	}

	outputToUse := stuber.Output{
		Data:    outputDataCopy,
		Stream:  streamCopy,
		Headers: headersCopy,
		Error:   stub.Output.Error,
		Delay:   stub.Output.Delay,
	}

	if outputToUse.Error != "" && template.IsTemplateString(outputToUse.Error) {
		errorStr, err := m.templateEngine.ProcessError(outputToUse.Error, templateData)
		if err != nil {
			return stuber.Output{}, errors.Wrap(err, "failed to process error template")
		}

		outputToUse.Error = errorStr
	}

	return outputToUse, nil
}

func NewGRPCServer(
	network, address string,
	params *protoloc.Arguments,
	budgerigar *stuber.Budgerigar,
	waiter Extender,
	recorder history.Recorder,
) *GRPCServer {
	return &GRPCServer{
		network:    network,
		address:    address,
		params:     params,
		budgerigar: budgerigar,
		waiter:     waiter,
		recorder:   recorder,
	}
}

func (s *GRPCServer) Build(ctx context.Context) (*grpc.Server, error) {
	descriptors, err := protoset.Build(ctx, s.params.Imports(), s.params.ProtoPath())
	if err != nil {
		return nil, errors.Wrap(err, "failed to build descriptors")
	}

	server := s.createServer(ctx)
	s.setupHealthCheck(server, nil)
	s.registerServices(ctx, server, descriptors, nil)
	s.startHealthCheckRoutine(ctx)

	return server, nil
}

// BuildFromDescriptorSet creates a gRPC server from a pre-built FileDescriptorSet.
// Used by the SDK for embedded mode. Does not use GlobalFiles.
// If recorder is non-nil, all gRPC calls are recorded for History/Verify.
func BuildFromDescriptorSet(
	ctx context.Context,
	fds *descriptorpb.FileDescriptorSet,
	budgerigar *stuber.Budgerigar,
	waiter Extender,
	recorder history.Recorder,
) (*grpc.Server, error) {
	reg, err := protodesc.NewFiles(fds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create files registry")
	}

	s := &GRPCServer{
		budgerigar: budgerigar,
		waiter:     waiter,
		recorder:   recorder,
	}
	server := s.createServer(ctx)
	s.setupHealthCheck(server, reg)
	s.registerServices(ctx, server, []*descriptorpb.FileDescriptorSet{fds}, reg)
	s.startHealthCheckRoutine(ctx)

	return server, nil
}

func (s *GRPCServer) createServer(ctx context.Context) *grpc.Server {
	logger := zerolog.Ctx(ctx)

	return grpc.NewServer(
		grpc.NumStreamWorkers(uint32(runtimeNumStreamWorkers)), //nolint:gosec
		grpc.MaxConcurrentStreams(maxConcurrentStreams),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     keepaliveMaxIdle,
			MaxConnectionAge:      keepaliveMaxAge,
			MaxConnectionAgeGrace: keepaliveMaxAgeGrace,
			Time:                  keepaliveTime,
			Timeout:               keepaliveTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             keepaliveMinTime,
			PermitWithoutStream: true,
		}),
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

func (s *GRPCServer) setupHealthCheck(server *grpc.Server, descResolver *protoregistry.Files) {
	healthcheck := health.NewServer()
	healthcheck.SetServingStatus("gripmock", healthgrpc.HealthCheckResponse_NOT_SERVING)
	healthgrpc.RegisterHealthServer(server, healthcheck)

	if descResolver != nil {
		reflectionSvr := reflection.NewServerV1(reflection.ServerOptions{
			Services:           server,
			DescriptorResolver: descResolver,
		})
		reflectiongrpc.RegisterServerReflectionServer(server, reflectionSvr)
		reflectiongrpcv1alpha.RegisterServerReflectionServer(server, reflection.NewServer(reflection.ServerOptions{
			Services:           server,
			DescriptorResolver: descResolver,
		}))
	} else {
		reflection.Register(server)
	}

	s.healthcheck = healthcheck
}

//nolint:lll
func (s *GRPCServer) registerServices(ctx context.Context, server *grpc.Server, descriptors []*descriptorpb.FileDescriptorSet, reg *protoregistry.Files) {
	logger := zerolog.Ctx(ctx)

	for _, descriptor := range descriptors {
		for _, file := range descriptor.GetFile() {
			for _, svc := range file.GetService() {
				serviceDesc := s.createServiceDesc(file, svc)
				s.registerServiceMethods(ctx, &serviceDesc, svc, reg)
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

//nolint:lll
func (s *GRPCServer) registerServiceMethods(ctx context.Context, serviceDesc *grpc.ServiceDesc, svc *descriptorpb.ServiceDescriptorProto, reg *protoregistry.Files) {
	logger := zerolog.Ctx(ctx)

	for _, method := range svc.GetMethod() {
		inputDesc, err := getMessageDescriptor(reg, method.GetInputType())
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to get input message descriptor")
		}

		outputDesc, err := getMessageDescriptor(reg, method.GetOutputType())
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to get output message descriptor")
		}

		m := s.createGrpcMocker(ctx, serviceDesc, svc, method, inputDesc, outputDesc)

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

func (s *GRPCServer) createGrpcMocker(
	ctx context.Context,
	serviceDesc *grpc.ServiceDesc,
	svc *descriptorpb.ServiceDescriptorProto,
	method *descriptorpb.MethodDescriptorProto,
	inputDesc, outputDesc protoreflect.MessageDescriptor,
) *grpcMocker {
	templateEngine := template.New(ctx, nil)

	return &grpcMocker{
		budgerigar:     s.budgerigar,
		templateEngine: templateEngine,
		recorder:       s.recorder,

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

func getServiceName(file *descriptorpb.FileDescriptorProto, svc *descriptorpb.ServiceDescriptorProto) string {
	if file.GetPackage() != "" {
		return fmt.Sprintf("%s.%s", file.GetPackage(), svc.GetName())
	}

	return svc.GetName()
}

//nolint:ireturn
func getMessageDescriptor(reg *protoregistry.Files, messageType string) (protoreflect.MessageDescriptor, error) {
	if reg == nil {
		reg = protoregistry.GlobalFiles
	}

	msgName := protoreflect.FullName(strings.TrimPrefix(messageType, "."))

	desc, err := reg.FindDescriptorByName(msgName)
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

func (m *grpcMocker) sendBidiResponses(
	stream grpc.ServerStream,
	output stuber.Output,
	stub *stuber.Stub,
	messageIndex int,
	requestTime time.Time,
) error {
	if len(output.Stream) > 0 {
		return m.sendStreamResponses(stream, output, stub, messageIndex, requestTime)
	}

	outputDataCopy := deepCopyMapAny(output.Data)

	headers := make(map[string]any)
	if md, ok := metadata.FromIncomingContext(stream.Context()); ok {
		headers = processHeaders(md)
	}

	templateData := template.Data{
		Request:      nil,
		Headers:      headers,
		MessageIndex: messageIndex,
		RequestTime:  requestTime,
		Timestamp:    requestTime,
		State:        make(map[string]any),
		Requests:     []any{},
		StubID:       stub.ID.String(),
		RequestID:    stub.ID.String(),
	}
	if err := m.templateEngine.ProcessMap(outputDataCopy, templateData); err != nil {
		return errors.Wrap(err, "failed to process dynamic templates")
	}

	outputMsg, err := m.newOutputMessage(outputDataCopy)
	if err != nil {
		return errors.Wrap(err, "failed to convert response to dynamic message")
	}

	return sendStreamMessage(stream, outputMsg)
}

//nolint:cyclop,funlen,nestif
func (m *grpcMocker) sendStreamResponses(
	stream grpc.ServerStream,
	output stuber.Output,
	stub *stuber.Stub,
	messageIndex int,
	requestTime time.Time,
) error {
	if stub.IsClientStream() {
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

		streamDataCopy := deepCopyMapAny(streamData)

		headers := make(map[string]any)
		if md, ok := metadata.FromIncomingContext(stream.Context()); ok {
			headers = processHeaders(md)
		}

		templateData := template.Data{
			Request:      nil,
			Headers:      headers,
			MessageIndex: messageIndex,
			RequestTime:  requestTime,
			Timestamp:    requestTime,
			State:        make(map[string]any),
			Requests:     []any{},
			StubID:       stub.ID.String(),
			RequestID:    stub.ID.String(),
		}
		if err := m.templateEngine.ProcessMap(streamDataCopy, templateData); err != nil {
			return errors.Wrap(err, "failed to process dynamic templates")
		}

		outputMsg, err := m.newOutputMessage(streamDataCopy)
		if err != nil {
			return errors.Wrap(err, "failed to convert response to dynamic message")
		}

		return sendStreamMessage(stream, outputMsg)
	}

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
