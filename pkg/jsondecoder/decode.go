package jsondecoder

import (
	"bytes"

	"github.com/bytedance/sonic/decoder"
)

//nolint:mnd
func UnmarshalSlice(data []byte, v interface{}) error {
	input := bytes.TrimSpace(data)

	// input[0] == "{" AND input[len(input)-1] == "}"
	if bytes.HasPrefix(input, []byte{123}) &&
		bytes.HasSuffix(input, []byte{125}) {
		// "[${input}]"
		input = append(append([]byte{91}, input...), 93)
	}

	streamDecoder := decoder.NewStreamDecoder(bytes.NewReader(input))
	streamDecoder.UseNumber()

	return streamDecoder.Decode(v)
}
