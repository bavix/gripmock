package repository

import (
	"strings"

	domain "github.com/bavix/gripmock/v3/internal/domain/types"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// ConvertStubs converts a slice of stuber.Stub to domain.Stub.
func ConvertStubs(stubs []*stuber.Stub) []domain.Stub {
	result := make([]domain.Stub, 0, len(stubs))
	for _, stub := range stubs {
		result = append(result, ConvertFromStuberStub(stub))
	}

	return result
}

// ConvertFromStuberStub converts stuber.Stub to domain.Stub.
// This is the complete implementation that handles both v4 and legacy formats.
func ConvertFromStuberStub(stub *stuber.Stub) domain.Stub {
	outputsRaw := convertOutputs(stub)
	inputs := convertInputs(stub)

	return domain.Stub{
		ID:               stub.ID.String(),
		Service:          stub.Service,
		Method:           stub.Method,
		Priority:         stub.Priority,
		Times:            stub.Times,
		ResponseHeaders:  stub.ResponseHeaders,
		ResponseTrailers: stub.ResponseTrailers,
		OutputsRaw:       outputsRaw,
		Inputs:           inputs,
	}
}

// convertOutputs converts stub outputs from legacy to v4 format.
func convertOutputs(stub *stuber.Stub) []map[string]any {
	// Use v4 fields if available
	if len(stub.OutputsRawV4) > 0 {
		return stub.OutputsRawV4
	}

	// Convert legacy Output to v4 outputs
	if stub.Output.Data != nil || stub.Output.Error != "" || len(stub.Output.Stream) > 0 {
		output := make(map[string]any)

		if stub.Output.Data != nil {
			output["data"] = stub.Output.Data
		}

		if stub.Output.Error != "" {
			output["error"] = stub.Output.Error
		}

		if stub.Output.Code != nil {
			output["code"] = stub.Output.Code
		}

		if len(stub.Output.Stream) > 0 {
			output["stream"] = stub.Output.Stream
		}

		return []map[string]any{output}
	}

	return nil
}

// convertInputs converts stub inputs from legacy to v4 format.
func convertInputs(stub *stuber.Stub) []domain.Matcher {
	// Use v4 fields if available
	if len(stub.InputsV4) > 0 {
		return stub.InputsV4
	}

	var inputs []domain.Matcher

	// Convert legacy Input/Inputs to v4 inputs
	if len(stub.Inputs) > 0 {
		// Client streaming - convert Inputs
		inputs = convertInputsArray(stub.Inputs)
	} else if stub.Input.Equals != nil || stub.Input.Contains != nil || stub.Input.Matches != nil {
		// Unary - convert Input
		inputs = []domain.Matcher{convertInputData(stub.Input)}
	}

	return inputs
}

// convertInputsArray converts an array of InputData to Matcher array.
func convertInputsArray(inputs []stuber.InputData) []domain.Matcher {
	result := make([]domain.Matcher, 0, len(inputs))
	for _, input := range inputs {
		result = append(result, convertInputData(input))
	}

	return result
}

// convertInputData converts a single InputData to Matcher.
func convertInputData(input stuber.InputData) domain.Matcher {
	// Convert Matches from map[string]any to map[string]string
	matches := make(map[string]string)
	for k, v := range input.Matches {
		if str, ok := v.(string); ok {
			matches[k] = str
		}
	}

	return domain.Matcher{
		Equals:           input.Equals,
		Contains:         input.Contains,
		Matches:          matches,
		IgnoreArrayOrder: input.IgnoreArrayOrder,
	}
}

// ContainsIgnoreCase checks if a string contains another string (case-insensitive).
func ContainsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
