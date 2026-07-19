package app

//nolint:revive
import (
	"context"
	stderrors "errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
)

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
