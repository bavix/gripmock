package app

//nolint:revive
import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	stderrors "errors"
	"fmt"
	"io"
	"maps"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
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

	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	"github.com/bavix/gripmock/v3/internal/domain/history"
	protoloc "github.com/bavix/gripmock/v3/internal/domain/proto"
	protosetdom "github.com/bavix/gripmock/v3/internal/domain/protoset"
	"github.com/bavix/gripmock/v3/internal/infra/grpccontext"
	protosetinfra "github.com/bavix/gripmock/v3/internal/infra/protoset"
	"github.com/bavix/gripmock/v3/internal/infra/proxyroutes"
	"github.com/bavix/gripmock/v3/internal/infra/session"
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
	unknownValue     = "unknown"

	// High-load gRPC server tuning.
	keepaliveMaxIdle     = 5 * time.Minute
	keepaliveMaxAge      = 30 * time.Minute
	keepaliveMaxAgeGrace = 5 * time.Second
	keepaliveTime        = 30 * time.Second
	keepaliveTimeout     = 10 * time.Second
	keepaliveMinTime     = 10 * time.Second
	maxConcurrentStreams = 100
	maxLoggingStreamMsgs = 32
	maxHistoryStreamMsgs = 100
	minStreamWorkers     = 4
)

const (
	jsonBufferInitialCap            = 4096
	bidiRecordingStreamInitCap      = 16
	bidiRecordingStreamResponsesCap = 16
)

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
	for _, v := range md.Get(sessionHeaderKey) {
		if sessionID := strings.TrimSpace(v); sessionID != "" {
			session.Touch(sessionID)

			return sessionID
		}
	}

	return ""
}

func sessionFromContext(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		return sessionFromMetadata(md)
	}

	return ""
}

type bidiRecordingStream struct {
	grpc.ServerStream

	requests  []map[string]any
	responses []map[string]any
	stubID    uuid.UUID
	maxItems  int
}

func (s *bidiRecordingStream) RecvMsg(m any) error {
	err := s.ServerStream.RecvMsg(m)
	if err != nil {
		return err
	}

	if msgMap := protoToMap(m); msgMap != nil && len(s.requests) < s.maxItems {
		s.requests = append(s.requests, msgMap)
	}

	return nil
}

func (s *bidiRecordingStream) SendMsg(m any) error {
	err := s.ServerStream.SendMsg(m)
	if err != nil {
		return err
	}

	if msgMap := protoToMap(m); msgMap != nil && len(s.responses) < s.maxItems {
		s.responses = append(s.responses, msgMap)
	}

	return nil
}

func (s *bidiRecordingStream) getRequests() []map[string]any { return s.requests }

func (s *bidiRecordingStream) getResponses() []map[string]any { return s.responses }

func (s *bidiRecordingStream) setStubID(id uuid.UUID) { s.stubID = id }

func (s *bidiRecordingStream) getStubID() uuid.UUID { return s.stubID }

func (m *grpcMocker) recordCall(
	ctx context.Context,
	stubID uuid.UUID,
	code uint32,
	timestamp time.Time,
	requests []map[string]any,
	responses []map[string]any,
	errMsg string,
) {
	if m.recorder == nil || len(requests) == 0 {
		return
	}

	if responses == nil {
		responses = []map[string]any{}
	}

	rec := history.CallRecord{
		Service:   m.fullServiceName,
		Method:    m.methodName,
		Session:   sessionFromContext(ctx),
		Requests:  requests,
		Responses: responses,
		Error:     errMsg,
		Code:      code,
		StubID:    stubID,
		Timestamp: timestamp,
	}

	if len(requests) > 0 {
		rec.Request = requests[0]
	}

	if len(responses) > 0 {
		rec.Response = responses[0]
	}

	m.recorder.Record(rec)
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
	network      string
	address      string
	params       *protoloc.Arguments
	budgerigar   *stuber.Budgerigar
	healthState  stuber.Aliveness
	waiter       Extender
	recorder     history.Recorder
	descriptors  *descriptors.Registry
	remoteClient protosetdom.RemoteClient
	tlsConfig    *tls.Config
	proxies      *proxyroutes.Registry
	otelEnabled  bool
	validator    *validator.Validate
}

type grpcMocker struct {
	budgerigar         *stuber.Budgerigar
	templateEngine     *template.Engine
	errorFormatter     *ErrorFormatter
	recorder           history.Recorder
	descriptorResolver protodesc.Resolver
	proxies            *proxyroutes.Registry
	validator          *validator.Validate

	inputDesc  protoreflect.MessageDescriptor
	outputDesc protoreflect.MessageDescriptor

	fullServiceName string
	serviceName     string
	methodName      string
	fullMethod      string

	serverStream bool
	clientStream bool

	strictServiceMatch bool
}

//nolint:cyclop
func (m *grpcMocker) streamHandler(srv any, stream grpc.ServerStream) error {
	info := &grpc.StreamServerInfo{
		FullMethod:     m.fullMethod,
		IsClientStream: m.clientStream,
		IsServerStream: m.serverStream,
	}

	handler := func(_ any, stream grpc.ServerStream) error {
		route := m.proxyRoute()
		behavior := newProxyBehavior(route)

		if behavior != nil && behavior.proxyOnly() {
			return m.proxyStream(stream, route, false)
		}

		var err error

		switch {
		case m.serverStream && !m.clientStream:
			err = m.handleServerStream(stream)
		case !m.serverStream && m.clientStream:
			err = m.handleClientStream(stream)
		case m.serverStream && m.clientStream:
			err = m.handleBidiStream(stream)
		default:
			err = status.Errorf(codes.Unimplemented, "Unknown stream type")
		}

		if behavior == nil {
			return err
		}

		if !behavior.canFallback(err) {
			return err
		}

		if fallbackErr, ok := stderrors.AsType[*serverStreamFallbackError](err); ok {
			return m.proxyServerStreamWithRequest(stream, route, fallbackErr.request, behavior.captureMiss())
		}

		if fallbackErr, ok := stderrors.AsType[*clientStreamFallbackError](err); ok {
			return m.proxyClientStreamWithRequests(stream, route, fallbackErr.requests, behavior.captureMiss())
		}

		if fallbackErr, ok := stderrors.AsType[*bidiStreamFallbackError](err); ok {
			return m.proxyBidiStreamWithRequests(stream, route, fallbackErr.requests, behavior.captureMiss())
		}

		return m.proxyStream(stream, route, behavior.captureMiss())
	}

	return grpc.StreamServerInterceptor(func(srv any, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, ss)
	})(srv, stream, info, handler)
}

func (m *grpcMocker) newQuery(ctx context.Context, msg *dynamicpb.Message) stuber.Query {
	query := stuber.Query{
		Service:       m.fullServiceName,
		Method:        m.methodName,
		StrictService: m.strictServiceMatch,
		Input:         []map[string]any{convertToMap(msg)},
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
		Service:       m.fullServiceName,
		Method:        m.methodName,
		StrictService: m.strictServiceMatch,
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

func (m *grpcMocker) delay(ctx context.Context, delayDur types.Duration) error {
	return delayResponse(ctx, delayDur)
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

	requestTime := time.Now()

	query := m.newQuery(stream.Context(), inputMsg)

	result, err := m.budgerigar.FindByQuery(query)

	result, err = m.ensureServerStreamResult(query, result, err)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return &serverStreamFallbackError{err: err, request: inputMsg}
		}

		return err
	}

	found := result.Found()

	if err := m.delay(stream.Context(), found.Output.Delay); err != nil {
		return err
	}

	outputToUse := found.Output
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

	if template.HasTemplatesInHeaders(outputToUse.Headers) {
		headersCopy := deepCopyStringMap(outputToUse.Headers)
		if err := m.templateEngine.ProcessHeaders(headersCopy, templateData); err != nil {
			return errors.Wrap(err, "failed to process header templates")
		}

		outputToUse.Headers = headersCopy
	}

	if err := m.setResponseHeadersAny(stream.Context(), stream, outputToUse.Headers); err != nil {
		return errors.Wrap(err, "failed to set headers")
	}

	m.applyEffects(stream.Context(), found, templateData)

	if found.Output.Stream != nil {
		if len(found.Output.Stream) > 0 {
			if err := m.handleArrayStreamData(stream, found, inputMsg, requestTime); err != nil {
				return err
			}

			if err := m.handleOutputError(stream.Context(), stream, outputToUse); err != nil { //nolint:wrapcheck
				return err
			}

			streamResponses := make([]map[string]any, len(found.Output.Stream))
			for i, item := range found.Output.Stream {
				if m, ok := item.(map[string]any); ok {
					streamResponses[i] = m
				}
			}

			m.recordCall(stream.Context(), found.ID, uint32(codes.OK), requestTime, []map[string]any{requestData}, streamResponses, "")

			return nil
		}

		if err := m.handleOutputError(stream.Context(), stream, outputToUse); err != nil { //nolint:wrapcheck
			return err
		}

		m.recordCall(
			stream.Context(),
			found.ID,
			uint32(codes.OK),
			requestTime,
			[]map[string]any{requestData},
			[]map[string]any{outputToUse.Data},
			"",
		)

		return nil
	}

	err = m.handleNonArrayStreamData(stream, found)
	if err != nil {
		return err
	}

	m.recordCall(
		stream.Context(),
		found.ID,
		uint32(codes.OK),
		requestTime,
		[]map[string]any{requestData},
		[]map[string]any{outputToUse.Data},
		"",
	)

	return nil
}

func (m *grpcMocker) ensureServerStreamResult(
	query stuber.Query,
	result *stuber.Result,
	err error,
) (*stuber.Result, error) {
	if err == nil && (result == nil || result.Found() != nil) {
		return result, nil
	}

	if result == nil {
		result = &stuber.Result{}
	}

	return nil, status.Error(codes.NotFound, m.errorFormatter.FormatStubNotFoundError(query, result).Error())
}

//nolint:funlen
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

		if err := m.delay(stream.Context(), found.Output.Delay); err != nil {
			return err
		}

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

//nolint:cyclop,funlen
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

		if err := m.delay(stream.Context(), found.Output.Delay); err != nil {
			return err
		}

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

	convertedData := convertMapNumericToStringNumber(data)

	enc := json.NewEncoder(pooled)
	if err := enc.Encode(convertedData); err != nil {
		return nil, fmt.Errorf("failed to marshal map to JSON: %w", err)
	}

	msg := dynamicpb.NewMessage(m.outputDesc)

	err := protojson.Unmarshal(pooled.Bytes(), msg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON into dynamic message: %w", err)
	}

	return msg, nil
}

func convertMapNumericToStringNumber(data map[string]any) map[string]any {
	result := make(map[string]any, len(data))
	for k, v := range data {
		result[k] = convertMapValue(v)
	}

	return result
}

func convertMapValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return convertMapNumericToStringNumber(val)
	case []any:
		return convertMapArray(val)
	case float64:
		return convertFloat64(val)
	case float32:
		return convertFloat64(float64(val))
	case int, int8, int16, int32, int64:
		return json.Number(strconv.FormatInt(toInt64(val), 10))
	case uint, uint8, uint16, uint32, uint64:
		return json.Number(strconv.FormatUint(toUint64(val), 10))
	default:
		return v
	}
}

func convertFloat64(f float64) json.Number {
	if isSafeInteger(f) {
		return json.Number(strconv.FormatInt(int64(f), 10))
	}

	return json.Number(strconv.FormatFloat(f, 'g', -1, 64))
}

func toInt64(v any) int64 {
	switch val := v.(type) {
	case int:
		return int64(val)
	case int8:
		return int64(val)
	case int16:
		return int64(val)
	case int32:
		return int64(val)
	case int64:
		return val
	default:
		return 0
	}
}

func toUint64(v any) uint64 {
	switch val := v.(type) {
	case uint:
		return uint64(val)
	case uint8:
		return uint64(val)
	case uint16:
		return uint64(val)
	case uint32:
		return uint64(val)
	case uint64:
		return val
	default:
		return 0
	}
}

func isSafeInteger(f float64) bool {
	return f == float64(int64(f))
}

func convertMapArray(arr []any) []any {
	result := make([]any, len(arr))
	for i, v := range arr {
		result[i] = convertMapValue(v)
	}

	return result
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
					return m.handleUnaryWithProxy(ctx, msg)
				}

				return nil, status.Errorf(codes.InvalidArgument, "expected *dynamicpb.Message, got %T", req)
			})
		}

		return m.handleUnaryWithProxy(ctx, req)
	}
}

func (m *grpcMocker) handleUnaryWithProxy(ctx context.Context, req *dynamicpb.Message) (*dynamicpb.Message, error) {
	route := m.proxyRoute()
	behavior := newProxyBehavior(route)

	if behavior == nil {
		return m.handleUnary(ctx, req)
	}

	if behavior.proxyOnly() {
		return m.proxyUnary(ctx, req, route, false)
	}

	if behavior.captureMiss() && m.captureShouldProxyUnaryByHeaders(ctx, req) {
		return m.proxyUnary(ctx, req, route, true)
	}

	resp, err := m.handleUnary(ctx, req)

	if _, ok := stderrors.AsType[*unaryStubMissError](err); !ok {
		return resp, err
	}

	return m.proxyUnary(ctx, req, route, behavior.captureMiss())
}

func (m *grpcMocker) captureShouldProxyUnaryByHeaders(ctx context.Context, req *dynamicpb.Message) bool {
	if !m.hasCaptureRequestHeaders(ctx) {
		return false
	}

	query := m.newQuery(ctx, req)

	report := m.budgerigar.InspectQuery(query)
	if report.MatchedStubID == nil {
		return true
	}

	found := m.budgerigar.FindByID(*report.MatchedStubID)
	if found == nil {
		return true
	}

	return found.Headers.Len() == 0
}

//nolint:cyclop,funlen
func (m *grpcMocker) handleUnary(ctx context.Context, req *dynamicpb.Message) (*dynamicpb.Message, error) {
	requestTime := time.Now()

	query := m.newQuery(ctx, req)

	result, err := m.budgerigar.FindByQuery(query)

	// Handle both error and nil result cases with unified error formatting
	if err != nil || (result != nil && result.Found() == nil) {
		errorFormatter := NewErrorFormatter()

		// Create empty result if we don't have one (error case)
		if result == nil {
			result = &stuber.Result{}
		}

		return nil, &unaryStubMissError{err: status.Error(codes.NotFound, errorFormatter.FormatStubNotFoundError(query, result).Error())}
	}

	found := result.Found()

	if err := m.delay(ctx, found.Output.Delay); err != nil {
		return nil, err
	}

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
		zerolog.Ctx(ctx).Err(err).Msg("failed to process dynamic templates")

		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to process dynamic templates: %v", err))
	}

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

	m.applyEffects(ctx, found, templateData)

	if err := m.handleOutputError(ctx, nil, outputToUse); err != nil {
		code := status.Code(err)
		m.recordCall(ctx, found.ID, uint32(code), requestTime, []map[string]any{requestData}, nil, err.Error())
		outputToUse.Error = err.Error()

		return nil, err //nolint:wrapcheck
	}

	outputMsg, err := m.newOutputMessage(outputDataCopy)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	m.recordCall(ctx, found.ID, uint32(codes.OK), requestTime, []map[string]any{requestData}, []map[string]any{outputDataCopy}, "")

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
	st, err := m.statusFromOutput(output)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	if st != nil {
		return st.Err()
	}

	return nil
}

func (m *grpcMocker) applyEffects(
	ctx context.Context,
	matched *stuber.Stub,
	templateData template.Data,
) {
	if len(matched.Effects) == 0 {
		return
	}

	prepared := make([]effectOperation, 0, len(matched.Effects))

	for i, effect := range matched.Effects {
		op, err := m.prepareEffect(effect, templateData, matched.Session)
		if err != nil {
			zerolog.Ctx(ctx).Err(err).
				Str("stub_id", matched.ID.String()).
				Int("effect_index", i).
				Str("effect_action", effect.Action).
				Msg("failed to prepare effect; skip all effects for request")

			return
		}

		prepared = append(prepared, op)
	}

	for i, op := range prepared {
		if err := m.applyEffectOperation(op); err != nil {
			zerolog.Ctx(ctx).Err(err).
				Str("stub_id", matched.ID.String()).
				Int("effect_index", i).
				Str("effect_action", op.action).
				Msg("failed to apply prepared effect")
		}
	}
}

type effectOperation struct {
	action        string
	upsertStub    *stuber.Stub
	deleteID      uuid.UUID
	parentSession string
}

func (m *grpcMocker) prepareEffect(
	effect stuber.Effect,
	templateData template.Data,
	parentSession string,
) (effectOperation, error) {
	switch effect.Action {
	case stuber.EffectActionUpsert:
		upsert, err := m.prepareUpsertEffect(effect, templateData, parentSession)
		if err != nil {
			return effectOperation{}, err
		}

		return effectOperation{action: effect.Action, upsertStub: upsert, parentSession: parentSession}, nil
	case stuber.EffectActionDelete:
		deleteID, err := m.prepareDeleteEffect(effect, templateData)
		if err != nil {
			return effectOperation{}, err
		}

		return effectOperation{action: effect.Action, deleteID: deleteID, parentSession: parentSession}, nil
	default:
		return effectOperation{}, errors.New("unknown effect action")
	}
}

func (m *grpcMocker) prepareUpsertEffect(
	effect stuber.Effect,
	templateData template.Data,
	parentSession string,
) (*stuber.Stub, error) {
	if len(effect.Stub) == 0 {
		return nil, errors.New("upsert effect requires stub payload")
	}

	payload := deepCopyMapAny(effect.Stub)
	if err := m.templateEngine.ProcessMap(payload, templateData); err != nil {
		return nil, errors.Wrap(err, "failed to process effect upsert templates")
	}

	stub, err := decodeEffectStub(payload)
	if err != nil {
		return nil, err
	}

	if stub.ID == uuid.Nil {
		stub.ID = uuid.New()
	}

	stub.Session = parentSession
	stub.Source = stuber.SourceRest

	if err := m.validator.Struct(stub); err != nil {
		return nil, errors.Wrap(err, "invalid generated upsert effect stub")
	}

	return stub, nil
}

func (m *grpcMocker) prepareDeleteEffect(
	effect stuber.Effect,
	templateData template.Data,
) (uuid.UUID, error) {
	idString := effect.ID
	if idString == "" {
		return uuid.Nil, errors.New("delete effect requires id")
	}

	if template.IsTemplateString(idString) {
		renderedID, err := m.templateEngine.Render(idString, templateData)
		if err != nil {
			return uuid.Nil, errors.Wrap(err, "failed to process effect delete id template")
		}

		idString = renderedID
	}

	id, err := uuid.Parse(idString)
	if err != nil {
		return uuid.Nil, errors.Wrap(err, "invalid effect delete id")
	}

	return id, nil
}

func (m *grpcMocker) applyEffectOperation(op effectOperation) error {
	switch op.action {
	case stuber.EffectActionUpsert:
		if op.upsertStub == nil {
			return errors.New("prepared upsert effect has nil stub")
		}

		m.budgerigar.PutMany(op.upsertStub)

		return nil
	case stuber.EffectActionDelete:
		existing := m.budgerigar.FindByID(op.deleteID)
		if existing == nil || !effectCanDeleteStub(existing, op.parentSession) {
			return nil
		}

		m.budgerigar.DeleteByID(op.deleteID)

		return nil
	default:
		return errors.New("unknown prepared effect action")
	}
}

func effectCanDeleteStub(stub *stuber.Stub, targetSession string) bool {
	if targetSession == "" {
		return stub.Session == ""
	}

	return stub.Session == targetSession
}

func decodeEffectStub(payload map[string]any) (*stuber.Stub, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal effect stub payload")
	}

	generated := &stuber.Stub{}
	if err := json.Unmarshal(body, generated); err != nil {
		return nil, errors.Wrap(err, "failed to decode effect stub payload")
	}

	return generated, nil
}

func (m *grpcMocker) tryV2API(messages []map[string]any, md metadata.MD) (*stuber.Result, error) {
	query := stuber.Query{
		Service:       m.fullServiceName,
		Method:        m.methodName,
		StrictService: m.strictServiceMatch,
		Input:         messages,
	}

	if len(md) > 0 {
		query.Headers = processHeaders(md)
		query.Session = sessionFromMetadata(md)
	}

	return m.budgerigar.FindByQuery(query)
}

func (m *grpcMocker) handleClientStream(stream grpc.ServerStream) error {
	requestTime := time.Now()

	messages, originalMessages, err := m.collectClientMessages(stream)
	if err != nil {
		return err
	}

	found, err := m.tryFindStub(stream, messages)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return &clientStreamFallbackError{err: err, requests: originalMessages}
		}

		return err
	}

	return m.sendClientStreamResponse(stream, found, messages, requestTime)
}

const clientMessagesInitCap = 16

func (m *grpcMocker) collectClientMessages(stream grpc.ServerStream) ([]map[string]any, []*dynamicpb.Message, error) {
	messages := make([]map[string]any, 0, clientMessagesInitCap)
	originalMessages := make([]*dynamicpb.Message, 0, clientMessagesInitCap)

	for {
		inputMsg := dynamicpb.NewMessage(m.inputDesc)

		err := receiveStreamMessage(stream, inputMsg)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, nil, err //nolint:wrapcheck
		}

		messages = append(messages, convertToMap(inputMsg))
		originalMessages = append(originalMessages, proto.CloneOf(inputMsg))
	}

	return messages, originalMessages, nil
}

func (m *grpcMocker) tryFindStub(stream grpc.ServerStream, messages []map[string]any) (*stuber.Stub, error) {
	md, _ := metadata.FromIncomingContext(stream.Context())

	result, foundErr := m.tryV2API(messages, md)

	if foundErr != nil || result == nil || result.Found() == nil {
		// Use error formatter to include "Closest Match" details
		errorFormatter := NewErrorFormatter()

		// Build query for error formatting
		query := stuber.Query{
			Service:       m.fullServiceName,
			Method:        m.methodName,
			StrictService: m.strictServiceMatch,
			Input:         messages,
		}
		if len(md) > 0 {
			query.Headers = processHeaders(md)
			query.Session = sessionFromMetadata(md)
		}

		// Create empty result if we don't have one
		if result == nil {
			result = &stuber.Result{}
		}

		errMsg := errorFormatter.FormatStubNotFoundError(query, result).Error()

		return nil, status.Error(codes.NotFound, errMsg)
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
	if err := m.delay(stream.Context(), found.Output.Delay); err != nil {
		return err
	}

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

	m.applyEffects(stream.Context(), found, templateData)

	outputMsg, err := m.newOutputMessage(outputDataCopy)
	if err != nil {
		return errors.Wrap(err, "failed to convert response to dynamic message")
	}

	err = stream.SendMsg(outputMsg)
	if err == nil {
		m.recordCall(stream.Context(), found.ID, uint32(codes.OK), requestTime, messages, []map[string]any{outputDataCopy}, "")
	}

	return err
}

func (m *grpcMocker) handleBidiStream(stream grpc.ServerStream) error {
	queryBidi := m.newQueryBidi(stream.Context())

	bidiResult, err := m.budgerigar.FindByQueryBidi(queryBidi)
	if err != nil {
		errorFormatter := NewErrorFormatter()

		query := stuber.Query{
			Service: m.fullServiceName,
			Method:  m.methodName,
			Input:   []map[string]any{},
		}
		if md, ok := metadata.FromIncomingContext(stream.Context()); ok {
			query.Headers = processHeaders(md)
			query.Session = sessionFromMetadata(md)
		}

		result := &stuber.Result{}

		return status.Error(codes.NotFound, errorFormatter.FormatStubNotFoundError(query, result).Error())
	}

	recordingStream := &bidiRecordingStream{
		ServerStream: stream,
		requests:     make([]map[string]any, 0, bidiRecordingStreamInitCap),
		responses:    make([]map[string]any, 0, bidiRecordingStreamResponsesCap),
		maxItems:     maxHistoryStreamMsgs,
	}

	requestTime := time.Now()

	for {
		inputMsg := dynamicpb.NewMessage(m.inputDesc)

		err := receiveStreamMessage(recordingStream, inputMsg)
		if errors.Is(err, io.EOF) {
			m.recordBidiStream(recordingStream, bidiResult, requestTime, "")

			return nil
		}

		if err != nil {
			m.recordBidiStream(recordingStream, bidiResult, requestTime, err.Error())

			if status.Code(err) == codes.NotFound {
				return &bidiStreamFallbackError{err: err, requests: []*dynamicpb.Message{inputMsg}}
			}

			return err //nolint:wrapcheck
		}

		if err := m.processBidiStreamMessageImpl(recordingStream, bidiResult, inputMsg); err != nil {
			m.recordBidiStream(recordingStream, bidiResult, requestTime, err.Error())

			return err
		}
	}
}

func (m *grpcMocker) processBidiStreamMessageImpl(
	stream grpc.ServerStream,
	bidiResult *stuber.BidiResult,
	inputMsg *dynamicpb.Message,
) error {
	requestTime := time.Now()

	stub, err := bidiResult.Next(convertToMap(inputMsg))
	if err != nil {
		wrappedErr := errors.Wrap(err, "failed to process bidirectional message")
		if errors.Is(err, stuber.ErrStubNotFound) {
			return &bidiStreamFallbackError{err: wrappedErr, requests: []*dynamicpb.Message{inputMsg}}
		}

		return wrappedErr
	}

	if err := m.delay(stream.Context(), stub.Output.Delay); err != nil {
		return err
	}

	return m.processBidiStreamSendResponse(stream, bidiResult, stub, inputMsg, requestTime)
}

func (m *grpcMocker) processBidiStreamSendResponse(
	stream grpc.ServerStream,
	bidiResult *stuber.BidiResult,
	stub *stuber.Stub,
	inputMsg *dynamicpb.Message,
	requestTime time.Time,
) error {
	requestData := convertToMap(inputMsg)

	headers := make(map[string]any)
	if md, ok := metadata.FromIncomingContext(stream.Context()); ok {
		headers = processHeaders(md)
	}

	td := template.Data{
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

	outputToUse, err := m.prepareBidiOutput(stub, td)
	if err != nil {
		return err
	}

	m.applyEffects(stream.Context(), stub, td)

	if bidiResult.GetMessageIndex() == 0 {
		if err := m.setResponseHeadersAny(stream.Context(), stream, outputToUse.Headers); err != nil {
			return errors.Wrap(err, "failed to set headers")
		}
	}

	if err := m.handleOutputError(stream.Context(), stream, outputToUse); err != nil { //nolint:wrapcheck
		return err
	}

	if recStream, ok := stream.(*bidiRecordingStream); ok {
		recStream.setStubID(stub.ID)
	}

	return m.sendBidiResponses(stream, outputToUse, stub, bidiResult.GetMessageIndex(), requestTime)
}

func (m *grpcMocker) recordBidiStream(
	stream *bidiRecordingStream,
	_ *stuber.BidiResult,
	requestTime time.Time,
	errMsg string,
) {
	if m.recorder == nil {
		return
	}

	code := uint32(codes.OK)
	if errMsg != "" {
		code = uint32(codes.Unknown)
	}

	requests := stream.getRequests()
	responses := stream.getResponses()

	rec := history.CallRecord{
		Service:   m.fullServiceName,
		Method:    m.methodName,
		Session:   sessionFromContext(stream.Context()),
		Requests:  requests,
		Responses: responses,
		Code:      code,
		Error:     errMsg,
		StubID:    stream.getStubID(),
		Timestamp: requestTime,
	}

	if len(requests) > 0 {
		rec.Request = requests[0]
	}

	if len(responses) > 0 {
		rec.Response = responses[0]
	}

	m.recorder.Record(rec)
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
		Code:    stub.Output.Code,
		Details: deepCopyDetails(stub.Output.Details),
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
	descriptorRegistry *descriptors.Registry,
	tlsConfig *tls.Config,
	remoteClient protosetdom.RemoteClient,
	otelEnabled bool,
	stubValidator *validator.Validate,
) *GRPCServer {
	registry := descriptorRegistry
	if registry == nil {
		registry = descriptors.NewRegistry()
	}

	v := stubValidator
	if v == nil {
		v = mustNewStubValidator()
	}

	var healthState stuber.Aliveness
	if budgerigar != nil {
		healthState = budgerigar
	}

	return &GRPCServer{
		network:      network,
		address:      address,
		params:       params,
		budgerigar:   budgerigar,
		healthState:  healthState,
		waiter:       waiter,
		recorder:     recorder,
		descriptors:  registry,
		remoteClient: remoteClient,
		tlsConfig:    tlsConfig,
		otelEnabled:  otelEnabled,
		validator:    v,
	}
}

func (s *GRPCServer) Build(ctx context.Context) (*grpc.Server, error) {
	var err error

	imports := []string{}
	protoPaths := []string{}

	if s.params != nil {
		imports = s.params.Imports()
		protoPaths = s.params.ProtoPath()
	}

	s.proxies, err = proxyroutes.New(ctx, protoPaths, s.remoteClient)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize proxy routes")
	}

	if s.proxies != nil {
		go func() {
			<-ctx.Done()
			s.proxies.Close()
		}()
	}

	descriptors, err := protosetdom.Build(ctx, imports, protoPaths, s.remoteClient)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build descriptors")
	}

	// Wait for stubs to load before registering services
	// This ensures stubs are available when gRPC server starts accepting requests
	if s.waiter != nil {
		s.waiter.Wait(ctx)
	}

	server := s.createServer(ctx)
	s.setupHealthCheck(server, nil)
	s.registerServices(ctx, server, descriptors, nil)

	// Mark server as ready synchronously after all descriptors and stubs are loaded.
	// This prevents race conditions where health checks arrive before the server is ready.
	s.markServerReady(ctx)

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

	var healthState stuber.Aliveness
	if budgerigar != nil {
		healthState = budgerigar
	}

	s := &GRPCServer{
		budgerigar:  budgerigar,
		healthState: healthState,
		waiter:      waiter,
		recorder:    recorder,
		descriptors: descriptors.NewRegistry(),
	}
	server := s.createServer(ctx)
	s.setupHealthCheck(server, reg)
	s.registerServices(ctx, server, []*descriptorpb.FileDescriptorSet{fds}, reg)

	// Mark server as ready synchronously after all descriptors and stubs are loaded.
	s.markServerReady(ctx)

	return server, nil
}

func (s *GRPCServer) createServer(ctx context.Context) *grpc.Server {
	logger := zerolog.Ctx(ctx)

	opts := []grpc.ServerOption{
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
		grpc.UnknownServiceHandler(s.handleUnknownService),
	}

	if s.otelEnabled {
		opts = append(opts, grpc.StatsHandler(otelgrpc.NewServerHandler()))
	}

	if s.tlsConfig != nil {
		opts = append(opts, grpc.Creds(credentials.NewTLS(s.tlsConfig)))

		logger.Info().Msg("gRPC server configured with TLS")
	}

	return grpc.NewServer(opts...)
}

func (s *GRPCServer) handleUnknownService(_ any, stream grpc.ServerStream) error {
	fullMethod, ok := grpc.MethodFromServerStream(stream)
	if !ok {
		return status.Error(codes.Unimplemented, "method not found")
	}

	serviceName, methodName := splitMethodName(fullMethod)
	if serviceName == unknownValue || methodName == unknownValue {
		return status.Error(codes.Unimplemented, "method not found")
	}

	methodDesc, err := s.findMethodDescriptor(serviceName, methodName)
	if err != nil {
		return status.Error(codes.Unimplemented, err.Error())
	}

	templateEngine := template.New(stream.Context(), nil)
	mocker := &grpcMocker{
		budgerigar:         s.budgerigar,
		templateEngine:     templateEngine,
		errorFormatter:     NewErrorFormatter(),
		recorder:           s.recorder,
		descriptorResolver: &dynamicDescriptorResolver{static: protoregistry.GlobalFiles, dynamic: s.descriptors},
		proxies:            s.proxies,
		validator:          s.validator,
		inputDesc:          methodDesc.Input(),
		outputDesc:         methodDesc.Output(),
		fullServiceName:    serviceName,
		serviceName:        serviceName,
		methodName:         methodName,
		fullMethod:         fullMethod,
		serverStream:       methodDesc.IsStreamingServer(),
		clientStream:       methodDesc.IsStreamingClient(),
		strictServiceMatch: s.proxies != nil && s.proxies.RouteByMethod(fullMethod) != nil,
	}

	if methodDesc.IsStreamingServer() || methodDesc.IsStreamingClient() {
		return mocker.streamHandler(nil, stream)
	}

	req := dynamicpb.NewMessage(methodDesc.Input())
	if err := stream.RecvMsg(req); err != nil {
		return err
	}

	resp, err := mocker.handleUnary(stream.Context(), req)
	if err != nil {
		return err
	}

	return stream.SendMsg(resp)
}

func (s *GRPCServer) findMethodDescriptor(serviceName, methodName string) (protoreflect.MethodDescriptor, error) { //nolint:ireturn
	if method := findMethodInGlobalFiles(serviceName, methodName); method != nil {
		return method, nil
	}

	var found protoreflect.MethodDescriptor

	s.descriptors.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		services := file.Services()
		for i := range services.Len() {
			service := services.Get(i)
			if string(service.FullName()) != serviceName {
				continue
			}

			methods := service.Methods()
			for j := range methods.Len() {
				method := methods.Get(j)
				if string(method.Name()) != methodName {
					continue
				}

				found = method

				return false
			}
		}

		return true
	})

	if found == nil {
		return nil, errors.Errorf("unknown service/method: %s/%s", serviceName, methodName)
	}

	return found, nil
}

func findMethodInGlobalFiles(serviceName, methodName string) protoreflect.MethodDescriptor { //nolint:ireturn
	var found protoreflect.MethodDescriptor

	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		services := file.Services()
		for i := range services.Len() {
			service := services.Get(i)
			if string(service.FullName()) != serviceName {
				continue
			}

			methods := service.Methods()
			for j := range methods.Len() {
				method := methods.Get(j)
				if string(method.Name()) != methodName {
					continue
				}

				found = method

				return false
			}
		}

		return true
	})

	return found
}

func (s *GRPCServer) setupHealthCheck(server *grpc.Server, descResolver *protoregistry.Files) {
	healthServer := health.NewServer()
	healthgrpc.RegisterHealthServer(server, newMockableHealthServer(healthServer, s.budgerigar, descResolver, s.proxies))

	provider := &dynamicServiceInfoProvider{base: server, registry: s.descriptors}

	var staticResolver protodesc.Resolver = protoregistry.GlobalFiles
	if descResolver != nil {
		staticResolver = descResolver
	}

	resolver := &dynamicDescriptorResolver{
		static:  staticResolver,
		dynamic: s.descriptors,
	}

	reflectionSvr := reflection.NewServerV1(reflection.ServerOptions{
		Services:           provider,
		DescriptorResolver: resolver,
	})
	reflectiongrpc.RegisterServerReflectionServer(server, reflectionSvr)

	reflectiongrpcv1alpha.RegisterServerReflectionServer(server, reflection.NewServer(reflection.ServerOptions{
		Services:           provider,
		DescriptorResolver: resolver,
	}))
}

type dynamicServiceInfoProvider struct {
	base     reflection.ServiceInfoProvider
	registry *descriptors.Registry
}

func (p *dynamicServiceInfoProvider) GetServiceInfo() map[string]grpc.ServiceInfo {
	result := make(map[string]grpc.ServiceInfo)

	if p.base != nil {
		maps.Copy(result, p.base.GetServiceInfo())
	}

	if p.registry != nil {
		p.registry.RangeFiles(func(file protoreflect.FileDescriptor) bool {
			services := file.Services()
			for i := range services.Len() {
				serviceName := string(services.Get(i).FullName())
				if _, ok := result[serviceName]; !ok {
					result[serviceName] = grpc.ServiceInfo{}
				}
			}

			return true
		})
	}

	return result
}

type dynamicDescriptorResolver struct {
	static  protodesc.Resolver
	dynamic *descriptors.Registry
}

func (r *dynamicDescriptorResolver) FindFileByPath(path string) (protoreflect.FileDescriptor, error) { //nolint:ireturn
	return (&protosetinfra.Fallback{Primary: r.dynamicFiles(), Fallback: r.static}).FindFileByPath(path)
}

func (r *dynamicDescriptorResolver) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) { //nolint:ireturn
	return (&protosetinfra.Fallback{Primary: r.dynamicFiles(), Fallback: r.static}).FindDescriptorByName(name)
}

func (r *dynamicDescriptorResolver) dynamicFiles() *protoregistry.Files {
	if r.dynamic == nil {
		return nil
	}

	reg := new(protoregistry.Files)

	r.dynamic.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		_ = reg.RegisterFile(file)

		return true
	})

	return reg
}

func (s *GRPCServer) registerServices(
	ctx context.Context,
	server *grpc.Server,
	descriptors []*descriptorpb.FileDescriptorSet,
	reg *protoregistry.Files,
) {
	logger := zerolog.Ctx(ctx)
	registered := make(map[string]struct{})

	for serviceName := range server.GetServiceInfo() {
		registered[serviceName] = struct{}{}
	}

	for _, descriptor := range descriptors {
		for _, file := range descriptor.GetFile() {
			for _, svc := range file.GetService() {
				serviceDesc := s.createServiceDesc(file, svc)

				if _, exists := registered[serviceDesc.ServiceName]; exists {
					logger.Warn().Str("service", serviceDesc.ServiceName).Msg("Service already registered; skipping")

					continue
				}

				if err := s.registerServiceMethods(ctx, &serviceDesc, svc, reg); err != nil {
					logger.Warn().Err(err).Str("service", serviceDesc.ServiceName).Msg("Skipping service due to descriptor error")

					continue
				}

				server.RegisterService(&serviceDesc, nil)
				registered[serviceDesc.ServiceName] = struct{}{}
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

func (s *GRPCServer) registerServiceMethods(
	ctx context.Context,
	serviceDesc *grpc.ServiceDesc,
	svc *descriptorpb.ServiceDescriptorProto,
	reg *protoregistry.Files,
) error {
	for _, method := range svc.GetMethod() {
		inputDesc, outputDesc, err := s.resolveMethodMessageDescriptors(serviceDesc.ServiceName, method, reg)
		if err != nil {
			return err
		}

		m := s.createGrpcMocker(ctx, serviceDesc, svc, method, inputDesc, outputDesc, reg)

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

	return nil
}

//nolint:ireturn
func (s *GRPCServer) resolveMethodMessageDescriptors(
	serviceName string,
	method *descriptorpb.MethodDescriptorProto,
	reg *protoregistry.Files,
) (protoreflect.MessageDescriptor, protoreflect.MessageDescriptor, error) {
	if reg != nil {
		inputDesc, err := getMessageDescriptor(reg, method.GetInputType())
		if err == nil {
			outputDesc, outErr := getMessageDescriptor(reg, method.GetOutputType())
			if outErr == nil {
				return inputDesc, outputDesc, nil
			}
		}
	}

	methodDesc, err := s.findMethodDescriptor(serviceName, method.GetName())
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to resolve method descriptor %s/%s", serviceName, method.GetName())
	}

	return methodDesc.Input(), methodDesc.Output(), nil
}

func (s *GRPCServer) createGrpcMocker(
	ctx context.Context,
	serviceDesc *grpc.ServiceDesc,
	svc *descriptorpb.ServiceDescriptorProto,
	method *descriptorpb.MethodDescriptorProto,
	inputDesc, outputDesc protoreflect.MessageDescriptor,
	reg *protoregistry.Files,
) *grpcMocker {
	templateEngine := template.New(ctx, nil)

	var resolver protodesc.Resolver = protoregistry.GlobalFiles
	if reg != nil {
		resolver = reg
	}

	return &grpcMocker{
		budgerigar:         s.budgerigar,
		templateEngine:     templateEngine,
		errorFormatter:     NewErrorFormatter(),
		recorder:           s.recorder,
		descriptorResolver: resolver,
		proxies:            s.proxies,
		validator:          s.validator,

		inputDesc:  inputDesc,
		outputDesc: outputDesc,

		fullServiceName: serviceDesc.ServiceName,
		serviceName:     svc.GetName(),
		methodName:      method.GetName(),
		fullMethod:      fmt.Sprintf("/%s/%s", serviceDesc.ServiceName, method.GetName()),

		serverStream: method.GetServerStreaming(),
		clientStream: method.GetClientStreaming(),

		strictServiceMatch: s.proxies != nil && s.proxies.RouteByMethod(fmt.Sprintf("/%s/%s", serviceDesc.ServiceName, method.GetName())) != nil,
	}
}

func (s *GRPCServer) markServerReady(ctx context.Context) {
	logger := zerolog.Ctx(ctx)

	logger.Info().Msg("gRPC server is ready to accept requests")

	if s.healthState != nil {
		s.healthState.SetAlive()
	}
}

func getServiceName(file *descriptorpb.FileDescriptorProto, svc *descriptorpb.ServiceDescriptorProto) string {
	if file.GetPackage() != "" {
		return fmt.Sprintf("%s.%s", file.GetPackage(), svc.GetName())
	}

	return svc.GetName()
}

func getMessageDescriptor(reg *protoregistry.Files, messageType string) (protoreflect.MessageDescriptor, error) { //nolint:ireturn
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

//nolint:cyclop,funlen,nestif,gocognit
func (m *grpcMocker) sendStreamResponses(
	stream grpc.ServerStream,
	output stuber.Output,
	stub *stuber.Stub,
	messageIndex int,
	requestTime time.Time,
) error {
	if stub.IsClientStream() {
		streamLen := len(output.Stream)
		if streamLen == 0 {
			return nil
		}

		if messageIndex < 0 {
			return nil
		}

		inputLen := len(stub.Inputs)
		if inputLen == 0 || messageIndex >= inputLen {
			return nil
		}

		start := messageIndex
		if start >= streamLen {
			return nil
		}

		end := start + 1
		if messageIndex == inputLen-1 {
			end = streamLen
		}

		for _, streamElement := range output.Stream[start:end] {
			streamData, ok := streamElement.(map[string]any)
			if !ok {
				continue
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

			if err := sendStreamMessage(stream, outputMsg); err != nil {
				return err //nolint:wrapcheck
			}
		}

		return nil
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
