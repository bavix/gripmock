package yaml2json

import (
	"bytes"
	"encoding/base64"
	"text/template"

	"github.com/goccy/go-json"
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

func (e *engine) uuid2int64(str string) string {
	v := e.uuid2bytes(str)

	//nolint:gomnd
	high := int64(v[0]) | int64(v[1])<<8 | int64(v[2])<<16 | int64(v[3])<<24 |
		int64(v[4])<<32 | int64(v[5])<<40 | int64(v[6])<<48 | int64(v[7])<<56

	//nolint:gomnd
	low := int64(v[8]) | int64(v[9])<<8 | int64(v[10])<<16 | int64(v[11])<<24 |
		int64(v[12])<<32 | int64(v[13])<<40 | int64(v[14])<<48 | int64(v[15])<<56

	var buffer bytes.Buffer

	//nolint:errchkjson
	_ = json.NewEncoder(&buffer).Encode(map[string]int64{
		"high": high,
		"low":  low,
	})

	return buffer.String()
}

func (e *engine) uuid2base64(input string) string {
	return e.bytes2base64(e.uuid2bytes(input))
}

func (e *engine) uuid2bytes(input string) []byte {
	v := uuid.MustParse(input)

	return v[:]
}

func (e *engine) bytes(v string) []byte {
	return []byte(v)
}

func (e *engine) string2base64(v string) string {
	return base64.StdEncoding.EncodeToString(e.bytes(v))
}

func (e *engine) bytes2base64(v []byte) string {
	return base64.StdEncoding.EncodeToString(v)
}

func (e *engine) funcMap() template.FuncMap {
	return template.FuncMap{
		"bytes":         e.bytes,
		"string2base64": e.string2base64,
		"bytes2base64":  e.bytes2base64,
		"uuid2base64":   e.uuid2base64,
		"uuid2bytes":    e.uuid2bytes,
		"uuid2int64":    e.uuid2int64,
	}
}
