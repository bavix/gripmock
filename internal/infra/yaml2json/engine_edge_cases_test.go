package yaml2json_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/yaml2json"
)

func TestRuntimeTemplateEscaping(t *testing.T) {
	t.Parallel()

	convertor := yaml2json.New(nil)

	tests := []struct {
		name     string
		input    string
		expected string // JSON expected output
	}{
		{
			name: "Request template escaped",
			input: `
input:
  equals:
    field: "{{.Request.field}}"
`,
			expected: `{"input": {"equals": {"field": "{{.Request.field}}"}}}`,
		},
		{
			name: "Headers template escaped",
			input: `
input:
  equals:
    auth: "{{.Headers.Authorization}}"
`,
			expected: `{"input": {"equals": {"auth": "{{.Headers.Authorization}}"}}}`,
		},
		{
			name: "State template escaped",
			input: `
input:
  equals:
    counter: "{{.State.counter}}"
`,
			expected: `{"input": {"equals": {"counter": "{{.State.counter}}"}}}`,
		},
		{
			name: "Mixed runtime and static - runtime escaped",
			input: `
input:
  equals:
    field: "{{.Request.field}}"
    static: {{ uuid2base64 "77465064-a0ce-48a3-b7e4-d50f88e55093" }}
`,
			expected: `{"input": {"equals": {"field": "{{.Request.field}}", ` +
				`"static": "{{ uuid2base64 \"77465064-a0ce-48a3-b7e4-d50f88e55093\" }}"}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := convertor.Execute(t.Context(), "test", []byte(tt.input))
			require.NoError(t, err)
			require.JSONEq(t, tt.expected, string(result))
		})
	}
}

func TestQuoteHandling(t *testing.T) {
	t.Parallel()

	convertor := yaml2json.New(nil)

	tests := []struct {
		name     string
		input    string
		expected string // JSON expected output
	}{
		{
			name: "Double quoted runtime template",
			input: `
field: "{{.Request.value}}"
`,
			expected: `{"field": "{{.Request.value}}"}`,
		},
		{
			name: "Single quoted runtime template",
			input: `
field: '{{.Request.value}}'
`,
			expected: `{"field": "{{.Request.value}}"}`,
		},
		{
			name: "Unquoted runtime template after colon",
			input: `
field: {{.Request.value}}
`,
			expected: `{"field": "{{.Request.value}}"}`,
		},
		{
			name: "Unquoted runtime template in list",
			input: `
list:
  - {{.Request.value}}
`,
			expected: `{"list": ["{{.Request.value}}"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := convertor.Execute(t.Context(), "test", []byte(tt.input))
			require.NoError(t, err)
			require.JSONEq(t, tt.expected, string(result))
		})
	}
}

func TestNoTemplateFunctions(t *testing.T) {
	t.Parallel()

	convertor := yaml2json.New(nil)

	input := `
input:
  equals:
    field: value
`
	result, err := convertor.Execute(t.Context(), "test", []byte(input))
	require.NoError(t, err)

	// YAML should be converted to JSON
	require.JSONEq(t, `{"input": {"equals": {"field": "value"}}}`, string(result))
}

func TestEmptyInput(t *testing.T) {
	t.Parallel()

	convertor := yaml2json.New(nil)

	result, err := convertor.Execute(t.Context(), "test", []byte{})
	require.NoError(t, err)
	// Empty YAML converts to null JSON
	require.Contains(t, string(result), "null")
}

func TestWhitespaceOnly(t *testing.T) {
	t.Parallel()

	convertor := yaml2json.New(nil)

	input := "   \n  \n  "
	result, err := convertor.Execute(t.Context(), "test", []byte(input))
	require.NoError(t, err)

	// Whitespace-only YAML converts to empty/null JSON
	require.Contains(t, string(result), "null")
}
