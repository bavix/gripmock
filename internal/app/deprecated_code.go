package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"log"
	"reflect"
	"regexp"

	"github.com/lithammer/fuzzysearch/fuzzy"

	"github.com/bavix/gripmock/pkg/storage"
)

type matchFunc func(interface{}, interface{}) bool

type closeMatch struct {
	rule   string
	expect map[string]interface{}
}

func findStub(stubStorage *storage.StubStorage, stub *findStubPayload) (*storage.Output, error) {
	stubs, err := stubStorage.ItemsBy(stub.Service, stub.Method)
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

	var closestMatch []closeMatch
	for _, strange := range stubs {
		if expect := strange.Input.Equals; expect != nil {
			closestMatch = append(closestMatch, closeMatch{"equals", expect})
			if equals(stub.Data, expect) {
				return &strange.Output, nil
			}
		}

		if expect := strange.Input.Contains; expect != nil {
			closestMatch = append(closestMatch, closeMatch{"contains", expect})
			if contains(strange.Input.Contains, stub.Data) {
				return &strange.Output, nil
			}
		}

		if expect := strange.Input.Matches; expect != nil {
			closestMatch = append(closestMatch, closeMatch{"matches", expect})
			if matches(strange.Input.Matches, stub.Data) {
				return &strange.Output, nil
			}
		}
	}

	return nil, stubNotFoundError(stub, closestMatch)
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
	var expectedStr, expectedStringOk = expect.(string)
	var actualStr, actualStringOk = actual.(string)

	if expectedStringOk && actualStringOk {
		match, err := regexp.Match(expectedStr, []byte(actualStr))
		if err != nil {
			log.Printf("Error on matching regex %s with %s error:%v\n", expect, actual, err)
		}
		return match
	}

	return reflect.DeepEqual(expect, actual)
}

func equals(expect, actual interface{}) bool {
	return find(expect, actual, true, true, reflect.DeepEqual)
}

func contains(expect, actual interface{}) bool {
	return find(expect, actual, true, false, reflect.DeepEqual)
}

func matches(expect, actual interface{}) bool {
	return find(expect, actual, true, false, regexMatch)
}

func find(expect, actual interface{}, acc, exactMatch bool, f matchFunc) bool {
	// circuit brake
	if !acc {
		return false
	}

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

		for expectItemIndex, expectItemValue := range expectArrayValue {
			actualItemValue := actualArrayValue[expectItemIndex]
			acc = find(expectItemValue, actualItemValue, acc, exactMatch, f)
		}

		return acc
	}

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
			acc = find(expectItemValue, actualItemValue, acc, exactMatch, f)
		}

		return acc
	}

	return f(expect, actual)
}

func validateStub(stub *storage.Stub) error {
	if stub.Service == "" {
		//fixme
		//nolint:goerr113
		return fmt.Errorf("service name can't be empty")
	}

	if stub.Method == "" {
		return fmt.Errorf("method name can't be emtpy")
	}

	// due to golang implementation
	// method name must capital
	title := cases.Title(language.English, cases.NoLower)
	stub.Method = title.String(stub.Method)

	switch {
	case stub.Input.Contains != nil:
		break
	case stub.Input.Equals != nil:
		break
	case stub.Input.Matches != nil:
		break
	default:
		//fixme
		//nolint:goerr113
		return fmt.Errorf("input cannot be empty")
	}

	// TODO: validate all input case

	if stub.Output.Error == "" && stub.Output.Data == nil && stub.Output.Code == nil {
		//fixme
		//nolint:goerr113
		return fmt.Errorf("output can't be empty")
	}

	return nil
}
