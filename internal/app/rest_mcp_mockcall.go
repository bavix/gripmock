package app

import (
	"time"

	"google.golang.org/grpc/codes"

	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
)

// mcpMockCall executes a mock invocation against the stub engine: it matches a
// stub for service+method+payload, renders its templated output (data, headers,
// error) exactly as the gRPC data plane would, records the call to history, and
// returns the rendered response with its status code.
//
// Two data-plane concerns are intentionally NOT reproduced here (they live in the
// gRPC/gateway transport and would require the proto descriptor codec + the full
// grpcMocker): protobuf shape validation of the payload against the method schema,
// and stub effects (state mutations / upsert / delete side effects). For a fully
// faithful call including those, invoke the gateway endpoint POST /{service}/{method}.
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

	return mcpRenderMockResponse(h, found, service, method, session, input, headers), nil
}

// mcpRenderMockResponse renders the matched stub's output and records the call.
//
//nolint:cyclop,funlen
func mcpRenderMockResponse(
	h *RestServer,
	found *stuber.Stub,
	service, method, session string,
	input []map[string]any,
	headers map[string]any,
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

	templateData := newTemplateData(firstRequest, headers, 0, requestTime, requests, found.ID.String())

	// Reuse the server's engine (built with the server's context at startup) —
	// no context is created per request.
	engine := h.templateEngine

	dataCopy, err := renderOutputData(engine, output, templateData)
	if err != nil {
		return h.mockTemplateError(found, service, method, session, input, requestTime, err)
	}

	streamCopy, err := renderMCPStream(engine, output, templateData)
	if err != nil {
		return h.mockTemplateError(found, service, method, session, input, requestTime, err)
	}

	if streamCopy != nil {
		output.Stream = streamCopy
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

	// Derive status exactly as the unary handler: an OK status ignores any error
	// string and returns the data message; a non-OK status returns the error and
	// no data message. Response headers are set in both cases.
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
		if len(output.Stream) > 0 {
			response["stream"] = output.Stream
			recordedData = output.Stream
		} else if dataCopy != nil {
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

	if delay := time.Duration(output.Delay); delay > 0 {
		response["delayMs"] = delay.Milliseconds()
	}

	h.recordMockCall(found, service, method, session, input, recordedData, uint32(code), errMsg, requestTime)

	return response
}

func renderMCPStream(engine *template.Engine, output stuber.Output, templateData template.Data) ([]any, error) {
	streamCopy, hasStreamTemplate, err := renderOutputStreamTemplate(engine, output, templateData)
	if err != nil || hasStreamTemplate {
		return streamCopy, err
	}

	if len(output.Stream) == 0 {
		return nil, nil
	}

	streamCopy = deepCopySliceAny(output.Stream)
	for i, item := range streamCopy {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		if err := engine.ProcessMap(itemMap, templateData); err != nil {
			return nil, err
		}

		streamCopy[i] = itemMap
	}

	return streamCopy, nil
}

// mockTemplateError mirrors the gRPC handler returning codes.Internal when a
// response template fails to render, and records the failed call.
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

// recordMockCall appends the invocation to history when the store is writable,
// so a mock_call is observable via history_list and marks the stub used.
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

	switch rendered := data.(type) {
	case map[string]any:
		record.Responses = []map[string]any{rendered}
	case []any:
		for _, item := range rendered {
			if itemMap, ok := item.(map[string]any); ok {
				record.Responses = append(record.Responses, itemMap)
			}
		}
	}

	recorder.Record(record)
}
