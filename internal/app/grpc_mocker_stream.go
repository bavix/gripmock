package app

//nolint:revive
import (
	"bytes"
	"context"
	stderrors "errors"
	"fmt"
	"io"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/goccy/go-json"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
	"github.com/bavix/gripmock/v3/internal/infra/types"
)

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

func (m *grpcMocker) delay(ctx context.Context, delayDur types.Duration) error {
	return delayResponse(ctx, delayDur)
}

//nolint:cyclop,funlen
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

	if found.Output.Stream == nil {
		return m.handleServerStreamOutput(stream, found, requestData, outputToUse, requestTime)
	}

	if len(found.Output.Stream) == 0 {
		if err := m.handleOutputError(stream.Context(), stream, outputToUse); err != nil { //nolint:wrapcheck
			return err
		}

		m.recordCall(stream.Context(), found.ID, uint32(codes.OK), requestTime, []map[string]any{requestData}, []any{outputToUse.Data}, "")

		return nil
	}

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

func (m *grpcMocker) handleServerStreamOutput(
	stream grpc.ServerStream,
	found *stuber.Stub,
	requestData map[string]any,
	outputToUse stuber.Output,
	requestTime time.Time,
) error {
	err := m.handleNonArrayStreamData(stream, found)
	if err != nil {
		return err
	}

	m.recordCall(stream.Context(), found.ID, uint32(codes.OK), requestTime, []map[string]any{requestData}, []any{outputToUse.Data}, "")

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
