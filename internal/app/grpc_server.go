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
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-json"
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

const serviceReflection = "grpc.reflection.v1.ServerReflection"

type GRPCServer struct {
	network         string
	address         string
	params          *protoloc.Arguments
	budgerigar      *stuber.Budgerigar
	healthState     stuber.Aliveness
	waiter          Extender
	recorder        history.Recorder
	descriptors     *descriptors.Registry
	remoteClient    protosetdom.RemoteClient
	tlsConfig       *tls.Config
	proxies         *proxyroutes.Registry
	otelEnabled     bool
	maxNestingDepth uint32
	validator       *validator.Validate
	errorFormatter  *ErrorFormatter
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

	maxNestingDepth uint32
}

func (m *grpcMocker) convertToMap(msg proto.Message) map[string]any {
	return convertToMapWithDepth(msg, int(m.maxNestingDepth))
}

//nolint:cyclop
func (m *grpcMocker) streamHandler(srv any, stream grpc.ServerStream) error {
	route := m.proxyRoute()

	if route == nil && m.proxies != nil {
		if m.fullMethod == "/grpc.health.v1.Health/Watch" {
			if routes := m.proxies.Routes(); len(routes) > 0 {
				route = routes[0]
			}
		}
	}

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

	var fallbackErr *fallbackError
	if !stderrors.As(err, &fallbackErr) {
		return m.proxyStream(stream, route, behavior.captureMiss())
	}

	switch fallbackErr.streamType {
	case StreamTypeServer:
		return m.proxyServerStreamWithRequest(stream, route, fallbackErr.request, behavior.captureMiss())
	case StreamTypeClient:
		return m.proxyClientStreamWithRequests(stream, route, fallbackErr.requests, behavior.captureMiss())
	case StreamTypeBidi:
		return m.proxyBidiStreamWithRequests(stream, route, fallbackErr.requests, behavior.captureMiss())
	case StreamTypeUnary:
		return m.proxyStream(stream, route, behavior.captureMiss())
	}

	return m.proxyStream(stream, route, behavior.captureMiss())
}

func (m *grpcMocker) newQuery(ctx context.Context, msg *dynamicpb.Message) stuber.Query {
	query := stuber.Query{
		Service:       m.fullServiceName,
		Method:        m.methodName,
		StrictService: m.strictServiceMatch,
		Input:         []map[string]any{m.convertToMap(msg)},
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

const defaultConvertDepth = 256

type convertScope struct {
	seen  map[protoreflect.Message]struct{}
	depth int
	max   int
}

func newConvertScope(maxDepth int) *convertScope {
	if maxDepth <= 0 {
		maxDepth = defaultConvertDepth
	}

	return &convertScope{
		seen: make(map[protoreflect.Message]struct{}),
		max:  maxDepth,
	}
}

func (c *convertScope) enter(msg protoreflect.Message) bool {
	if msg == nil || !msg.IsValid() {
		return false
	}

	if c.depth >= c.max {
		return false
	}

	if _, ok := c.seen[msg]; ok {
		return false
	}

	c.seen[msg] = struct{}{}
	c.depth++

	return true
}

func (c *convertScope) exit() {
	c.depth--
}

func convertToMap(msg proto.Message) map[string]any {
	if msg == nil {
		return nil
	}

	return convertToMapVisited(msg.ProtoReflect(), newConvertScope(defaultConvertDepth))
}

func convertToMapWithDepth(msg proto.Message, maxDepth int) map[string]any {
	if msg == nil {
		return nil
	}

	return convertToMapVisited(msg.ProtoReflect(), newConvertScope(maxDepth))
}

func convertToMapVisited(message protoreflect.Message, scope *convertScope) map[string]any {
	if !scope.enter(message) {
		return nil
	}
	defer scope.exit()

	desc := message.Descriptor()
	result := make(map[string]any, desc.Fields().Len())

	for i := range desc.Fields().Len() {
		fd := desc.Fields().Get(i)

		if fd.Cardinality() == protoreflect.Repeated && !message.Has(fd) {
			continue
		}

		fieldName := string(fd.Name())
		result[fieldName] = convertValueVisited(fd, message.Get(fd), scope)
	}

	return result
}

func convertValueVisited(fd protoreflect.FieldDescriptor, value protoreflect.Value, scope *convertScope) any {
	switch {
	case fd.IsList():
		return convertListVisited(fd, value.List(), scope)
	case fd.IsMap():
		return convertMapVisited(fd, value.Map(), scope)
	default:
		return convertScalarVisited(fd, value, scope)
	}
}

func convertListVisited(fd protoreflect.FieldDescriptor, list protoreflect.List, scope *convertScope) []any {
	result := make([]any, list.Len())
	elemType := fd.Message()

	for i := range list.Len() {
		elem := list.Get(i)

		if elemType != nil {
			if m := elem.Message(); m.IsValid() {
				result[i] = convertToMapVisited(m, scope)
			}
		} else {
			result[i] = convertScalarVisited(fd, elem, scope)
		}
	}

	return result
}

func convertMapVisited(fd protoreflect.FieldDescriptor, m protoreflect.Map, scope *convertScope) map[string]any {
	result := make(map[string]any)
	keyType := fd.MapKey()
	valType := fd.MapValue().Message()

	m.Range(func(key protoreflect.MapKey, val protoreflect.Value) bool {
		convertedKey, ok := convertScalar(keyType, key.Value()).(string)
		if !ok {
			return true
		}

		if valType != nil {
			if m := val.Message(); m.IsValid() {
				result[convertedKey] = convertToMapVisited(m, scope)
			}
		} else {
			result[convertedKey] = convertScalar(fd.MapValue(), val)
		}

		return true
	})

	return result
}

func convertScalar(fd protoreflect.FieldDescriptor, value protoreflect.Value) any {
	return convertScalarVisited(fd, value, nil)
}

//nolint:cyclop
func convertScalarVisited(fd protoreflect.FieldDescriptor, value protoreflect.Value, scope *convertScope) any {
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
		if scope == nil {
			return convertToMap(value.Message().Interface())
		}

		m := value.Message()
		if !m.IsValid() {
			return nil
		}

		return convertToMapVisited(m, scope)
	default:
		return nil
	}
}

func (m *grpcMocker) delay(ctx context.Context, delayDur types.Duration) error {
	return delayResponse(ctx, delayDur)
}

//nolint:nestif,cyclop,funlen,gocognit
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
			return newServerStreamFallbackError(err, inputMsg)
		}

		return err
	}

	found := result.Found()

	if err := m.delay(stream.Context(), found.Output.Delay); err != nil {
		return err
	}

	outputToUse := found.Output
	requestData := m.convertToMap(inputMsg)

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

			streamResponses := make([]any, 0, len(found.Output.Stream))
			for _, item := range found.Output.Stream {
				if itemMap, ok := item.(map[string]any); ok {
					clean := deepCopyMapAny(itemMap)
					stuber.ExtractGripMockDelay(clean)
					streamResponses = append(streamResponses, clean)
				} else {
					streamResponses = append(streamResponses, item)
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
			[]any{outputToUse.Data},
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
		[]any{outputToUse.Data},
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

		if err := m.handleStreamElement(stream, found, streamData, i, inputMsg, requestTime); err != nil {
			return err
		}
	}

	return nil
}

func (m *grpcMocker) handleStreamElement(
	stream grpc.ServerStream,
	found *stuber.Stub,
	streamData any,
	i int,
	inputMsg *dynamicpb.Message,
	requestTime time.Time,
) error {
	outputData, ok := streamData.(map[string]any)
	if !ok {
		return status.Errorf(codes.Internal, "invalid data format in stream array at index %d", i)
	}

	outputDataCopy := deepCopyMapAny(outputData)

	delay := found.Output.Delay
	if d, ok := stuber.ExtractGripMockDelay(outputDataCopy); ok {
		delay = d
	}

	if err := m.delay(stream.Context(), delay); err != nil {
		return err
	}

	requestData := m.convertToMap(inputMsg)

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

		outputDataCopy := deepCopyAny(found.Output.Data)

		inputMsg := dynamicpb.NewMessage(m.inputDesc)
		if err := stream.RecvMsg(inputMsg); err == nil {
			requestTime := time.Now()
			requestData := m.convertToMap(inputMsg)

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
			if dataMap, ok := outputDataCopy.(map[string]any); ok {
				if err := m.templateEngine.ProcessMap(dataMap, templateData); err != nil {
					return errors.Wrap(err, "failed to process dynamic templates")
				}

				outputDataCopy = dataMap
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

// newOutputMessage converts stub data (a map, scalar, or nil) into a dynamicpb.Message
// for the response descriptor. Map payloads have numeric values converted to
// json.Number so int64 fields survive the JSON round trip; scalar payloads (e.g. a
// well-known type whose JSON encoding is a primitive: string for Timestamp, number
// for wrappers, object for Struct) are JSON-marshaled as-is and fed to protojson,
// which natively understands the canonical JSON form for every WKT.
func (m *grpcMocker) newOutputMessage(data any) (*dynamicpb.Message, error) {
	pooled, _ := jsonBufferPool.Get().(*bytes.Buffer)
	if pooled == nil {
		pooled = bytes.NewBuffer(make([]byte, 0, jsonBufferInitialCap))
	}

	pooled.Reset()

	defer func() {
		pooled.Reset()
		jsonBufferPool.Put(pooled)
	}()

	payload := data
	if dataMap, ok := data.(map[string]any); ok {
		payload = convertMapNumericToStringNumber(dataMap, m.outputDesc)
	}

	enc := json.NewEncoder(pooled)
	if err := enc.Encode(payload); err != nil {
		return nil, fmt.Errorf("failed to marshal output to JSON: %w", err)
	}

	msg := dynamicpb.NewMessage(m.outputDesc)

	jsonBytes := pooled.Bytes()
	if err := protojson.Unmarshal(jsonBytes, msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON into dynamic message: %w (json=%s)", err, string(jsonBytes))
	}

	return msg, nil
}

func convertMapNumericToStringNumber(data map[string]any, desc protoreflect.MessageDescriptor) map[string]any {
	result := make(map[string]any, len(data))

	for k, v := range data {
		var fd protoreflect.FieldDescriptor
		if desc != nil {
			fd = desc.Fields().ByName(protoreflect.Name(k))
			if fd == nil {
				fd = desc.Fields().ByJSONName(k)
			}
		}

		result[k] = convertMapValue(v, fd)
	}

	return result
}

func convertMapValue(v any, fd protoreflect.FieldDescriptor) any {
	switch val := v.(type) {
	case map[string]any:
		var nestedDesc protoreflect.MessageDescriptor
		if fd != nil && fd.Kind() == protoreflect.MessageKind {
			nestedDesc = fd.Message()
		}

		return convertMapNumericToStringNumber(val, nestedDesc)
	case []any:
		return convertMapArray(val, fd)
	case string:
		return convertStringValue(val, fd)
	case float64:
		return convertFloat64(val)
	case float32:
		return convertFloat64(float64(val))
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return convertIntLikeValue(val)
	default:
		return v
	}
}

func convertIntLikeValue(v any) any {
	switch val := v.(type) {
	case int, int8, int16, int32, int64:
		return json.Number(strconv.FormatInt(toInt64(val), 10))
	default:
		return json.Number(strconv.FormatUint(toUint64(val), 10))
	}
}

func convertStringValue(val string, fd protoreflect.FieldDescriptor) any {
	if fd == nil || !isNumericKind(fd.Kind()) {
		return val
	}

	// 64-bit integers keep string representation to avoid float64 precision loss.
	// Protojson accepts both string and number for these types.
	if is64BitIntKind(fd.Kind()) {
		return val
	}

	if f, err := strconv.ParseFloat(val, 64); err == nil {
		return convertFloat64(f)
	}

	return val
}

func is64BitIntKind(k protoreflect.Kind) bool {
	switch k {
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return true
	case protoreflect.BoolKind, protoreflect.EnumKind,
		protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Uint32Kind,
		protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind,
		protoreflect.FloatKind, protoreflect.DoubleKind,
		protoreflect.StringKind, protoreflect.BytesKind,
		protoreflect.MessageKind, protoreflect.GroupKind:
		return false
	default:
		return false
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

func convertMapArray(arr []any, fd protoreflect.FieldDescriptor) []any {
	result := make([]any, len(arr))

	for i, v := range arr {
		result[i] = convertMapValue(v, fd)
	}

	return result
}

func isNumericKind(k protoreflect.Kind) bool {
	switch k {
	case protoreflect.DoubleKind, protoreflect.FloatKind,
		protoreflect.Int32Kind, protoreflect.Int64Kind,
		protoreflect.Uint32Kind, protoreflect.Uint64Kind,
		protoreflect.Sint32Kind, protoreflect.Sint64Kind,
		protoreflect.Sfixed32Kind, protoreflect.Sfixed64Kind,
		protoreflect.Fixed32Kind, protoreflect.Fixed64Kind:
		return true
	case protoreflect.BoolKind, protoreflect.EnumKind,
		protoreflect.StringKind, protoreflect.BytesKind,
		protoreflect.MessageKind, protoreflect.GroupKind:
		return false
	default:
		return false
	}
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
					return m.handleUnaryWithProxy(ctx, nil, msg)
				}

				return nil, status.Errorf(codes.InvalidArgument, "expected *dynamicpb.Message, got %T", req)
			})
		}

		return m.handleUnaryWithProxy(ctx, nil, req)
	}
}

//nolint:cyclop
func (m *grpcMocker) handleUnaryWithProxy(
	ctx context.Context,
	stream grpc.ServerStream,
	req *dynamicpb.Message,
) (*dynamicpb.Message, error) {
	route := m.proxyRoute()

	// Health check is excluded from the proxy index by the reflection
	// client (shouldSkipService). Fall back to the first available route.
	if route == nil && m.proxies != nil {
		if m.fullMethod == "/grpc.health.v1.Health/Check" {
			if routes := m.proxies.Routes(); len(routes) > 0 {
				route = routes[0]
			}
		}
	}

	behavior := newProxyBehavior(route)

	if behavior == nil {
		return m.handleUnary(ctx, stream, req)
	}

	if behavior.proxyOnly() {
		return m.proxyUnary(ctx, stream, req, route, false)
	}

	if behavior.captureMiss() && m.captureShouldProxyUnaryByHeaders(ctx, req) {
		return m.proxyUnary(ctx, stream, req, route, true)
	}

	resp, err := m.handleUnary(ctx, stream, req)

	var fallbackErr *fallbackError
	if !stderrors.As(err, &fallbackErr) || fallbackErr.streamType != StreamTypeUnary {
		return resp, err
	}

	return m.proxyUnary(ctx, stream, req, route, behavior.captureMiss())
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
func (m *grpcMocker) handleUnary(ctx context.Context, stream grpc.ServerStream, req *dynamicpb.Message) (*dynamicpb.Message, error) {
	requestTime := time.Now()

	query := m.newQuery(ctx, req)

	result, err := m.budgerigar.FindByQuery(query)

	// Handle both error and nil result cases with unified error formatting
	if err != nil || (result != nil && result.Found() == nil) {
		// Create empty result if we don't have one (error case)
		if result == nil {
			result = &stuber.Result{}
		}

		return nil, newUnaryFallbackError(status.Error(codes.NotFound, m.errorFormatter.FormatStubNotFoundError(query, result).Error()))
	}

	found := result.Found()

	if err := m.delay(ctx, found.Output.Delay); err != nil {
		return nil, err
	}

	outputToUse := found.Output
	requestData := m.convertToMap(req)

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

	outputDataCopy := deepCopyAny(outputToUse.Data)

	if dataMap, ok := outputDataCopy.(map[string]any); ok {
		if err := m.templateEngine.ProcessMap(dataMap, templateData); err != nil {
			zerolog.Ctx(ctx).Err(err).Msg("failed to process dynamic templates")

			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to process dynamic templates: %v", err))
		}

		outputDataCopy = dataMap
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

	if err := m.setResponseHeadersAny(ctx, stream, outputToUse.Headers); err != nil {
		return nil, err //nolint:wrapcheck
	}

	m.applyEffects(ctx, found, templateData)

	if err := m.handleOutputError(ctx, stream, outputToUse); err != nil {
		code := status.Code(err)
		m.recordCall(ctx, found.ID, uint32(code), requestTime, []map[string]any{requestData}, nil, err.Error())
		outputToUse.Error = err.Error()

		return nil, err //nolint:wrapcheck
	}

	outputMsg, err := m.newOutputMessage(outputDataCopy)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	m.recordCall(ctx, found.ID, uint32(codes.OK), requestTime, []map[string]any{requestData}, []any{outputDataCopy}, "")

	return outputMsg, nil
}

func (m *grpcMocker) setResponseHeadersAny(ctx context.Context, stream grpc.ServerStream, headers map[string]string) error {
	if len(headers) == 0 {
		return nil
	}

	mdResp := make(metadata.MD, len(headers))
	for k, v := range headers {
		switch strings.ToLower(k) {
		case "content-type", "content-length", "content-encoding", "grpc-status", "grpc-message", "grpc-status-details-bin":
			continue
		}

		mdResp.Append(k, strings.Split(v, ";")...)
	}

	if len(mdResp) == 0 {
		return nil
	}

	if stream != nil {
		return stream.SetHeader(mdResp)
	}

	_ = grpc.SetHeader(ctx, mdResp)

	return nil
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

func (m *grpcMocker) matchFirstMessage(stream grpc.ServerStream, messages []map[string]any) *stuber.Stub {
	stubs, _ := m.budgerigar.FindBy(m.fullServiceName, m.methodName)
	for _, s := range stubs {
		if !s.MatchOnFirstMessage {
			continue
		}

		query := stuber.Query{
			Service:       m.fullServiceName,
			Method:        m.methodName,
			StrictService: m.strictServiceMatch,
			Input:         []map[string]any{messages[0]},
		}
		if md, ok := metadata.FromIncomingContext(stream.Context()); ok {
			query.Headers = processHeaders(md)
			query.Session = sessionFromMetadata(md)
		}

		result, matchErr := m.budgerigar.FindByQuery(query)
		if matchErr == nil && result != nil && result.Found() != nil {
			return result.Found()
		}
	}

	return nil
}

func (m *grpcMocker) handleClientStream(stream grpc.ServerStream) error {
	requestTime := time.Now()

	messages, originalMessages, err := m.collectClientMessages(stream)
	if err != nil {
		return err
	}

	zerolog.Ctx(stream.Context()).Debug().Int("msg_count", len(messages)).Msg("client_stream: collected messages")

	if len(messages) > 0 {
		if found := m.matchFirstMessage(stream, messages); found != nil {
			return m.sendClientStreamResponse(stream, found, messages, requestTime)
		}
	}

	found, err := m.tryFindStub(stream, messages)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return newClientStreamFallbackError(err, originalMessages)
		}

		return err
	}

	return m.sendClientStreamResponse(stream, found, messages, requestTime)
}

const clientMessagesInitCap = 16

func (m *grpcMocker) collectClientMessages(stream grpc.ServerStream) ([]map[string]any, []*dynamicpb.Message, error) {
	messages := make([]map[string]any, 0, clientMessagesInitCap)
	originalMessages := make([]*dynamicpb.Message, 0, clientMessagesInitCap)

	for i := 0; ; i++ {
		inputMsg := dynamicpb.NewMessage(m.inputDesc)

		err := receiveStreamMessage(stream, inputMsg)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, nil, err //nolint:wrapcheck
		}

		msgMap := m.convertToMap(inputMsg)
		messages = append(messages, msgMap)
		originalMessages = append(originalMessages, proto.CloneOf(inputMsg))
	}

	return messages, originalMessages, nil
}

func (m *grpcMocker) tryFindStub(stream grpc.ServerStream, messages []map[string]any) (*stuber.Stub, error) {
	md, _ := metadata.FromIncomingContext(stream.Context())

	result, foundErr := m.tryV2API(messages, md)

	if foundErr != nil || result == nil || result.Found() == nil {
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

		errMsg := m.errorFormatter.FormatStubNotFoundError(query, result).Error()

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

	outputDataCopy := deepCopyAny(found.Output.Data)

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
	if dataMap, ok := outputDataCopy.(map[string]any); ok {
		if err := m.templateEngine.ProcessMap(dataMap, templateData); err != nil {
			return errors.Wrap(err, "failed to process dynamic templates")
		}

		outputDataCopy = dataMap
	}

	m.applyEffects(stream.Context(), found, templateData)

	outputMsg, err := m.newOutputMessage(outputDataCopy)
	if err != nil {
		return errors.Wrap(err, "failed to convert response to dynamic message")
	}

	err = stream.SendMsg(outputMsg)
	if err == nil {
		m.recordCall(stream.Context(), found.ID, uint32(codes.OK), requestTime, messages, []any{outputDataCopy}, "")
	}

	return err
}

func (m *grpcMocker) handleBidiStream(stream grpc.ServerStream) error {
	queryBidi := m.newQueryBidi(stream.Context())

	// Check for custom handler
	stubs, _ := m.budgerigar.FindBy(queryBidi.Service, queryBidi.Method)
	if len(stubs) > 0 && stubs[0].Handler != nil {
		return stubs[0].Handler(stream.Context(), stream)
	}

	bidiResult, err := m.budgerigar.FindByQueryBidi(queryBidi)
	if err != nil {
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

		return status.Error(codes.NotFound, m.errorFormatter.FormatStubNotFoundError(query, result).Error())
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
				return newBidiStreamFallbackError(err, []*dynamicpb.Message{inputMsg})
			}

			return err
		}

		if err := m.processBidiStreamMessage(recordingStream, bidiResult, inputMsg); err != nil {
			m.recordBidiStream(recordingStream, bidiResult, requestTime, err.Error())

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
	inputMap := m.convertToMap(inputMsg)

	stub, err := bidiResult.Next(inputMap)
	if err != nil {
		wrappedErr := errors.Wrap(err, "failed to process bidirectional message")
		if errors.Is(err, stuber.ErrStubNotFound) {
			return newBidiStreamFallbackError(wrappedErr, []*dynamicpb.Message{inputMsg})
		}

		return wrappedErr
	}

	if err := m.delay(stream.Context(), stub.Output.Delay); err != nil {
		return err
	}

	return m.sendBidiResponse(stream, stub, inputMsg, bidiResult, requestTime)
}

func (m *grpcMocker) sendBidiResponse(
	stream grpc.ServerStream,
	stub *stuber.Stub,
	inputMsg *dynamicpb.Message,
	bidiResult *stuber.BidiResult,
	requestTime time.Time,
) error {
	requestData := m.convertToMap(inputMsg)
	md, _ := metadata.FromIncomingContext(stream.Context())

	headers := make(map[string]any)
	if len(md) > 0 {
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

	if err := m.handleOutputError(stream.Context(), stream, outputToUse); err != nil {
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

//nolint:cyclop
func (m *grpcMocker) prepareBidiOutput(stub *stuber.Stub, templateData template.Data) (stuber.Output, error) {
	outputDataCopy := deepCopyAny(stub.Output.Data)
	if dataMap, ok := outputDataCopy.(map[string]any); ok {
		if err := m.templateEngine.ProcessMap(dataMap, templateData); err != nil {
			return stuber.Output{}, errors.Wrap(err, "failed to process dynamic templates")
		}

		outputDataCopy = dataMap
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
	maxNestingDepth uint32,
	stubValidator *validator.Validate,
	errorFormatter *ErrorFormatter,
) *GRPCServer {
	registry := descriptorRegistry
	if registry == nil {
		registry = descriptors.NewRegistry()
	}

	v := stubValidator
	if v == nil {
		v = mustNewStubValidator()
	}

	e := errorFormatter
	if e == nil {
		e = NewErrorFormatter()
	}

	var healthState stuber.Aliveness
	if budgerigar != nil {
		healthState = budgerigar
	}

	return &GRPCServer{
		network:         network,
		address:         address,
		params:          params,
		budgerigar:      budgerigar,
		healthState:     healthState,
		waiter:          waiter,
		recorder:        recorder,
		descriptors:     registry,
		remoteClient:    remoteClient,
		tlsConfig:       tlsConfig,
		otelEnabled:     otelEnabled,
		maxNestingDepth: maxNestingDepth,
		validator:       v,
		errorFormatter:  e,
	}
}

func (s *GRPCServer) Proxies() *proxyroutes.Registry {
	return s.proxies
}

//nolint:cyclop
func (s *GRPCServer) Build(ctx context.Context) (*grpc.Server, error) {
	var err error

	imports := []string{}
	protoPaths := []string{}
	sources := []string{}

	var descriptors []*descriptorpb.FileDescriptorSet

	if s.params != nil {
		imports = s.params.Imports()
		protoPaths = s.params.ProtoPath()
		sources = s.params.Sources()
	}

	if s.params != nil && s.params.HasProxyBindings() {
		descriptors, s.proxies, err = s.buildProxiesWithBindings(ctx, imports)
	} else {
		descriptors, s.proxies, err = s.buildProxiesFromSources(ctx, imports, protoPaths, sources)
	}

	if err != nil {
		return nil, err
	}

	if s.proxies != nil {
		s.startProxyCleanup(ctx)
		s.registerProxyDescriptors(ctx)
	}

	if len(protoPaths) > 0 {
		nonProxyDescriptors, err := protosetdom.Build(ctx, imports, protoPaths, s.remoteClient)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build descriptors")
		}

		descriptors = append(descriptors, nonProxyDescriptors...)
	}

	if s.proxies != nil {
		proxyFiles := s.proxies.Files()
		if len(proxyFiles) > 0 {
			descriptors = append(descriptors, proxyFiles...)
		}
	}

	if s.waiter != nil {
		s.waiter.Wait(ctx)
	}

	server := s.createServer(ctx)
	s.setupHealthCheck(server, nil)
	s.registerServices(ctx, server, descriptors, nil)
	s.markServerReady(ctx)

	return server, nil
}

func (s *GRPCServer) buildProxiesWithBindings(ctx context.Context, imports []string) (
	[]*descriptorpb.FileDescriptorSet,
	*proxyroutes.Registry,
	error,
) {
	var err error

	bindings := s.params.ProxyBindings()
	proxyBindings := make([]proxyroutes.ProxyDescriptorBinding, 0, len(bindings))
	logger := zerolog.Ctx(ctx)

	for _, binding := range bindings {
		logger.Info().
			Str("proxy", binding.ProxyURL).
			Strs("sources", binding.Sources).
			Msg("processing proxy binding")

		var bindingDescriptors []*descriptorpb.FileDescriptorSet

		if len(binding.Sources) > 0 {
			bindingDescriptors, err = protosetdom.Build(ctx, imports, binding.Sources, s.remoteClient)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to build descriptors for proxy %s", binding.ProxyURL)
			}

			logger.Info().
				Str("proxy", binding.ProxyURL).
				Int("num_descriptors", len(bindingDescriptors)).
				Msg("built descriptors for proxy")
		} else {
			logger.Info().
				Str("proxy", binding.ProxyURL).
				Msg("no sources for proxy, will use reflection")
		}

		proxyBindings = append(proxyBindings, proxyroutes.ProxyDescriptorBinding{
			ProxyURL:    binding.ProxyURL,
			Descriptors: bindingDescriptors,
		})
	}

	proxies, err := proxyroutes.NewWithPerProxyDescriptors(ctx, proxyBindings, s.remoteClient)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to initialize proxy routes")
	}

	proxyFiles := proxies.Files()
	if len(proxyFiles) > 0 {
		descriptors := make([]*descriptorpb.FileDescriptorSet, 0, len(proxyFiles))

		return append(descriptors, proxyFiles...), proxies, nil
	}

	return nil, proxies, nil
}

func (s *GRPCServer) buildProxiesFromSources(ctx context.Context, imports []string, protoPaths []string, sources []string) (
	[]*descriptorpb.FileDescriptorSet,
	*proxyroutes.Registry,
	error,
) {
	allPaths := make([]string, 0, len(protoPaths)+len(sources))
	allPaths = append(allPaths, protoPaths...)
	allPaths = append(allPaths, sources...)

	var descriptors []*descriptorpb.FileDescriptorSet

	if len(allPaths) > 0 {
		var err error

		descriptors, err = protosetdom.Build(ctx, imports, allPaths, s.remoteClient)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to build descriptors")
		}
	}

	proxies, err := proxyroutes.New(ctx, allPaths, s.remoteClient, descriptors)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to initialize proxy routes")
	}

	return descriptors, proxies, nil
}

func (s *GRPCServer) startProxyCleanup(ctx context.Context) {
	go func() {
		<-ctx.Done()
		s.proxies.Close()
	}()
}

func (s *GRPCServer) registerProxyDescriptors(ctx context.Context) {
	proxyFiles := s.proxies.Files()
	if len(proxyFiles) == 0 {
		return
	}

	for i, fds := range proxyFiles {
		source := fmt.Sprintf("proxy-descriptor-set-%d", i)
		if err := protosetdom.RegisterDescriptorSetFiles(ctx, source, fds); err != nil {
			zerolog.Ctx(ctx).Err(err).Int("index", i).Msg("failed to register proxy descriptor set")
		}
	}
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
		budgerigar:     budgerigar,
		healthState:    healthState,
		waiter:         waiter,
		recorder:       recorder,
		descriptors:    descriptors.NewRegistry(),
		validator:      mustNewStubValidator(),
		errorFormatter: NewErrorFormatter(),
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
		errorFormatter:     s.errorFormatter,
		recorder:           s.recorder,
		descriptorResolver: &dynamicDescriptorResolver{static: protoregistry.GlobalFiles, dynamic: s.descriptors},
		proxies:            s.proxies,
		validator:          s.validator,
		maxNestingDepth:    s.maxNestingDepth,
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

	resp, err := mocker.handleUnary(stream.Context(), stream, req)
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
	return findMethodInFiles(protoregistry.GlobalFiles, serviceName, methodName)
}

// methodFilesLister abstracts a descriptor registry that supports iteration
// over file descriptors. Implemented by *protoregistry.Files and
// *descriptors.Registry.
type methodFilesLister interface {
	RangeFiles(f func(protoreflect.FileDescriptor) bool)
}

func findMethodInFiles(files methodFilesLister, serviceName, methodName string) protoreflect.MethodDescriptor { //nolint:ireturn
	var found protoreflect.MethodDescriptor

	files.RangeFiles(func(file protoreflect.FileDescriptor) bool {
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
		errorFormatter:     s.errorFormatter,
		recorder:           s.recorder,
		descriptorResolver: resolver,
		proxies:            s.proxies,
		validator:          s.validator,
		maxNestingDepth:    s.maxNestingDepth,

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

	outputDataCopy := deepCopyAny(output.Data)

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
	if dataMap, ok := outputDataCopy.(map[string]any); ok {
		if err := m.templateEngine.ProcessMap(dataMap, templateData); err != nil {
			return errors.Wrap(err, "failed to process dynamic templates")
		}

		outputDataCopy = dataMap
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

			delayDelay := output.Delay
			if d, ok := stuber.ExtractGripMockDelay(streamDataCopy); ok {
				delayDelay = d
			}

			if err := m.delay(stream.Context(), delayDelay); err != nil {
				return err
			}

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
		streamDataCopy := streamElement
		if streamData, ok := streamElement.(map[string]any); ok {
			copied := deepCopyMapAny(streamData)
			streamDataCopy = copied

			delayDelay := output.Delay
			if d, found := stuber.ExtractGripMockDelay(copied); found {
				delayDelay = d
			}

			if delayDelay != 0 {
				if err := m.delay(stream.Context(), delayDelay); err != nil {
					return err
				}
			}
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
