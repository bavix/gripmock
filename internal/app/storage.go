package app

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"regexp"

	"github.com/goccy/go-json"
	"github.com/lithammer/fuzzysearch/fuzzy"

	"github.com/bavix/gripmock/pkg/storage"
)

var ErrNotFound = errors.New("not found")

type matchFunc func(interface{}, interface{}) bool

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
		stubStorage.MarkUsed(stubs[0].ID)

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

		stubStorage.MarkUsed(strange.ID)

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

	highestRank := struct {
		rank  float32
		match closeMatch
	}{0, closeMatch{}}

	for _, closeMatchValue := range closestMatches {
		rank := rankMatch(string(expectString), closeMatchValue.expect)

		// the higher the better
		if rank > highestRank.rank {
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

	template += fmt.Sprintf("\n\nClosest Match \n\n%s:%s", closestMatch.rule, closestMatchString)

	//fixme
	//nolint:goerr113
	return fmt.Errorf(template)
}

// we made our own simple ranking logic
// count the matches field_name and value then compare it with total field names and values
// the higher the better.
func rankMatch(expect string, closeMatch map[string]interface{}) float32 {
	occurrence := 0

	for key, value := range closeMatch {
		if fuzzy.Match(key+":", expect) {
			occurrence++
		}

		if fuzzy.Match(fmt.Sprint(value), expect) {
			occurrence++
		}
	}

	if occurrence == 0 {
		return 0
	}
	totalFields := len(closeMatch) * 2

	return float32(occurrence) / float32(totalFields)
}

func regexMatch(expect, actual interface{}) bool {
	var (
		expectedStr, expectedStringOk = expect.(string)
		actualStr, actualStringOk     = actual.(string)
	)

	if expectedStringOk && actualStringOk {
		match, err := regexp.MatchString(expectedStr, actualStr)
		if err != nil {
			log.Printf("Error on matching regex %s with %s error:%v\n", expect, actual, err)
		}

		return match
	}

	return reflect.DeepEqual(expect, actual)
}

func equals(expect, actual map[string]interface{}, ignoreArrayOrder bool) bool {
	return find(expect, actual, true, true, reflect.DeepEqual, ignoreArrayOrder)
}

func contains(expect, actual map[string]interface{}, ignoreArrayOrder bool) bool {
	return find(expect, actual, true, false, reflect.DeepEqual, ignoreArrayOrder)
}

func matches(expect, actual map[string]interface{}, ignoreArrayOrder bool) bool {
	return find(expect, actual, true, false, regexMatch, ignoreArrayOrder)
}

//nolint:cyclop
func find(expect, actual interface{}, acc, exactMatch bool, f matchFunc, ignoreArrayOrder bool) bool {
	// circuit brake
	if !acc {
		return false
	}

	//nolint:nestif
	if expectArrayValue, expectArrayOk := expect.([]interface{}); expectArrayOk {
		actualArrayValue, actualArrayOk := actual.([]interface{})
		if !actualArrayOk {
			return false
		}

		if exactMatch {
			if len(expectArrayValue) != len(actualArrayValue) {
				return false
			}
		} else if len(expectArrayValue) > len(actualArrayValue) {
			return false
		}

		if ignoreArrayOrder {
			return cmpValue(expectArrayValue, actualArrayValue, f)
		}

		for expectItemIndex, expectItemValue := range expectArrayValue {
			actualItemValue := actualArrayValue[expectItemIndex]
			acc = find(expectItemValue, actualItemValue, acc, exactMatch, f, ignoreArrayOrder)
		}

		return acc
	}

	//nolint:nestif
	if expectMapValue, expectMapOk := expect.(map[string]interface{}); expectMapOk {
		actualMapValue, actualMapOk := actual.(map[string]interface{})
		if !actualMapOk {
			return false
		}

		if exactMatch {
			if len(expectMapValue) != len(actualMapValue) {
				return false
			}
		} else if len(expectMapValue) > len(actualMapValue) {
			return false
		}

		for expectItemKey, expectItemValue := range expectMapValue {
			actualItemValue := actualMapValue[expectItemKey]
			acc = find(expectItemValue, actualItemValue, acc, exactMatch, f, ignoreArrayOrder)
		}

		return acc
	}

	return f(expect, actual)
}

func cmpValue(a, b []interface{}, f matchFunc) bool {
	if len(a) != len(b) {
		return false
	}

	if f(a, b) {
		return true
	}

	d := len(a)
	c := make([]interface{}, 0, d)

	usedA := make(map[int]bool, len(a))
	usedB := make(map[int]bool, len(b))

	for i := 0; i < d; i++ {
		for ia, va := range a {
			if usedA[ia] {
				continue
			}

			for ib, vb := range b {
				if usedB[ib] {
					continue
				}

				if f(va, vb) {
					c = append(c, va)

					usedA[ia] = true
					usedB[ib] = true
				}
			}
		}
	}

	return d == len(c)
}
