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

const (
	errMsgProcessTemplates = "failed to process dynamic templates"
	errMsgConvertToDynamic = "failed to convert response to dynamic message"
)

//nolint:funlen
func (m *grpcMocker) handleBidiStream(stream grpc.ServerStream) error {
	queryBidi := m.newQueryBidi(stream.Context())

	// Check for a custom handler on ANY matching stub, not just stubs[0] —
	// FindBy ordering is not guaranteed, so a handler on a later stub would
	// otherwise be silently ignored.
	stubs, _ := m.budgerigar.FindBy(queryBidi.Service, queryBidi.Method)
	for _, st := range stubs {
		if st.Handler != nil {
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
		responses:     make([]map[string]any, 0, bidiRecordingStreamResponsesCap),
		maxItems:      maxHistoryStreamMsgs,
		recordHeaders: m.recorder != nil,
	}

	requestTime := time.Now()

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

		if err := m.processBidiStreamMessage(recordingStream, bidiResult, inputMsg); err != nil {
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
			// Once mock responses have been emitted for earlier messages, do NOT
			// fall back to the proxy — replaying only the failing message upstream
			// would interleave a mock response with a proxied one on one stream.
			if rec, ok := stream.(*bidiRecordingStream); ok && len(rec.getResponses()) > 0 {
				return status.Error(codes.NotFound, wrappedErr.Error())
			}

			return newBidiStreamFallbackError(wrappedErr, []*dynamicpb.Message{inputMsg})
		}

		return wrappedErr
	}

	if err := delayResponse(stream.Context(), stub.Output.Delay); err != nil {
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

	td := newTemplateData(requestData, headers, bidiResult.GetMessageIndex(), requestTime, []any{requestData}, stub.ID.String())

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

	return m.sendBidiResponses(stream, outputToUse, stub, bidiResult.GetMessageIndex())
}

// recordBidiStreamUnlessProxied records the failed bidi exchange unless the
// proxy will retry the call — then the proxy leg owns the history record and
// pre-recording the NotFound here would double-count it.
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

	// Derive the real gRPC status code from the error (a configured Output.Error
	// /Output.Code arrives as a status error), rather than hardcoding Unknown.
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
			return stuber.Output{}, errors.Wrap(err, errMsgProcessTemplates)
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
) error {
	if len(output.Stream) > 0 {
		return m.sendStreamResponses(stream, output, stub, messageIndex)
	}

	// output.Data was already rendered by prepareBidiOutput with the correct
	// request/index context. Re-rendering here (with an empty Request/State)
	// would reinterpret any literal {{...}} in the result and degrade templates.
	outputMsg, err := m.newOutputMessage(output.Data)
	if err != nil {
		return errors.Wrap(err, errMsgConvertToDynamic)
	}

	return sendStreamMessage(stream, outputMsg)
}

func (m *grpcMocker) sendStreamResponses(
	stream grpc.ServerStream,
	output stuber.Output,
	stub *stuber.Stub,
	messageIndex int,
) error {
	if stub.IsClientStream() {
		return m.sendClientStreamResponses(stream, output, stub, messageIndex)
	}

	return m.sendServerStreamResponses(stream, output)
}

//nolint:cyclop
func (m *grpcMocker) sendClientStreamResponses(
	stream grpc.ServerStream,
	output stuber.Output,
	stub *stuber.Stub,
	messageIndex int,
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

		if err := delayResponse(stream.Context(), delayDelay); err != nil {
			return err
		}

		// Stream elements were already rendered once by prepareBidiOutput with the
		// correct request context; do not re-render with an empty context here.
		outputMsg, err := m.newOutputMessage(streamDataCopy)
		if err != nil {
			return errors.Wrap(err, errMsgConvertToDynamic)
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
				if err := delayResponse(stream.Context(), delayDelay); err != nil {
					return err
				}
			}
		}

		outputMsg, err := m.newOutputMessage(streamDataCopy)
		if err != nil {
			return errors.Wrap(err, errMsgConvertToDynamic)
		}

		if err := sendStreamMessage(stream, outputMsg); err != nil {
			return err //nolint:wrapcheck
		}
	}

	return nil
}
