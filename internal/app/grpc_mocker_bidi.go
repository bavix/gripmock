package app

//nolint:revive
import (
	"io"
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

func (m *grpcMocker) sendStreamResponses(
	stream grpc.ServerStream,
	output stuber.Output,
	stub *stuber.Stub,
	messageIndex int,
	requestTime time.Time,
) error {
	if stub.IsClientStream() {
		return m.sendClientStreamResponses(stream, output, stub, messageIndex, requestTime)
	}

	return m.sendServerStreamResponses(stream, output)
}

//nolint:cyclop,funlen
func (m *grpcMocker) sendClientStreamResponses(
	stream grpc.ServerStream,
	output stuber.Output,
	stub *stuber.Stub,
	messageIndex int,
	requestTime time.Time,
) error {
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

func (m *grpcMocker) sendServerStreamResponses(
	stream grpc.ServerStream,
	output stuber.Output,
) error {
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
