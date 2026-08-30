package template

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Regression: unescapeTemplateQuotes ran unconditionally and stripped \" inside
// every {{...}}, corrupting a legitimate escaped quote in a string literal. It
// must now only apply as a fallback when the raw template fails to parse.
func TestRenderPreservesEscapedQuoteInLiteral(t *testing.T) {
	t.Parallel()

	engine := New(t.Context(), nil)

	// The string literal "a\"b" is valid template syntax and parses as-is to a"b.
	// The old unconditional unescape stripped the \" to "a"b" → parse error.
	out, err := engine.Render(`{{ printf "%s" "a\"b" }}`, Data{})
	require.NoError(t, err)
	require.Equal(t, `a"b`, out)
}

// IsTemplateString must require the closing delimiter to come AFTER the opening.
func TestIsTemplateStringDelimiterOrder(t *testing.T) {
	t.Parallel()

	require.True(t, IsTemplateString("Hello {{.Request.name}}"))
	require.False(t, IsTemplateString("a }} b {{ c"), "closing before opening is not a template")
	require.False(t, IsTemplateString("no delimiters"))
	require.False(t, IsTemplateString(""), "empty string")
	require.False(t, IsTemplateString("{{ unterminated"), "opening only")
}

// The unescape fallback only fires when the raw template fails to parse: a
// JSON-escaped template (\" reached the engine literally) must still render.
func TestRenderUnescapeFallback(t *testing.T) {
	t.Parallel()

	engine := New(t.Context(), nil)

	out, err := engine.Render(`{{ printf "%s" \"x\" }}`, Data{})
	require.NoError(t, err)
	require.Equal(t, "x", out)
}

// Regression: a global normalizeArgs wrapper spread a single array argument into
// positional args for EVERY builtin, so unary helpers like json/upper got only
// the first element. Array-spread now lives inside the math aggregates only.
func TestArrayArgHandling(t *testing.T) {
	t.Parallel()

	engine := New(t.Context(), nil)

	sum, err := engine.Render(`{{ sum .Request.nums }}`, Data{Request: map[string]any{"nums": []any{1, 2, 3}}})
	require.NoError(t, err)
	require.Equal(t, "6", sum, "aggregate must flatten a single array arg")

	arr, err := engine.Render(`{{ json .Request.items }}`, Data{Request: map[string]any{"items": []any{"a", "b", "c"}}})
	require.NoError(t, err)
	require.Equal(t, `["a","b","c"]`, arr, "unary helper must receive the whole array")
}

//nolint:funlen
func TestEngineRender(t *testing.T) {
	t.Parallel()

	engine := New(t.Context(), nil)

	tests := []struct {
		name     string
		template string
		data     Data
		expected string
		wantErr  bool
	}{
		{
			name:     "simple string",
			template: "Hello World",
			data:     Data{},
			expected: "Hello World",
			wantErr:  false,
		},
		{
			name:     "request data",
			template: "Hello {{.Request.name}}",
			data: Data{
				Request: map[string]any{
					"name": "John",
				},
			},
			expected: "Hello John",
			wantErr:  false,
		},
		{
			name:     "headers data",
			template: "Authorization: {{.Headers.authorization}}",
			data: Data{
				Headers: map[string]any{
					"authorization": "Bearer token123",
				},
			},
			expected: "Authorization: Bearer token123",
			wantErr:  false,
		},
		{
			name:     "message index",
			template: "Message {{.MessageIndex}}",
			data: Data{
				MessageIndex: 5,
			},
			expected: "Message 5",
			wantErr:  false,
		},
		{
			name:     "stub id",
			template: "Stub {{.StubID}}",
			data: Data{
				StubID: "test-stub-123",
			},
			expected: "Stub test-stub-123",
			wantErr:  false,
		},
		{
			name:     "extract field from requests",
			template: "{{sum (extract .Requests \"value\")}}",
			data: Data{
				Requests: []any{
					map[string]any{"value": 10.0},
					map[string]any{"value": 20.0},
					map[string]any{"value": 30.0},
				},
			},
			expected: "60",
			wantErr:  false,
		},
		{
			name:     "invalid template",
			template: "Hello {{.Request.name",
			data:     Data{},
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		// capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := engine.Render(tt.template, tt.data)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestIsTemplateString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "no template",
			input:    "Hello World",
			expected: false,
		},
		{
			name:     "has template",
			input:    "Hello {{.Request.name}}",
			expected: true,
		},
		{
			name:     "incomplete template",
			input:    "Hello {{.Request.name",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		// capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := IsTemplateString(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestEngineProcessHeaders(t *testing.T) {
	t.Parallel()

	engine := New(t.Context(), nil)

	headers := map[string]string{
		"x-user-id": "{{.Request.user_id}}",
		"x-role":    "{{.Headers.role}}",
		"static":    "static-value",
	}

	templateData := Data{
		Request: map[string]any{
			"user_id": "12345",
		},
		Headers: map[string]any{
			"role": "admin",
		},
	}

	err := engine.ProcessHeaders(headers, templateData)
	require.NoError(t, err)

	require.Equal(t, "12345", headers["x-user-id"])
	require.Equal(t, "admin", headers["x-role"])
	require.Equal(t, "static-value", headers["static"])
}

func TestEngineProcessError(t *testing.T) {
	t.Parallel()

	engine := New(t.Context(), nil)

	tests := []struct {
		name         string
		errorStr     string
		templateData Data
		expected     string
		wantErr      bool
	}{
		{
			name:     "no template",
			errorStr: "Simple error message",
			templateData: Data{
				Request: map[string]any{},
			},
			expected: "Simple error message",
			wantErr:  false,
		},
		{
			name:     "with template",
			errorStr: "Error for user {{.Request.user_id}}",
			templateData: Data{
				Request: map[string]any{
					"user_id": "12345",
				},
			},
			expected: "Error for user 12345",
			wantErr:  false,
		},
		{
			name:     "invalid template",
			errorStr: "Error {{.Request.user_id}}",
			templateData: Data{
				Request: map[string]any{},
			},
			expected: "Error <no value>",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		// capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := engine.ProcessError(tt.errorStr, tt.templateData)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestProcessMapMaxRecursionDepthExceeded(t *testing.T) {
	t.Parallel()

	engine := New(t.Context(), nil)

	// Build nested structure exceeding MaxRecursionDepth (250)
	data := make(map[string]any)
	current := data

	for range MaxRecursionDepth + 1 {
		nested := make(map[string]any)
		current["nested"] = nested
		current = nested
	}

	_, err := engine.ProcessValue(data, Data{})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrMaxRecursionDepthExceeded)
}

func TestProcessMapAtMaxRecursionDepth(t *testing.T) {
	t.Parallel()

	engine := New(t.Context(), nil)

	// Build nested structure at exactly MaxRecursionDepth - should succeed
	data := make(map[string]any)
	current := data

	for range MaxRecursionDepth {
		nested := make(map[string]any)
		current["k"] = nested
		current = nested
	}

	current["leaf"] = "value"

	_, err := engine.ProcessValue(data, Data{})
	require.NoError(t, err)
}

func TestHasTemplatesInHeaders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		headers  map[string]string
		expected bool
	}{
		{
			name: "no templates",
			headers: map[string]string{
				"content-type":  "application/json",
				"authorization": "Bearer token",
			},
			expected: false,
		},
		{
			name: "has templates",
			headers: map[string]string{
				"x-user-id":    "{{.Request.user_id}}",
				"content-type": "application/json",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		// capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := HasTemplatesInHeaders(tt.headers)
			require.Equal(t, tt.expected, result)
		})
	}
}
