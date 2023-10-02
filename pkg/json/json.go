package json

import (
	"bytes"

	fastjson "github.com/goccy/go-json"
)

type RawMessage fastjson.RawMessage

func Marshal(v any) ([]byte, error) {
	return fastjson.Marshal(v)
}

func Unmarshal(data []byte, v any) error {
	decoder := fastjson.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	if err := decoder.Decode(&v); err != nil {
		return err
	}

	return nil
}
