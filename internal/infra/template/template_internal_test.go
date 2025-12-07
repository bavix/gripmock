package template

import (
	"testing"

	"github.com/stretchr/testify/require"
)

//nolint:funlen
func TestEngine_Render(t *testing.T) {
	t.Parallel()

	engine := New(nil)

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

func TestEngine_ProcessMap(t *testing.T) {
	t.Parallel()

	engine := New(nil)

	data := map[string]any{
		"name": "{{.Request.name}}",
		"age":  "{{.Request.age}}",
		"nested": map[string]any{
			"title": "{{.Request.title}}",
		},
		"array": []any{
			"{{.Request.item1}}",
			"{{.Request.item2}}",
		},
	}

	templateData := Data{
		Request: map[string]any{
			"name":  "John",
			"age":   "30",
			"title": "Mr",
			"item1": "first",
			"item2": "second",
		},
	}

	err := engine.ProcessMap(data, templateData)
	require.NoError(t, err)

	require.Equal(t, "John", data["name"])
	require.Equal(t, "30", data["age"])
	nested, ok := data["nested"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "Mr", nested["title"])

	array, ok := data["array"].([]any)
	require.True(t, ok)
	require.Equal(t, "first", array[0])
	require.Equal(t, "second", array[1])
}

func TestEngine_ProcessHeaders(t *testing.T) {
	t.Parallel()

	engine := New(nil)

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

func TestEngine_ProcessError(t *testing.T) {
	t.Parallel()

	engine := New(nil)

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

func TestHasTemplates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     map[string]any
		expected bool
	}{
		{
			name: "no templates",
			data: map[string]any{
				"name": "John",
				"age":  30,
			},
			expected: false,
		},
		{
			name: "has templates",
			data: map[string]any{
				"name": "{{.Request.name}}",
				"age":  30,
			},
			expected: true,
		},
		{
			name: "nested templates",
			data: map[string]any{
				"user": map[string]any{
					"name": "{{.Request.name}}",
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		// capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := HasTemplates(tt.data)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestHasTemplatesInStream(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		stream   []any
		expected bool
	}{
		{
			name: "no templates",
			stream: []any{
				map[string]any{"name": "John"},
				map[string]any{"age": 30},
			},
			expected: false,
		},
		{
			name: "has templates",
			stream: []any{
				map[string]any{"name": "{{.Request.name}}"},
				map[string]any{"age": 30},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		// capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := HasTemplatesInStream(tt.stream)
			require.Equal(t, tt.expected, result)
		})
	}
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
