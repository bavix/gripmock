package yaml2json

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"text/template"

	"github.com/google/uuid"
)

type engine struct{}

func (e *engine) Execute(name string, data []byte) ([]byte, error) {
	var buffer bytes.Buffer

	parse, err := template.New(name).Funcs(e.funcMap()).Parse(string(data))
	if err != nil {
		return nil, err
	}

	if err := parse.Execute(&buffer, nil); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (e *engine) funcMap() template.FuncMap {
	return template.FuncMap{
		"uuidToBase64StdEncoding": func(guid string) string {
			v := uuid.MustParse(guid)

			return base64.StdEncoding.EncodeToString(v[:])
		},
		"uuidToHighLowLittleEndian": func(guid string) string {
			v := uuid.MustParse(guid)

			high := int64(v[0]) | int64(v[1])<<8 | int64(v[2])<<16 | int64(v[3])<<24 |
				int64(v[4])<<32 | int64(v[5])<<40 | int64(v[6])<<48 | int64(v[7])<<56

			low := int64(v[8]) | int64(v[9])<<8 | int64(v[10])<<16 | int64(v[11])<<24 |
				int64(v[12])<<32 | int64(v[13])<<40 | int64(v[14])<<48 | int64(v[15])<<56

			var buffer bytes.Buffer

			err := json.NewEncoder(&buffer).Encode(map[string]int64{
				"high": high,
				"low":  low,
			})

			if err != nil {
				return guid
			}

			return buffer.String()
		},
	}
}
