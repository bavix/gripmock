package yaml2json

import (
	"bytes"
	"sync"
	"text/template"

	"github.com/bavix/gripmock/v3/internal/infra/encoding"
)

type engine struct {
	funcs      template.FuncMap
	bufferPool *sync.Pool
}

func newEngine() *engine {
	utils := encoding.NewTemplateUtils()

	return &engine{
		funcs: template.FuncMap{
			"bytes":         utils.Conversion.StringToBytes,
			"string2base64": utils.Base64.StringToBase64,
			"bytes2base64":  utils.Base64.BytesToBase64,
			"uuid2base64":   utils.UUID.UUIDToBase64,
			"uuid2bytes":    utils.UUID.UUIDToBytes,
			"uuid2int64":    utils.UUID.UUIDToInt64,
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
