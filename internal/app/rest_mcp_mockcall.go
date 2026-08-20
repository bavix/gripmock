package app

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"

	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
)

func mcpMockCall(h *RestServer, args map[string]any) (map[string]any, error) {
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

	return mcpRenderMockResponse(h, found, service, method, session, input, headers, result.MatchNumber()), nil
}

//nolint:cyclop,funlen
func mcpRenderMockResponse(
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

	engine := h.templateEngine

	dataCopy := copyForTemplates(output.Data)
	if dataMap, ok := dataCopy.(map[string]any); ok {
		if err := engine.ProcessMap(dataMap, templateData); err != nil {
			return h.mockTemplateError(found, service, method, session, input, requestTime, err)
		}

		dataCopy = dataMap
	}

	if template.HasTemplatesInHeaders(output.Headers) {
		headersCopy := deepCopyStringMap(output.Headers)
		if err := engine.ProcessHeaders(headersCopy, templateData); err != nil {
			return h.mockTemplateError(found, service, method, session, input, requestTime, err)
		}

		output.Headers = headersCopy
	}

	if template.HasTemplatesInHeaders(output.Trailers) {
		trailersCopy := deepCopyStringMap(output.Trailers)
		if err := engine.ProcessHeaders(trailersCopy, templateData); err != nil {
			return h.mockTemplateError(found, service, method, session, input, requestTime, err)
		}

		output.Trailers = trailersCopy
	}

	if output.Error != "" && template.IsTemplateString(output.Error) {
		errorStr, err := engine.ProcessError(output.Error, templateData)
		if err != nil {
			return h.mockTemplateError(found, service, method, session, input, requestTime, err)
		}

		output.Error = errorStr
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

	var recordedData any

	if errMsg == "" {
		if dataCopy != nil {
			response["data"] = dataCopy
			recordedData = dataCopy
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
	h.effects().apply(context.Background(), found, templateData)

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
	data any,
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

	if dataMap, ok := data.(map[string]any); ok {
		record.Responses = []map[string]any{dataMap}
	}

	recordOwned(recorder, record)
}
