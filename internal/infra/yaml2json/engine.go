package yaml2json

import (
	"bytes"
	"encoding/base64"
	"strconv"
	"strings"
	"sync"
	"text/template"

	"github.com/google/uuid"

	"github.com/bavix/apis/pkg/uuidconv"
)

type engine struct {
	funcs      template.FuncMap
	bufferPool *sync.Pool
}

func newEngine() *engine {
	return &engine{
		funcs: template.FuncMap{
			"bytes":         convBytes,
			"string2base64": string2base64,
			"bytes2base64":  bytes2base64,
			"uuid2base64":   uuid2base64,
			"uuid2bytes":    uuid2bytes,
			"uuid2int64":    uuid2int64,
		},
		bufferPool: &sync.Pool{
			New: func() any {
				return new(bytes.Buffer)
			},
		},
	}
}

func (e *engine) Execute(name string, data []byte) ([]byte, error) {
	t := template.New(name).Funcs(e.funcs)

	t, err := t.Parse(string(data))
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	buf, _ := e.bufferPool.Get().(*bytes.Buffer)

	defer func() {
		buf.Reset()
		e.bufferPool.Put(buf)
	}()

	err = t.Execute(buf, nil)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	return buf.Bytes(), nil
}

func uuid2int64(str string) string {
	v := uuid.MustParse(str)

	high, low := uuidconv.UUID2DoubleInt(v)

	var sb strings.Builder

	sb.Grow(32) //nolint:mnd
	sb.WriteString(`{"high":`)
	sb.WriteString(strconv.FormatInt(high, 10))
	sb.WriteString(`,"low":`)
	sb.WriteString(strconv.FormatInt(low, 10))
	sb.WriteString(`}`)

	return sb.String()
}

func uuid2base64(input string) string {
	v := uuid.MustParse(input)

	return base64.StdEncoding.EncodeToString(v[:])
}

// uuid2bytes converts a UUID string to a byte slice.
func uuid2bytes(input string) []byte {
	v := uuid.MustParse(input)

	return v[:]
}

func convBytes(v string) []byte {
	return []byte(v)
}

func string2base64(v string) string {
	return base64.StdEncoding.EncodeToString([]byte(v))
}

func bytes2base64(v []byte) string {
	return base64.StdEncoding.EncodeToString(v)
}
