package sdk

import (
	"bytes"

	fastjson "encoding/json"
)

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
