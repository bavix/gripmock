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
	template := fmt.Sprintf("Can't find stub \n\nService: %s \n\nMethod: %s \n\nInput\n\n", expect.Service, expect.Method)

	if len(expect.Input) > 0 {
		expectString, err := json.MarshalIndent(expect.Input[0], "", "\t")
		if err != nil {
			return errors.Wrapf(err, "failed to marshal expect data")
		}

		template += string(expectString)
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

	addClosestMatch("equals", result.Similar().Input.Equals)
	addClosestMatch("contains", result.Similar().Input.Contains)
	addClosestMatch("matches", result.Similar().Input.Matches)

	return fmt.Errorf("%s", template) //nolint:err113
}
