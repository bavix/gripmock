package app

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"

	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func mcpMockCall(ctx context.Context, h *RestServer, args map[string]any) (map[string]any, error) {
	service, _ := args["service"].(string)
	if service == "" {
		return nil, mcpRequiredArgError("service")
	}

	method, _ := args["method"].(string)
	if method == "" {
		return nil, mcpRequiredArgError("method")
	}

	input, err := mcpSearchInput(args)
	if err != nil {
		return nil, err
	}

	headers, err := mcpHeadersArg(args)
	if err != nil {
		return nil, err
	}

	session, _ := args["session"].(string)

	result, searchErr := h.budgerigar.FindByQuery(stuber.Query{
		Service: service,
		Method:  method,
		Session: session,
		Headers: headers,
		Input:   input,
	})
	if searchErr != nil {
		return mcpSearchNotMatchedResponse(searchErr), nil
	}

	found := result.Found()
	if found == nil {
		response := map[string]any{"matched": false}
		if similar := result.Similar(); similar != nil {
			response["similarStubId"] = similar.ID.String()
		}

		return response, nil
	}

	return mcpRenderMockResponse(ctx, h, found, service, method, session, input, headers, result.MatchNumber()), nil
}

//nolint:cyclop,funlen
func mcpRenderMockResponse(
	ctx context.Context,
	h *RestServer,
	found *stuber.Stub,
	service, method, session string,
	input []map[string]any,
	headers map[string]any,
	matchNumber int,
) map[string]any {
	requestTime := time.Now()
	output := found.Output

	requests := make([]any, len(input))
	for i, msg := range input {
		requests[i] = msg
	}

	var firstRequest map[string]any
	if len(input) > 0 {
		firstRequest = input[0]
	}

	templateData := newTemplateData(firstRequest, headers, 0, requestTime,
		requests, found, matchNumber)

	output, err := renderOutput(h.templateEngine, output, templateData,
		renderOptions{renderStream: true})
	if err != nil {
		return h.mockTemplateError(found, service, method, session, input, requestTime, err)
	}

	if err := delayTemplated(ctx, h.templateEngine, output.Delay, templateData); err != nil {
		return h.mockTemplateError(found, service, method, session, input, requestTime, err)
	}

	code := codes.OK
	errMsg := ""

	if st := outputStatusBase(output); st != nil {
		code = st.Code()
		errMsg = st.Message()
	}

	response := map[string]any{
		"matched":  true,
		"stubId":   found.ID.String(),
		"code":     uint32(code),
		"codeName": code.String(),
	}

	var recordedData []any

	if errMsg == "" {
		messages := output.Messages()

		switch {
		case output.IsServerStream():
			response["stream"] = messages
			recordedData = cleanStreamResponses(messages)
		case len(messages) > 0:
			response["data"] = messages[0]
			recordedData = messages
		}
	} else {
		response["error"] = errMsg

		if len(output.Details) > 0 {
			response["details"] = output.Details
		}
	}

	if len(output.Headers) > 0 {
		response["headers"] = output.Headers
	}

	if len(output.Trailers) > 0 {
		response["trailers"] = output.Trailers
	}

	resolvedDelay, _ := resolveDelay(h.templateEngine, output.Delay, templateData)
	if delay := time.Duration(resolvedDelay); delay > 0 {
		response["delayMs"] = delay.Milliseconds()
	}

	h.recordMockCall(found, service, method, session, input, recordedData, uint32(code), errMsg, requestTime)
	h.effects().apply(ctx, found, templateData)

	return response
}

func (h *RestServer) mockTemplateError(
	found *stuber.Stub,
	service, method, session string,
	input []map[string]any,
	requestTime time.Time,
	cause error,
) map[string]any {
	errMsg := "failed to process templates: " + cause.Error()

	h.recordMockCall(found, service, method, session, input, nil, uint32(codes.Internal), errMsg, requestTime)

	return map[string]any{
		"matched":  true,
		"stubId":   found.ID.String(),
		"code":     uint32(codes.Internal),
		"codeName": codes.Internal.String(),
		"error":    errMsg,
	}
}

func (h *RestServer) recordMockCall(
	found *stuber.Stub,
	service, method, session string,
	input []map[string]any,
	responses []any,
	code uint32,
	errMsg string,
	requestTime time.Time,
) {
	recorder, ok := h.history.(history.Recorder)
	if !ok {
		return
	}

	record := history.CallRecord{
		StubID:    found.ID,
		Service:   service,
		Method:    method,
		Session:   session,
		Requests:  input,
		Code:      code,
		Error:     errMsg,
		Timestamp: requestTime,
	}

	for _, response := range responses {
		if dataMap, ok := response.(map[string]any); ok {
			record.Responses = append(record.Responses, dataMap)
		}
	}

	recordOwned(recorder, record)
}
