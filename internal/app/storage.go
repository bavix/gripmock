package app

import (
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/goccy/go-json"
	"github.com/gripmock/stuber"
)

func stubNotFoundError(expect stuber.Query, result *stuber.Result) error {
	template := fmt.Sprintf("Can't find stub \n\nService: %s \n\nMethod: %s \n\nInput\n\n", expect.Service, expect.Method)

	expectString, err := json.MarshalIndent(expect.Data, "", "\t")
	if err != nil {
		return errors.Wrapf(err, "failed to marshal expect data")
	}

	template += string(expectString)

	if result.Similar() == nil {
		return fmt.Errorf("%s", template) //nolint:err113
	}

	addClosestMatch := func(key string, match map[string]any) {
		if len(match) > 0 {
			matchString, err := json.MarshalIndent(match, "", "\t")
			if err != nil {
				return
			}

			template += fmt.Sprintf("\n\nClosest Match \n\n%s:%s", key, matchString)
		}
	}

	addClosestMatch("equals", result.Similar().Input.Equals)
	addClosestMatch("contains", result.Similar().Input.Contains)
	addClosestMatch("matches", result.Similar().Input.Matches)

	return fmt.Errorf("%s", template) //nolint:err113
}

func stubNotFoundErrorV2(expect stuber.QueryV2, result *stuber.Result) error {
	template := fmt.Sprintf("Can't find stub \n\nService: %s \n\nMethod: %s \n\n", expect.Service, expect.Method)

	// Handle streaming input
	template += formatInputSection(expect.Input)

	if result.Similar() == nil {
		return fmt.Errorf("%s", template) //nolint:err113
	}

	// Add closest matches
	template += formatClosestMatches(result)

	return fmt.Errorf("%s", template) //nolint:err113
}

// formatInputSection formats the input section of the error message.
// It takes a slice of input messages (each represented as a map) and returns a formatted string.
// If there are multiple messages, it formats them as a stream; if one, as a single input; if none, indicates empty input.
func formatInputSection(input []map[string]any) string {
	switch {
	case len(input) > 1:
		return formatStreamInput(input)
	case len(input) == 1:
		return formatSingleInput(input[0])
	default:
		return "Input: (empty)\n\n"
	}
}

// formatStreamInput formats multiple input messages.
// It iterates through each message in the input slice and formats them with proper indentation.
// Each message is numbered and formatted as JSON for better readability.
func formatStreamInput(input []map[string]any) string {
	result := "Stream Input (multiple messages):\n\n"
	for i, msg := range input {
		result += fmt.Sprintf("Message %d:\n", i)

		expectString, err := json.MarshalIndent(msg, "", "\t")
		if err != nil {
			continue
		}

		result += string(expectString) + "\n\n"
	}

	return result
}

// formatSingleInput formats a single input message.
// It takes a single message map and formats it as JSON with proper indentation.
// Returns a formatted string with "Input:" prefix for consistency.
func formatSingleInput(input map[string]any) string {
	result := "Input:\n\n"

	expectString, err := json.MarshalIndent(input, "", "\t")
	if err != nil {
		return result
	}

	return result + string(expectString) + "\n\n"
}

// formatClosestMatches formats the closest matches section.
// It processes the similar stub result and formats the closest matches for either stream or regular input.
// Returns a formatted string containing the closest matching stub information.
func formatClosestMatches(result *stuber.Result) string {
	addClosestMatch := func(key string, match map[string]any) string {
		if len(match) > 0 {
			matchString, err := json.MarshalIndent(match, "", "\t")
			if err != nil {
				return ""
			}

			return fmt.Sprintf("\n\nClosest Match \n\n%s:%s", key, matchString)
		}

		return ""
	}

	// Check if similar stub has stream input
	if len(result.Similar().Stream) > 0 {
		return formatStreamClosestMatches(result, addClosestMatch)
	}

	// Fallback to regular input matching
	var template string

	template += addClosestMatch("equals", result.Similar().Input.Equals)
	template += addClosestMatch("contains", result.Similar().Input.Contains)
	template += addClosestMatch("matches", result.Similar().Input.Matches)

	return template
}

// formatStreamClosestMatches formats closest matches for stream input.
// It processes stream-specific closest matches and formats them with proper message numbering.
// The addClosestMatch function is used to format individual match types (equals, contains, matches).
func formatStreamClosestMatches(result *stuber.Result, addClosestMatch func(string, map[string]any) string) string {
	template := "\n\nSimilar stub found with stream input:\n"
	for i, streamInput := range result.Similar().Stream {
		template += fmt.Sprintf("\nStream Message %d:\n", i)

		if streamInput.Equals != nil {
			template += addClosestMatch("equals", streamInput.Equals)
		}

		if streamInput.Contains != nil {
			template += addClosestMatch("contains", streamInput.Contains)
		}

		if streamInput.Matches != nil {
			template += addClosestMatch("matches", streamInput.Matches)
		}
	}

	return template
}
