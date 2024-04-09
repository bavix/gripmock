package app

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gripmock/deeply"

	"github.com/bavix/gripmock/internal/pkg/features"
	"github.com/bavix/gripmock/pkg/storage"
)

var ErrNotFound = errors.New("not found")

type closeMatch struct {
	rule         string
	expect       map[string]interface{}
	headerRule   string
	headerExpect map[string]interface{}
}

//nolint:cyclop
func findStub(stubStorage *storage.StubStorage, stub *findStubPayload) (*storage.Output, error) {
	stubs, err := stubStorage.ItemsBy(stub.Service, stub.Method, stub.ID)
	if errors.Is(err, storage.ErrServiceNotFound) {
		//fixme
		//nolint:goerr113
		return nil, fmt.Errorf("can't find stub for Service: %s", stub.Service)
	}

	if errors.Is(err, storage.ErrMethodNotFound) {
		//fixme
		//nolint:goerr113
		return nil, fmt.Errorf("can't find stub for Service:%s and Method:%s", stub.Service, stub.Method)
	}

	if len(stubs) == 0 {
		//fixme
		//nolint:goerr113
		return nil, fmt.Errorf("stub for Service:%s and Method:%s is empty", stub.Service, stub.Method)
	}

	if stub.ID != nil {
		if !stub.features.Has(features.RequestInternal) {
			stubStorage.MarkUsed(stubs[0].ID)
		}

		return &stubs[0].Output, nil
	}

	var closestMatch []closeMatch

	for _, strange := range stubs {
		cmpData, cmpDataErr := inputCmp(strange.Input, stub.Data, strange.Input.IgnoreArrayOrder)
		if cmpDataErr != nil {
			if cmpData != nil {
				closestMatch = append(closestMatch, *cmpData)
			}

			continue
		}

		if strange.CheckHeaders() {
			if cmpHeaders, cmpHeadersErr := inputCmp(strange.Headers, stub.Headers, false); cmpHeadersErr != nil {
				if cmpHeaders != nil {
					closestMatch = append(closestMatch, closeMatch{
						rule:         cmpData.rule,
						expect:       cmpData.expect,
						headerRule:   cmpHeaders.rule,
						headerExpect: cmpHeaders.expect,
					})
				}

				continue
			}
		}

		if !stub.features.Has(features.RequestInternal) {
			stubStorage.MarkUsed(strange.ID)
		}

		return &strange.Output, nil
	}

	return nil, stubNotFoundError(stub, closestMatch)
}

func inputCmp(input storage.InputInterface, data map[string]interface{}, ignoreArrayOrder bool) (*closeMatch, error) {
	if expect := input.GetEquals(); expect != nil {
		closeMatchVal := closeMatch{rule: "equals", expect: expect}

		if equals(input.GetEquals(), data, ignoreArrayOrder) {
			return &closeMatchVal, nil
		}

		return &closeMatchVal, ErrNotFound
	}

	if expect := input.GetContains(); expect != nil {
		closeMatchVal := closeMatch{rule: "contains", expect: expect}

		if contains(input.GetContains(), data, ignoreArrayOrder) {
			return &closeMatchVal, nil
		}

		return &closeMatchVal, ErrNotFound
	}

	if expect := input.GetMatches(); expect != nil {
		closeMatchVal := closeMatch{rule: "matches", expect: expect}

		if matches(input.GetMatches(), data, ignoreArrayOrder) {
			return &closeMatchVal, nil
		}

		return &closeMatchVal, ErrNotFound
	}

	return nil, ErrNotFound
}

func stubNotFoundError(stub *findStubPayload, closestMatches []closeMatch) error {
	highestRank := struct {
		rank  float64
		match closeMatch
	}{0, closeMatch{}}

	for _, closeMatchValue := range closestMatches {
		if rank := deeply.RankMatch(stub.Data, closeMatchValue.expect); rank > highestRank.rank {
			highestRank.rank = rank
			highestRank.match = closeMatchValue
		}
	}

	var closestMatch closeMatch
	if highestRank.rank == 0 {
		closestMatch = closestMatches[0]
	} else {
		closestMatch = highestRank.match
	}

	closestMatchString, err := json.MarshalIndent(closestMatch.expect, "", "\t")
	if err != nil {
		return err
	}

	template := fmt.Sprintf("Can't find stub \n\nService: %s \n\nMethod: %s \n\nInput\n\n", stub.Service, stub.Method)
	expectString, err := json.MarshalIndent(stub.Data, "", "\t")
	if err != nil {
		return err
	}

	template += string(expectString)

	if len(closestMatches) == 0 {
		//fixme
		//nolint:goerr113
		return fmt.Errorf(template)
	}
	template += fmt.Sprintf("\n\nClosest Match \n\n%s:%s", closestMatch.rule, closestMatchString)

	//fixme
	//nolint:goerr113
	return fmt.Errorf(template)
}

func equals(expect, actual map[string]interface{}, ignoreArrayOrder bool) bool {
	if ignoreArrayOrder {
		return deeply.EqualsIgnoreArrayOrder(expect, actual)
	}

	return deeply.Equals(expect, actual)
}

func contains(expect, actual map[string]interface{}, ignoreArrayOrder bool) bool {
	if ignoreArrayOrder {
		return deeply.ContainsIgnoreArrayOrder(expect, actual)
	}

	return deeply.Contains(expect, actual)
}

func matches(expect, actual map[string]interface{}, ignoreArrayOrder bool) bool {
	if ignoreArrayOrder {
		return deeply.MatchesIgnoreArrayOrder(expect, actual)
	}

	return deeply.Matches(expect, actual)
}
