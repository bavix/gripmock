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
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
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

	return m.withHealthVisibility(query)
}

// withHealthVisibility mirrors mockableHealthServer.findStub: the health
// service is the only caller allowed to see the reserved internal stubs. On
// native gRPC health is served by that handler, but the gateways route it
// through this mocker, and without the flag a user stub would override the
// runtime status of the protected "gripmock" service.
func (m *grpcMocker) withHealthVisibility(query stuber.Query) stuber.Query {
	if m.fullServiceName != HealthServiceFullName {
		return query
	}

	return stuber.WithInternalStubs(query)
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

	if err := delayResponse(stream.Context(), found.Output.Delay); err != nil {
		return err
	}

	outputToUse := found.Output
	requestData := m.convertToMap(inputMsg)

	headers := make(map[string]any)
	if md, ok := metadata.FromIncomingContext(stream.Context()); ok {
		headers = processHeaders(md)
	}

	templateData := newTemplateData(requestData, headers, 0, requestTime, []any{requestData}, found.ID.String())

	streamCopy, hasStreamTemplate, err := renderOutputStreamTemplate(m.templateEngine, outputToUse, templateData)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	if hasStreamTemplate {
		outputToUse.Stream = streamCopy
	}

	if template.HasTemplatesInHeaders(outputToUse.Headers) {
		headersCopy := deepCopyStringMap(outputToUse.Headers)
		if err := m.templateEngine.ProcessHeaders(headersCopy, templateData); err != nil {
			return errors.Wrap(err, "failed to process header templates")
		}

		outputToUse.Headers = headersCopy
	}

	if outputToUse.Error != "" && template.IsTemplateString(outputToUse.Error) {
		errorStr, err := m.templateEngine.ProcessError(outputToUse.Error, templateData)
		if err != nil {
			return errors.Wrap(err, "failed to process error template")
		}

		outputToUse.Error = errorStr
	}

	if err := m.setResponseHeadersAny(stream.Context(), stream, outputToUse.Headers); err != nil {
		return errors.Wrap(err, "failed to set headers")
	}

	if err := m.renderTrailers(&outputToUse, templateData); err != nil {
		return errors.Wrap(err, "failed to process trailer templates")
	}

	m.setResponseTrailersAny(stream.Context(), stream, outputToUse.Trailers)

	m.applyEffects(stream.Context(), found, templateData)

	if outputToUse.Stream == nil {
		return m.handleServerStreamOutput(stream, found, requestData, outputToUse, requestTime)
	}

	if len(outputToUse.Stream) == 0 {
		callErr := m.handleOutputError(stream.Context(), stream, outputToUse)

		m.recordServerStreamUnlessProxied(stream.Context(), found, requestTime,
			requestData, []any{outputToUse.Data}, recordedMetadata(outputToUse), callErr)

		return callErr //nolint:wrapcheck
	}

	var (
		sent    int
		callErr error
	)

	if hasStreamTemplate {
		sent, callErr = m.handleRenderedArrayStreamData(stream, outputToUse)
	} else {
		sent, callErr = m.handleArrayStreamData(stream, found, inputMsg, requestTime)
	}

	if callErr == nil {
		callErr = m.handleOutputError(stream.Context(), stream, outputToUse)
	}

	m.recordServerStreamUnlessProxied(stream.Context(), found, requestTime,
		requestData, cleanStreamResponses(outputToUse.Stream[:sent]), recordedMetadata(outputToUse), callErr)

	return callErr //nolint:wrapcheck
}

func (m *grpcMocker) streamElementError(element stuber.GripMockElement, templateData template.Data) error {
	msg := element.Error
	if msg != "" && template.IsTemplateString(msg) {
		rendered, err := m.templateEngine.ProcessError(msg, templateData)
		if err != nil {
			return errors.Wrap(err, "failed to process error template")
		}

		msg = rendered
	}

	st, err := m.statusFromOutput(stuber.Output{
		Error:   msg,
		Code:    element.Code,
		Details: element.Details,
	})
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	if st == nil {
		return nil
	}

	return st.Err()
}

func cleanStreamResponses(items []any) []any {
	responses := make([]any, 0, len(items))

	for _, item := range items {
		itemMap, ok := item.(map[string]any)
		if !ok {
			responses = append(responses, item)

			continue
		}

		clean := deepCopyMapAny(itemMap)
		stuber.ExtractGripMockDelay(clean)
		responses = append(responses, clean)
	}

	return responses
}

func (m *grpcMocker) recordServerStreamUnlessProxied(
	ctx context.Context,
	found *stuber.Stub,
	requestTime time.Time,
	requestData map[string]any,
	responses []any,
	headers map[string]string,
	callErr error,
) {
	if callErr != nil && m.proxyFallbackWillServe(callErr) {
		return
	}

	code := uint32(codes.OK)
	errMsg := ""

	if callErr != nil {
		code = uint32(status.Code(callErr))
		errMsg = callErr.Error()
	}

	m.recordCall(ctx, found.ID, code, requestTime,
		[]map[string]any{requestData}, responses, headers, errMsg)
}

func (m *grpcMocker) handleServerStreamOutput(
	stream grpc.ServerStream,
	found *stuber.Stub,
	requestData map[string]any,
	outputToUse stuber.Output,
	requestTime time.Time,
) error {
	callErr := m.handleNonArrayStreamData(stream, found, outputToUse, requestData, requestTime)

	m.recordServerStreamUnlessProxied(stream.Context(), found, requestTime,
		requestData, []any{outputToUse.Data}, recordedMetadata(outputToUse), callErr)

	return callErr
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
) (int, error) {
	done := stream.Context().Done()

	for i, streamData := range found.Output.Stream {
		select {
		case <-done:
			return i, stream.Context().Err()
		default:
		}

		if err := m.handleStreamElement(stream, found, streamData, i, inputMsg, requestTime); err != nil {
			return i, err
		}
	}

	return len(found.Output.Stream), nil
}

// handleRenderedArrayStreamData sends structural template results without
// processing their scalar values as templates a second time.
func (m *grpcMocker) handleRenderedArrayStreamData(stream grpc.ServerStream, output stuber.Output) (int, error) {
	done := stream.Context().Done()

	for i, streamData := range output.Stream {
		select {
		case <-done:
			return i, stream.Context().Err()
		default:
		}

		payload, err := m.prepareStreamElement(stream.Context(), streamData, output.Delay)
		if err != nil {
			return i, err
		}

		outputMsg, err := m.newOutputMessage(payload)
		if err != nil {
			return i, errors.Wrap(err, "failed to convert response to dynamic message")
		}

		if err := sendStreamMessage(stream, outputMsg); err != nil {
			return i, err
		}
	}

	return len(output.Stream), nil
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
	element := stuber.ExtractGripMock(outputDataCopy)

	delay := found.Output.Delay
	if d, ok := element.Delay, element.HasDelay; ok {
		delay = d
	}

	if err := delayResponse(stream.Context(), delay); err != nil {
		return err
	}

	requestData := m.convertToMap(inputMsg)

	headers := make(map[string]any)
	if md, ok := metadata.FromIncomingContext(stream.Context()); ok {
		headers = processHeaders(md)
	}

	templateData := newTemplateData(requestData, headers, i, requestTime, []any{requestData}, found.ID.String())

	if element.HasError {
		return m.streamElementError(element, templateData)
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

//nolint:cyclop
func (m *grpcMocker) handleNonArrayStreamData(
	stream grpc.ServerStream,
	found *stuber.Stub,
	outputToUse stuber.Output,
	requestData map[string]any,
	requestTime time.Time,
) error {
	// Use outputToUse (templated headers/error already rendered by the caller),
	// not found.Output — otherwise an error template is emitted unrendered.
	if err := m.handleOutputError(stream.Context(), stream, outputToUse); err != nil {
		return err
	}

	done := stream.Context().Done()

	for {
		select {
		case <-done:
			return stream.Context().Err()
		default:
		}

		if err := delayResponse(stream.Context(), found.Output.Delay); err != nil {
			return err
		}

		// Render against the request the caller already consumed. A server-stream
		// client half-closes after one message, so a fresh RecvMsg here returns
		// EOF and would otherwise leave the data template unrendered; only a genuine
		// follow-up message (bidi-like) overrides the captured request.
		msgData, msgTime := requestData, requestTime

		inputMsg := dynamicpb.NewMessage(m.inputDesc)
		if err := stream.RecvMsg(inputMsg); err == nil {
			msgData = m.convertToMap(inputMsg)
			msgTime = time.Now()
		}

		headers := make(map[string]any)
		if md, ok := metadata.FromIncomingContext(stream.Context()); ok {
			headers = processHeaders(md)
		}

		templateData := newTemplateData(msgData, headers, 0, msgTime, []any{msgData}, found.ID.String())

		outputDataCopy, err := renderOutputData(m.templateEngine, found.Output, templateData)
		if err != nil {
			return status.Error(codes.Internal, err.Error())
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

	if err := m.typeResolver.Unmarshal(jsonBytes, msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON into dynamic message: %w (json=%s)", err, string(jsonBytes))
	}

	return msg, nil
}
