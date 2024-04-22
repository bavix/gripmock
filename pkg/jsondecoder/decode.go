package jsondecoder

import (
	"bytes"
	"encoding/json"
)

func UnmarshalSlice(data []byte, v interface{}) error {
	input := bytes.TrimSpace(data)

	// input[0] == "{" AND input[len(input)-1] == "}"
	if bytes.HasPrefix(input, []byte{123}) &&
		bytes.HasSuffix(input, []byte{125}) {
		// "[${input}]"
		input = append(append([]byte{91}, input...), 93)
	}

	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber()

	return decoder.Decode(v)
}
