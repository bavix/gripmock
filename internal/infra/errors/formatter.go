package errors

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"text/template"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

//go:embed error.tmpl
var errorTemplate string

//nolint:gochecknoglobals
var tmpl = template.Must(
	template.New("stub_not_found").
		Funcs(templateFuncs).
		Parse(normalizeLineEndings(errorTemplate)))

var templateFuncs = template.FuncMap{ //nolint:gochecknoglobals
	"toJSON": func(v any) string {
		sanitized, ok := sanitizeValue(v)
		if !ok {
			return "{}"
		}

		b, err := json.MarshalIndent(sanitized, "", "\t")
		if err != nil {
			return "{}"
		}

		return string(b)
	},
	"hasMap": func(m map[string]any) bool {
		return len(filterFuncs(m)) > 0
	},
	"hasInputData": func(d stuber.InputData) bool {
		return len(filterFuncs(d.Equals)) > 0 ||
			len(filterFuncs(d.Contains)) > 0 ||
			len(filterFuncs(d.Matches)) > 0 ||
			len(d.AnyOf) > 0
	},
	"hasHeaderData": func(h stuber.InputHeader) bool {
		return h.Len() > 0
	},
	"firstAnyOfInput": func(anyOf []stuber.AnyOfElement) *stuber.AnyOfElement {
		if len(anyOf) == 0 {
			return nil
		}

		return &anyOf[0]
	},
	"firstAnyOfHeader": func(anyOf []stuber.AnyOfHeaderElement) *stuber.AnyOfHeaderElement {
		if len(anyOf) == 0 {
			return nil
		}

		return &anyOf[0]
	},
	"hiddenAnyOf": func(total int) int {
		if total <= 1 {
			return 0
		}

		return total - 1
	},
	"dict": func(values ...any) map[string]any {
		result := make(map[string]any)

		for i := 0; i+1 < len(values); i += 2 {
			key, ok := values[i].(string)
			if !ok {
				continue
			}

			result[key] = values[i+1]
		}

		return result
	},
}

// Result interface for testing - allows mocking stuber.Result.
type Result interface {
	Found() *stuber.Stub
	Similar() *stuber.Stub
}

// StubNotFoundFormatter provides unified formatting for "stub not found" errors.
type StubNotFoundFormatter struct{}

// NewStubNotFoundFormatter creates a new formatter instance.
func NewStubNotFoundFormatter() *StubNotFoundFormatter {
	return &StubNotFoundFormatter{}
}

// Format formats error messages for stub not found scenarios.
// Supports both unary (single Input element) and streaming (multiple Input elements).
func (f *StubNotFoundFormatter) Format(expect stuber.Query, result Result) error {
	blocks := make([]string, 0, 5) //nolint:mnd

	// Block 1: title
	blocks = append(blocks, "No matching stub found")

	// Block 2: service + method
	blocks = append(blocks, fmt.Sprintf("Service: %s\nMethod: %s", expect.Service, expect.Method))

	// Block 3 (optional): request headers
	if len(filterFuncs(expect.Headers)) > 0 {
		b, err := json.MarshalIndent(filterFuncs(expect.Headers), "", "\t")
		if err != nil {
			b = []byte("{}")
		}

		blocks = append(blocks, "Request headers:\n"+string(b))
	}

	// Block 4: request input
	blocks = append(blocks, buildInputBlock(expect.Input))

	// Block 5: similar stub or "not found"
	similar := result.Similar()

	if similar == nil {
		blocks = append(blocks, "No similar stubs found.")
	} else {
		var buf bytes.Buffer

		_ = tmpl.Execute(&buf, similar)
		blocks = append(blocks, strings.TrimRight(buf.String(), "\n"))
	}

	return fmt.Errorf("%s", strings.Join(blocks, "\n\n")) //nolint:err113
}

func buildInputBlock(input []map[string]any) string {
	if len(input) == 0 {
		return "Request input:\n(empty)"
	}

	if len(input) == 1 {
		b, err := json.MarshalIndent(filterFuncs(input[0]), "", "\t")
		if err != nil {
			b = []byte("{}")
		}

		return "Request input:\n" + string(b)
	}

	var sb strings.Builder

	sb.WriteString("Request input (stream):")

	for i, item := range input {
		b, err := json.MarshalIndent(filterFuncs(item), "", "\t")
		if err != nil {
			b = []byte("{}")
		}

		fmt.Fprintf(&sb, "\n[%d]\n%s", i, string(b))
	}

	return sb.String()
}

func filterFuncs(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}

	r := make(map[string]any, len(m))
	for k, v := range m {
		sanitized, ok := sanitizeValue(v)
		if ok {
			r[k] = sanitized
		}
	}

	return r
}

func sanitizeValue(v any) (any, bool) {
	if v == nil {
		return nil, true
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil, true
	}

	return sanitizeReflectValue(rv)
}

func sanitizeReflectValue(rv reflect.Value) (any, bool) {
	switch rv.Kind() {
	case reflect.Func, reflect.Chan, reflect.UnsafePointer, reflect.Complex64, reflect.Complex128:
		return nil, false
	case reflect.Map:
		return sanitizeMap(rv)
	case reflect.Slice, reflect.Array:
		return sanitizeSlice(rv)
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Interface, reflect.Pointer,
		reflect.String, reflect.Struct, reflect.Invalid:
		return sanitizePrimitive(rv)
	default:
		return sanitizePrimitive(rv)
	}
}

func sanitizeMap(rv reflect.Value) (any, bool) {
	if rv.Type().Key().Kind() != reflect.String {
		return nil, false
	}

	result := make(map[string]any, rv.Len())

	for _, key := range rv.MapKeys() {
		value, ok := sanitizeValue(rv.MapIndex(key).Interface())
		if ok {
			result[key.String()] = value
		}
	}

	return result, true
}

func sanitizeSlice(rv reflect.Value) (any, bool) {
	result := make([]any, 0, rv.Len())

	for i := range rv.Len() {
		value, ok := sanitizeValue(rv.Index(i).Interface())
		if ok {
			result = append(result, value)
		}
	}

	return result, true
}

func sanitizePrimitive(rv reflect.Value) (any, bool) {
	if _, err := json.Marshal(rv.Interface()); err != nil {
		return nil, false
	}

	return rv.Interface(), true
}

func normalizeLineEndings(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}
