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
	if len(expect.Input) > 1 {
		template += "Stream Input (multiple messages):\n\n"
		for i, input := range expect.Input {
			template += fmt.Sprintf("Message %d:\n", i)
			expectString, err := json.MarshalIndent(input, "", "\t")
			if err != nil {
				return errors.Wrapf(err, "failed to marshal expect data for message %d", i)
			}
			template += string(expectString) + "\n\n"
		}
	} else if len(expect.Input) == 1 {
		template += "Input:\n\n"
		expectString, err := json.MarshalIndent(expect.Input[0], "", "\t")
		if err != nil {
			return errors.Wrapf(err, "failed to marshal expect data")
		}
		template += string(expectString) + "\n\n"
	} else {
		template += "Input: (empty)\n\n"
	}

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

	// Check if similar stub has stream input
	if result.Similar().Stream != nil && len(result.Similar().Stream) > 0 {
		template += "\n\nSimilar stub found with stream input:\n"
		for i, streamInput := range result.Similar().Stream {
			template += fmt.Sprintf("\nStream Message %d:\n", i)
			if streamInput.Equals != nil {
				addClosestMatch("equals", streamInput.Equals)
			}
			if streamInput.Contains != nil {
				addClosestMatch("contains", streamInput.Contains)
			}
			if streamInput.Matches != nil {
				addClosestMatch("matches", streamInput.Matches)
			}
		}
	} else {
		// Fallback to regular input matching
		addClosestMatch("equals", result.Similar().Input.Equals)
		addClosestMatch("contains", result.Similar().Input.Contains)
		addClosestMatch("matches", result.Similar().Input.Matches)
	}

	return fmt.Errorf("%s", template) //nolint:err113
}
