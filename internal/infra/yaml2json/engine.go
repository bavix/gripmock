package yaml2json

import (
	"bytes"
	"strings"
	"sync"
	"text/template"

	templatepkg "github.com/bavix/gripmock/v3/internal/infra/template"
	"github.com/bavix/gripmock/v3/pkg/plugins"
)

type engine struct {
	funcs      template.FuncMap
	bufferPool *sync.Pool
}

func newEngine(reg plugins.Registry) *engine {
	templateFuncs := templatepkg.Functions(reg)

	return &engine{
		funcs: templateFuncs,
		bufferPool: &sync.Pool{
			New: func() any {
				return new(bytes.Buffer)
			},
		},
	}
}

// isDynamicTemplate checks if a string contains dynamic template syntax.
func isDynamicTemplate(s string) bool {
	// Dynamic templates reference runtime data like Request/Headers/MessageIndex/Requests/State.
	if !strings.Contains(s, "{{") || !strings.Contains(s, "}}") {
		return false
	}

	markers := []string{
		".Request",      // request payload
		".Headers",      // request headers
		".MessageIndex", // stream message index
		".Requests",     // all client messages
		".State",        // request state
	}

	for _, m := range markers {
		if strings.Contains(s, m) {
			return true
		}
	}

	return false
}

func (e *engine) Execute(name string, data []byte) ([]byte, error) {
	// Check if the data contains dynamic templates
	if isDynamicTemplate(string(data)) {
		// For dynamic templates, we need to escape them so they don't get processed statically
		// We'll replace {{ with {{`{{`}} and }} with {{`}}`}} to escape them
		escapedData := escapeDynamicTemplates(string(data))

		return []byte(escapedData), nil
	}

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

// escapeDynamicTemplates escapes dynamic template syntax so it doesn't get processed statically.
func escapeDynamicTemplates(data string) string {
	// This is a simple approach - we'll just return the data as-is
	// The dynamic processing will happen at runtime in the gRPC server
	return data
}
