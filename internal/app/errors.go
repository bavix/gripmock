package app

import (
	"encoding/json"
	"fmt"

	"github.com/gripmock/stuber"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorFormatter provides methods for formatting error messages.
type ErrorFormatter struct{}

// NewErrorFormatter creates a new ErrorFormatter instance.
func NewErrorFormatter() *ErrorFormatter {
	return &ErrorFormatter{}
}

// FormatStubNotFoundErrorV2 formats error messages for V2 API stub not found scenarios.
func (f *ErrorFormatter) FormatStubNotFoundErrorV2(expect stuber.QueryV2, result *stuber.Result) error {
	template := fmt.Sprintf("Can't find stub \n\nService: %s \n\nMethod: %s \n\n", expect.Service, expect.Method)

	// Handle streaming input
	template += f.formatInputSection(expect.Input)

	if result.Similar() == nil {
		return fmt.Errorf("%s", template) //nolint:err113
	}

	// Add closest matches
	template += f.formatClosestMatches(result)

	return fmt.Errorf("%s", template) //nolint:err113
}

// FormatStubNotFoundError formats error messages for V1 API stub not found scenarios.
func (f *ErrorFormatter) FormatStubNotFoundError(expect stuber.Query, result *stuber.Result) error {
	template := fmt.Sprintf("Can't find stub \n\nService: %s \n\nMethod: %s \n\n", expect.Service, expect.Method)

	// Handle input
	template += f.formatSingleInput(expect.Data)

	if result.Similar() == nil {
		return fmt.Errorf("%s", template) //nolint:err113
	}

	// Add closest matches
	template += f.formatClosestMatches(result)

	return fmt.Errorf("%s", template) //nolint:err113
}

// CreateStubNotFoundError creates a gRPC status error for stub not found scenarios.
func (f *ErrorFormatter) CreateStubNotFoundError(serviceName, methodName string, details ...string) error {
	msg := fmt.Sprintf("Failed to find response (service: %s, method: %s)", serviceName, methodName)

	if len(details) > 0 {
		msg += " - " + details[0]
	}

	return status.Error(codes.NotFound, msg)
}

// CreateClientStreamError creates a gRPC status error for client stream scenarios.
func (f *ErrorFormatter) CreateClientStreamError(serviceName, methodName string, err error) error {
	msg := fmt.Sprintf("Failed to find response for client stream (service: %s, method: %s)", serviceName, methodName)

	if err != nil {
		msg += fmt.Sprintf(" - Error: %v", err)
	}

	return status.Error(codes.NotFound, msg)
}

// formatInputSection formats the input section of the error message.
func (f *ErrorFormatter) formatInputSection(input []map[string]any) string {
	switch {
	case len(input) > 1:
		return f.formatStreamInput(input)
	case len(input) == 1:
		return f.formatSingleInput(input[0])
	default:
		return "Input: (empty)\n\n"
	}
}

// formatStreamInput formats multiple input messages.
func (f *ErrorFormatter) formatStreamInput(input []map[string]any) string {
	result := "Stream Input (multiple messages):\n\n"
	for i, msg := range input {
		result += fmt.Sprintf("Message %d:\n", i)

		expectString, err := json.MarshalIndent(msg, "", "\t")
		if err != nil {
			// If JSON marshaling fails, include the raw message as fallback
			result += fmt.Sprintf("Error marshaling message: %v\nRaw message: %+v\n\n", err, msg)

			continue
		}

		result += string(expectString) + "\n\n"
	}

	return result
}

// formatSingleInput formats a single input message.
func (f *ErrorFormatter) formatSingleInput(input map[string]any) string {
	result := "Input:\n\n"

	expectString, err := json.MarshalIndent(input, "", "\t")
	if err != nil {
		// If JSON marshaling fails, include the raw message as fallback
		return result + fmt.Sprintf("Error marshaling input: %v\nRaw input: %+v\n\n", err, input)
	}

	return result + string(expectString) + "\n\n"
}

// formatClosestMatches formats the closest matches section.
func (f *ErrorFormatter) formatClosestMatches(result *stuber.Result) string {
	addClosestMatch := func(key string, match map[string]any) string {
		if len(match) > 0 {
			matchString, err := json.MarshalIndent(match, "", "\t")
			if err != nil {
				// If JSON marshaling fails, include the raw match as fallback
				return fmt.Sprintf("\n\nClosest Match \n\n%s: Error marshaling match: %v\nRaw match: %+v", key, err, match)
			}

			return fmt.Sprintf("\n\nClosest Match \n\n%s:%s", key, matchString)
		}

		return ""
	}

	// Check if similar stub has stream input
	similar := result.Similar()
	if similar != nil && len(similar.Stream) > 0 {
		return f.formatStreamClosestMatches(similar, addClosestMatch)
	}

	// Fallback to regular input matching
	var template string

	template += addClosestMatch("equals", result.Similar().Input.Equals)
	template += addClosestMatch("contains", result.Similar().Input.Contains)
	template += addClosestMatch("matches", result.Similar().Input.Matches)

	return template
}

// formatStreamClosestMatches formats closest matches for stream input.
func (f *ErrorFormatter) formatStreamClosestMatches(stub *stuber.Stub, addClosestMatch func(string, map[string]any) string) string {
	var template string

	for i, streamMsg := range stub.Stream {
		// Convert InputData to map representation
		streamData := map[string]any{
			"equals":   streamMsg.Equals,
			"contains": streamMsg.Contains,
			"matches":  streamMsg.Matches,
		}
		template += addClosestMatch(fmt.Sprintf("stream[%d]", i), streamData)
	}

	return template
}
