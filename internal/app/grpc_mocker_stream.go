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

// convertToMap renders a request for matching, recording and history.
//
// A method behind a proxy keeps every singular field the request declares, set or
// not. Both sides of a capture must speak the same vocabulary: a stub recorded
// without the fields the caller left unset would answer requests that do set them,
// and one recorded with fields the lookup never sends would never match again.
func (m *grpcMocker) convertToMap(msg proto.Message) map[string]any {
	if m.proxyRoute() != nil {
		return convertRequestForCapture(msg, int(m.maxNestingDepth))
	}

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
			m.recordUnmatched(stream.Context(), requestTime, []map[string]any{m.convertToMap(inputMsg)}, err)

			return newServerStreamFallbackError(err, inputMsg)
		}

		return err
	}

	found := result.Found()

	outputToUse := found.Output
	requestData := m.convertToMap(inputMsg)

	headers := make(map[string]any)
	if md, ok := metadata.FromIncomingContext(stream.Context()); ok {
		headers = processHeaders(md)
	}

	matchNumber := result.MatchNumber()
	templateData := newTemplateData(requestData, headers, 0, requestTime,
		[]any{requestData}, found, matchNumber)

	if !streamDelaysPerMessage(found) {
		err := delayTemplated(stream.Context(), m.templateEngine, found.Output.Delay, templateData)
		if err != nil {
			return err
		}
	}

	outputToUse, err = renderOutput(m.templateEngine, outputToUse, templateData,
		renderOptions{skipData: true})
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	err = m.setResponseHeadersAny(stream.Context(), stream, outputToUse.Headers)
	if err != nil {
		return errors.Wrap(err, "failed to set headers")
	}

	m.setResponseTrailersAny(stream.Context(), stream, outputToUse.Trailers)

	m.applyEffects(stream.Context(), found, templateData)

	if found.ServerStreamHandler != nil {
		callErr := handlerStatusError(found.ServerStreamHandler(stream.Context(), requestData, stream))

		m.recordServerStreamUnlessProxied(stream.Context(), found, requestTime,
			requestData, nil, recordedMetadata(outputToUse), callErr)

		return callErr
	}

	if !found.Output.IsServerStream() {
		return m.handleServerStreamOutput(stream, found, requestData, outputToUse, requestTime, matchNumber)
	}

	messages := outputToUse.Messages()
	if len(messages) == 0 {
		callErr := m.handleOutputError(stream.Context(), stream, outputToUse)

		m.recordServerStreamUnlessProxied(stream.Context(), found, requestTime,
			requestData, nil, recordedMetadata(outputToUse), callErr)

		return callErr //nolint:wrapcheck
	}

	sent, callErr := m.handleArrayStreamData(stream, found, messages, inputMsg, requestTime,
		matchNumber, found.Output.HasTemplate())
	if callErr == nil {
		callErr = m.handleOutputError(stream.Context(), stream, outputToUse)
	}

	m.recordServerStreamUnlessProxied(stream.Context(), found, requestTime,
		requestData, cleanStreamResponses(messages[:sent]), recordedMetadata(outputToUse), callErr)

	return callErr //nolint:wrapcheck
}

func streamDelaysPerMessage(found *stuber.Stub) bool {
	if found.ServerStreamHandler != nil {
		return false
	}

	return !found.Output.IsServerStream() || found.Output.HasTemplate() || len(found.Output.Messages()) > 0
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
	matchNumber int,
) error {
	callErr := m.handleNonArrayStreamData(stream, found, outputToUse, requestData, requestTime, matchNumber)

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
	elements []any,
	inputMsg *dynamicpb.Message,
	requestTime time.Time,
	matchNumber int,
	prerendered bool,
) (int, error) {
	done := stream.Context().Done()

	for i, streamData := range elements {
		select {
		case <-done:
			return i, stream.Context().Err()
		default:
		}

		err := m.handleStreamElement(stream, found, streamData, i, inputMsg, requestTime, matchNumber, prerendered)
		if err != nil {
			return i, err
		}
	}

	return len(elements), nil
}

func (m *grpcMocker) handleStreamElement(
	stream grpc.ServerStream,
	found *stuber.Stub,
	streamData any,
	i int,
	inputMsg *dynamicpb.Message,
	requestTime time.Time,
	matchNumber int,
	prerendered bool,
) error {
	payload := streamData

	var element stuber.GripMockElement

	if outputData, ok := streamData.(map[string]any); ok {
		outputDataCopy := deepCopyMapAny(outputData)
		element = stuber.ExtractGripMock(outputDataCopy)
		payload = outputDataCopy
	}

	requestData := m.convertToMap(inputMsg)

	headers := make(map[string]any)
	if md, ok := metadata.FromIncomingContext(stream.Context()); ok {
		headers = processHeaders(md)
	}

	templateData := newTemplateData(requestData, headers, i, requestTime,
		[]any{requestData}, found, matchNumber)

	err := delayTemplated(stream.Context(), m.templateEngine, elementDelay(found.Output.Delay, element), templateData)
	if err != nil {
		return err
	}

	if element.HasError {
		return m.streamElementError(element, templateData)
	}

	if !prerendered {
		rendered, err := renderData(m.templateEngine, payload, templateData)
		if err != nil {
			return err
		}

		payload = rendered
	}

	outputMsg, err := m.newOutputMessage(payload)
	if err != nil {
		return errors.Wrap(err, "failed to convert response to dynamic message")
	}

	return sendStreamMessage(stream, outputMsg)
}

func (m *grpcMocker) handleNonArrayStreamData(
	stream grpc.ServerStream,
	found *stuber.Stub,
	outputToUse stuber.Output,
	requestData map[string]any,
	requestTime time.Time,
	matchNumber int,
) error {
	err := m.handleOutputError(stream.Context(), stream, outputToUse)
	if err != nil {
		return err
	}

	done := stream.Context().Done()

	for {
		select {
		case <-done:
			return stream.Context().Err()
		default:
		}

		finished, err := m.sendNonArrayStreamReply(stream, found, requestData, requestTime, matchNumber)
		if err != nil {
			return err
		}

		if finished {
			return nil
		}
	}
}

// sendNonArrayStreamReply renders and sends one reply of a non-array server stream
// and reports whether the client finished the call afterwards.
func (m *grpcMocker) sendNonArrayStreamReply(
	stream grpc.ServerStream,
	found *stuber.Stub,
	requestData map[string]any,
	requestTime time.Time,
	matchNumber int,
) (bool, error) {
	msgData, msgTime := requestData, requestTime

	inputMsg := dynamicpb.NewMessage(m.inputDesc)

	err := stream.RecvMsg(inputMsg)
	if err == nil {
		msgData = m.convertToMap(inputMsg)
		msgTime = time.Now()
	}

	headers := make(map[string]any)
	if md, ok := metadata.FromIncomingContext(stream.Context()); ok {
		headers = processHeaders(md)
	}

	templateData := newTemplateData(msgData, headers, 0, msgTime,
		[]any{msgData}, found, matchNumber)

	err = delayTemplated(stream.Context(), m.templateEngine, found.Output.Delay, templateData)
	if err != nil {
		return false, err
	}

	outputDataCopy, err := m.renderSingleMessage(found.Output, templateData)
	if err != nil {
		return false, err
	}

	outputMsg, err := m.newOutputMessage(outputDataCopy)
	if err != nil {
		return false, errors.Wrap(err, "failed to convert response to dynamic message")
	}

	err = sendStreamMessage(stream, outputMsg)
	if err != nil {
		return false, err //nolint:wrapcheck
	}

	err = stream.RecvMsg(nil)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return true, nil
		}

		return false, errors.Wrap(err, "failed to receive message")
	}

	return false, nil
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

	err := enc.Encode(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal output to JSON: %w", err)
	}

	msg := dynamicpb.NewMessage(m.outputDesc)

	jsonBytes := pooled.Bytes()

	err = m.typeResolver.Unmarshal(jsonBytes, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON into dynamic message: %w (json=%s)", err, string(jsonBytes))
	}

	return msg, nil
}

const errStreamOutputOnUnary = "stub answers with a stream, but this method answers with a single message; use output.data"

// singleMessage is the payload of a stub that answers with one message; a stream stub
// has no single payload and yields nil.
func (m *grpcMocker) renderSingleMessage(output stuber.Output, templateData template.Data) (any, error) {
	if output.HasTemplate() {
		document, _ := output.Document()

		return renderDocumentData(m.templateEngine, document, templateData)
	}

	return renderData(m.templateEngine, singleMessage(output), templateData)
}

func singleMessage(output stuber.Output) any {
	messages := output.Messages()
	if len(messages) == 0 {
		return nil
	}

	return messages[0]
}
