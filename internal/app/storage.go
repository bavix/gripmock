package app

import (
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/gripmock/stuber"
)

func stubNotFoundError2(expect stuber.Query, result *stuber.Result) error {
	template := fmt.Sprintf("Can't find stub \n\nService: %s \n\nMethod: %s \n\nInput\n\n", expect.Service, expect.Method)

	expectString, err := sonic.ConfigFastest.MarshalIndent(expect.Data, "", "\t")
	if err != nil {
		return err
	}

	template += string(expectString)

	if result.Similar() == nil {
		// fixme
		//nolint:goerr113,perfsprint
		return fmt.Errorf(template)
	}

	if len(result.Similar().Input.Equals) > 0 {
		closestMatchString, err := sonic.ConfigFastest.MarshalIndent(result.Similar().Input.Equals, "", "\t")
		if err != nil {
			return err
		}

		template += fmt.Sprintf("\n\nClosest Match \n\n%s:%s", "equals", closestMatchString)
	}

	if len(result.Similar().Input.Contains) > 0 {
		closestMatchString, err := sonic.ConfigFastest.MarshalIndent(result.Similar().Input.Contains, "", "\t")
		if err != nil {
			return err
		}

		template += fmt.Sprintf("\n\nClosest Match \n\n%s:%s", "contains", closestMatchString)
	}

	if len(result.Similar().Input.Matches) > 0 {
		closestMatchString, err := sonic.ConfigFastest.MarshalIndent(result.Similar().Input.Matches, "", "\t")
		if err != nil {
			return err
		}

		template += fmt.Sprintf("\n\nClosest Match \n\n%s:%s", "matches", closestMatchString)
	}

	// fixme
	//nolint:goerr113,perfsprint
	return fmt.Errorf(template)
}
