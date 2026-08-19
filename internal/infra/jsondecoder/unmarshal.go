package jsondecoder

import (
	"bytes"
	"slices"

	"github.com/goccy/go-json"
)

const minJSONLength = 2

// UnmarshalSlice is a function that parses JSON data into a slice of the provided interface.
func UnmarshalSlice(data []byte, v any) error {
	input := bytes.TrimSpace(data)

	if len(input) < minJSONLength {
		return &json.SyntaxError{}
	}

	if len(input) > 0 && input[0] == '{' && input[len(input)-1] == '}' {
		input = slices.Concat([]byte{'['}, input, []byte{']'})
	}

	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber()

	return decoder.Decode(v) //nolint:wrapcheck
}

func Unmarshal(data []byte, v any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	return decoder.Decode(v) //nolint:wrapcheck
}
