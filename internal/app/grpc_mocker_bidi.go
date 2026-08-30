package app

//nolint:revive
import (
	"context"
	"io"
	"maps"
	"time"

	"github.com/cockroachdb/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
)

const (
	errMsgProcessTemplates = "failed to process dynamic templates"
	errMsgConvertToDynamic = "failed to convert response to dynamic message"
)

//nolint:funlen
func (m *grpcMocker) handleBidiStream(stream grpc.ServerStream) error {
	queryBidi := m.newQueryBidi(stream.Context())

	stubs, _ := m.budgerigar.FindBy(queryBidi.Service, queryBidi.Method)
	for _, st := range stubs {
		if stuber.HandlerCandidate(st, queryBidi) {
			return st.Handler(stream.Context(), stream)
		}
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
		ServerStream:  stream,
		requests:      make([]map[string]any, 0, bidiRecordingStreamInitCap),
		responses:     make([]any, 0, bidiRecordingStreamResponsesCap),
		maxItems:      maxHistoryStreamMsgs,
		recordHeaders: m.recorder != nil,
	}

	requestTime := time.Now()

	defer func() {
		m.setResponseTrailersAny(stream.Context(), stream, recordingStream.stubTrailers)
	}()

	for {
		inputMsg := dynamicpb.NewMessage(m.inputDesc)

		err := receiveStreamMessage(recordingStream, inputMsg)
		if errors.Is(err, io.EOF) {
			m.recordBidiStream(recordingStream, bidiResult, requestTime, nil)

			return nil
		}

		if err != nil {
			m.recordBidiStreamUnlessProxied(recordingStream, bidiResult, requestTime, err)

			if status.Code(err) == codes.NotFound {
				return newBidiStreamFallbackError(err, []*dynamicpb.Message{inputMsg})
			}

			return err
		}

		err = m.processBidiStreamMessage(recordingStream, bidiResult, inputMsg)
		if err != nil {
			m.recordBidiStreamUnlessProxied(recordingStream, bidiResult, requestTime, err)

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
			if rec, ok := stream.(*bidiRecordingStream); ok && len(rec.getResponses()) > 0 {
				return status.Error(codes.NotFound, wrappedErr.Error())
			}

			return newBidiStreamFallbackError(wrappedErr, []*dynamicpb.Message{inputMsg})
		}

		return wrappedErr
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

	td := newTemplateData(requestData, headers, bidiResult.GetMessageIndex(), requestTime,
		[]any{requestData}, stub, bidiResult.MatchNumber())

	if !stub.Output.IsServerStream() {
		err := delayTemplated(stream.Context(), m.templateEngine, stub.Output.Delay, td)
		if err != nil {
			return err
		}
	}

	outputToUse, err := m.prepareBidiOutput(stub, td)
	if err != nil {
		return err
	}

	m.applyEffects(stream.Context(), stub, td)

	if bidiResult.GetMessageIndex() == 0 {
		err := m.setResponseHeadersAny(stream.Context(), stream, outputToUse.Headers)
		if err != nil {
			return errors.Wrap(err, "failed to set headers")
		}
	}

	if recStream, ok := stream.(*bidiRecordingStream); ok {
		recStream.mergeStubTrailers(outputToUse.Trailers)
	}

	err = m.handleOutputError(stream.Context(), stream, outputToUse)
	if err != nil {
		return err
	}

	if recStream, ok := stream.(*bidiRecordingStream); ok {
		recStream.setStubID(stub.ID)
	}

	return m.sendBidiResponses(stream, outputToUse, stub, bidiResult.GetMessageIndex(), td)
}

func (m *grpcMocker) recordBidiStreamUnlessProxied(
	stream *bidiRecordingStream,
	bidiResult *stuber.BidiResult,
	requestTime time.Time,
	callErr error,
) {
	if m.proxyFallbackWillServe(callErr) {
		return
	}

	m.recordBidiStream(stream, bidiResult, requestTime, callErr)
}

func (m *grpcMocker) recordBidiStream(
	stream *bidiRecordingStream,
	_ *stuber.BidiResult,
	requestTime time.Time,
	callErr error,
) {
	if m.recorder == nil {
		return
	}

	code := uint32(codes.OK)

	errMsg := ""

	if callErr != nil {
		code = uint32(status.Code(callErr))
		errMsg = callErr.Error()
	}

	requests := stream.getRequests()
	responses := stream.getResponses()

	rec := history.CallRecord{
		Service:         m.fullServiceName,
		Method:          m.methodName,
		Session:         sessionFromContext(stream.Context()),
		Requests:        requests,
		Responses:       responses,
		ResponseHeaders: stream.getResponseHeaders(),
		Code:            code,
		Error:           errMsg,
		StubID:          stream.getStubID(),
		ElapsedMS:       time.Since(requestTime).Milliseconds(),
		Timestamp:       requestTime,
	}

	recordOwned(m.recorder, rec)
}

func (m *grpcMocker) prepareBidiOutput(stub *stuber.Stub, templateData template.Data) (stuber.Output, error) {
	outputToUse, err := renderOutput(m.templateEngine, stub.Output, templateData,
		renderOptions{})
	if err != nil {
		return stuber.Output{}, status.Error(codes.Internal, err.Error())
	}

	outputToUse.Headers = maps.Clone(outputToUse.Headers)
	outputToUse.Trailers = maps.Clone(outputToUse.Trailers)
	outputToUse.Details = deepCopyDetails(outputToUse.Details)

	return outputToUse, nil
}

func (m *grpcMocker) sendBidiResponses(
	stream grpc.ServerStream,
	output stuber.Output,
	stub *stuber.Stub,
	messageIndex int,
	templateData template.Data,
) error {
	if !stub.Output.IsServerStream() {
		outputMsg, err := m.newOutputMessage(singleMessage(output))
		if err != nil {
			return errors.Wrap(err, errMsgConvertToDynamic)
		}

		return sendStreamMessage(stream, outputMsg)
	}

	messages := output.Messages()
	if len(messages) == 0 {
		return nil
	}

	return m.sendStreamResponses(stream, output, stub, messageIndex, messages, templateData)
}

func (m *grpcMocker) sendStreamResponses(
	stream grpc.ServerStream,
	output stuber.Output,
	stub *stuber.Stub,
	messageIndex int,
	messages []any,
	templateData template.Data,
) error {
	if stub.Output.HasTemplate() {
		return m.sendServerStreamResponses(stream, output, messages, templateData)
	}

	if stub.IsClientStream() {
		return m.sendClientStreamResponses(stream, output, stub, messageIndex, messages, templateData)
	}

	elements, err := renderStreamElements(m.templateEngine, messages, templateData)
	if err != nil {
		return err
	}

	return m.sendServerStreamResponses(stream, output, elements, templateData)
}

//nolint:cyclop
func (m *grpcMocker) sendClientStreamResponses(
	stream grpc.ServerStream,
	output stuber.Output,
	stub *stuber.Stub,
	messageIndex int,
	messages []any,
	templateData template.Data,
) error {
	streamLen := len(messages)
	if streamLen == 0 {
		return nil
	}

	if messageIndex < 0 {
		return nil
	}

	inputLen := len(stub.Matchers())
	if inputLen == 0 {
		return nil
	}

	if messageIndex >= inputLen || messageIndex >= streamLen {
		return exhaustedBidiScriptError(stub, messageIndex, inputLen)
	}

	start := messageIndex

	end := start + 1
	if messageIndex == inputLen-1 {
		end = streamLen
	}

	elements, err := renderStreamElements(m.templateEngine, messages[start:end], templateData)
	if err != nil {
		return err
	}

	for _, streamElement := range elements {
		if _, ok := streamElement.(map[string]any); !ok {
			continue
		}

		payload, err := m.prepareStreamElement(stream.Context(), streamElement, output, templateData)
		if err != nil {
			return err
		}

		outputMsg, err := m.newOutputMessage(payload)
		if err != nil {
			return errors.Wrap(err, errMsgConvertToDynamic)
		}

		err = sendStreamMessage(stream, outputMsg)
		if err != nil {
			return err //nolint:wrapcheck
		}
	}

	return nil
}

func exhaustedBidiScriptError(stub *stuber.Stub, messageIndex, inputLen int) error {
	return status.Errorf(codes.NotFound,
		"stub %s scripts %d message(s) for %s/%s, but the client sent message #%d; "+
			"declare another inputs/output.stream pair or stop sending",
		stub.ID, inputLen, stub.Service, stub.Method, messageIndex+1)
}

func (m *grpcMocker) prepareStreamElement(
	ctx context.Context,
	element any,
	output stuber.Output,
	td template.Data,
) (any, error) {
	data, ok := element.(map[string]any)
	if !ok {
		return element, nil
	}

	payload := copyForTemplates(data)

	var marker stuber.GripMockElement

	if _, marked := data[stuber.GripMockKey]; marked {
		copied := deepCopyMapAny(data)
		marker = stuber.ExtractGripMock(copied)
		payload = copied
	}

	if marker.HasError {
		return nil, m.streamElementError(marker, td)
	}

	err := delayTemplated(ctx, m.templateEngine, elementDelay(output.Delay, marker), td)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func (m *grpcMocker) sendServerStreamResponses(
	stream grpc.ServerStream,
	output stuber.Output,
	elements []any,
	templateData template.Data,
) error {
	for _, streamElement := range elements {
		payload, err := m.prepareStreamElement(stream.Context(), streamElement, output, templateData)
		if err != nil {
			return err
		}

		outputMsg, err := m.newOutputMessage(payload)
		if err != nil {
			return errors.Wrap(err, errMsgConvertToDynamic)
		}

		err = sendStreamMessage(stream, outputMsg)
		if err != nil {
			return err //nolint:wrapcheck
		}
	}

	return nil
}
