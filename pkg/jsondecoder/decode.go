package jsondecoder

import (
	"bytes"
	stdjson "encoding/json"

	"github.com/gripmock/json"
)

const minJSONLength = 2

// UnmarshalSlice is a function that parses JSON data into a slice of the provided interface.
// It handles the case where the input data is not a JSON array by wrapping it in an array.
//
// Examples:
//
//	data := []byte(`{"name": "Bob"}`)
//	var result []map[string]interface{}
//	err := UnmarshalSlice(data, &result)
//	// result is now [{"name": "Bob"}]
//
//	data := []byte(`{"name": "Bob"}`)
//	var result []map[string]string
//	err := UnmarshalSlice(data, &result)
//	// result is now [{"name": "Bob"}]
//
//	data := []byte(`{"name": "Bob"}`)
//	var result []interface{}
//	err := UnmarshalSlice(data, &result)
//	// result is now [{"name": "Bob"}]
//
//	data := []byte(`{"name": "Bob"}`)
//	var result []map[string]string
//	err := UnmarshalSlice(data, &result)
//	// result is now [{"name": "Bob"}]
//	// NOTE: if the input data is not a JSON array, it is wrapped in an array before decoding
func UnmarshalSlice(data []byte, v interface{}) error {
	input := bytes.TrimSpace(data)

	if len(input) < minJSONLength {
		return &stdjson.SyntaxError{}
	}

	// If the input is not a JSON array, wrap it in an array
	if len(input) > 0 && input[0] == '{' && input[len(input)-1] == '}' {
		input = append(append([]byte{'['}, input...), ']')
	}

	return json.Decode(bytes.NewReader(input), v)
}
