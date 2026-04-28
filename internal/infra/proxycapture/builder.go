package proxycapture

import (
	"strings"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/bavix/gripmock/v3/internal/infra/protoconv"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func ResponseHeaders(head metadata.MD, tail metadata.MD) map[string]string {
	if len(head) == 0 && len(tail) == 0 {
		return nil
	}

	out := make(map[string]string)

	appendHeaders := func(source metadata.MD) {
		for key, values := range source {
			if len(values) == 0 {
				continue
			}

			joined := strings.Join(values, ";")
			if previous, ok := out[key]; ok && previous != "" {
				out[key] = previous + ";" + joined
			} else {
				out[key] = joined
			}
		}
	}

	appendHeaders(head)
	appendHeaders(tail)

	if len(out) == 0 {
		return nil
	}

	return out
}

func BuildUnaryStub(
	service string,
	method string,
	session string,
	request map[string]any,
	requestHeaders map[string]any,
	response map[string]any,
	responseHeaders map[string]string,
	callErr error,
) *stuber.Stub {
	stub := &stuber.Stub{
		Service: service,
		Method:  method,
		Session: session,
		Source:  stuber.SourceProxy,
		Headers: stuber.InputHeader{Equals: requestHeaders},
		Input:   stuber.InputData{Equals: request},
		Output:  stuber.Output{Data: response, Headers: responseHeaders},
	}

	applyStatusError(&stub.Output, callErr, true)

	return stub
}

func BuildServerStreamStub(
	service string,
	method string,
	session string,
	request map[string]any,
	requestHeaders map[string]any,
	responses []map[string]any,
	responseHeaders map[string]string,
	callErr error,
) *stuber.Stub {
	stub := &stuber.Stub{
		Service: service,
		Method:  method,
		Session: session,
		Source:  stuber.SourceProxy,
		Headers: stuber.InputHeader{Equals: requestHeaders},
		Input:   stuber.InputData{Equals: request},
		Output:  stuber.Output{Stream: toStreamOutput(responses), Headers: responseHeaders},
	}

	applyStatusError(&stub.Output, callErr, false)

	return stub
}

func BuildClientStreamStub(
	service string,
	method string,
	session string,
	requests []map[string]any,
	requestHeaders map[string]any,
	response map[string]any,
	responseHeaders map[string]string,
	callErr error,
) *stuber.Stub {
	stub := &stuber.Stub{
		Service: service,
		Method:  method,
		Session: session,
		Source:  stuber.SourceProxy,
		Headers: stuber.InputHeader{Equals: requestHeaders},
		Inputs:  toInputs(requests),
		Output:  stuber.Output{Data: response, Headers: responseHeaders},
	}

	applyStatusError(&stub.Output, callErr, true)

	return stub
}

func BuildBidiStub(
	service string,
	method string,
	session string,
	requests []map[string]any,
	requestHeaders map[string]any,
	responses []map[string]any,
	responseHeaders map[string]string,
	callErr error,
) *stuber.Stub {
	stub := &stuber.Stub{
		Service: service,
		Method:  method,
		Session: session,
		Source:  stuber.SourceProxy,
		Headers: stuber.InputHeader{Equals: requestHeaders},
		Inputs:  toInputs(requests),
		Output:  stuber.Output{Stream: toStreamOutput(responses), Headers: responseHeaders},
	}

	applyStatusError(&stub.Output, callErr, false)

	return stub
}

func toInputs(requests []map[string]any) []stuber.InputData {
	inputs := make([]stuber.InputData, 0, len(requests))
	for _, request := range requests {
		inputs = append(inputs, stuber.InputData{Equals: request})
	}

	return inputs
}

func toStreamOutput(responses []map[string]any) []any {
	streamOutput := make([]any, 0, len(responses))
	for _, response := range responses {
		streamOutput = append(streamOutput, response)
	}

	return streamOutput
}

func applyStatusError(output *stuber.Output, callErr error, clearData bool) {
	if output == nil || callErr == nil {
		return
	}

	st := status.Convert(callErr)
	code := st.Code()

	output.Code = &code
	output.Error = st.Message()
	output.Details = statusDetailsToMaps(callErr)

	if clearData {
		output.Data = nil
	}
}

func statusDetailsToMaps(callErr error) []map[string]any {
	if callErr == nil {
		return nil
	}

	st := status.Convert(callErr)

	details := st.Details()
	if len(details) == 0 {
		return nil
	}

	out := make([]map[string]any, 0, len(details))
	for _, detail := range details {
		msg, ok := detail.(proto.Message)
		if !ok {
			continue
		}

		mapped := protoconv.ConvertToMap(msg)
		if mapped == nil {
			continue
		}

		typeURL := "type.googleapis.com/" + string(msg.ProtoReflect().Descriptor().FullName())
		mapped["type"] = typeURL

		out = append(out, mapped)
	}

	if len(out) == 0 {
		return nil
	}

	return out
}
