package app

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	infraTemplate "github.com/bavix/gripmock/v3/internal/infra/template"
)

func TestRenderOutputDataStructuralTemplate(t *testing.T) {
	t.Parallel()

	engine := infraTemplate.New(t.Context(), nil)
	output := stuber.Output{DataTemplate: `items:
{{ range .Request.items }}
  - id: {{ .id }}
{{ else }}
  []
{{ end }}`}

	value, err := renderOutputData(engine, output, infraTemplate.Data{Request: map[string]any{
		"items": []any{map[string]any{"id": 1}, map[string]any{"id": 2}},
	}})
	require.NoError(t, err)

	root, ok := value.(map[string]any)
	require.True(t, ok)
	items, ok := root["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 2)
	require.Equal(t, json.Number("1"), items[0].(map[string]any)["id"]) //nolint:forcetypeassert
}

func TestRenderOutputStreamTemplate(t *testing.T) {
	t.Parallel()

	engine := infraTemplate.New(t.Context(), nil)
	output := stuber.Output{StreamTemplate: `
{{ range .Request.items }}
- id: {{ .id }}
{{ else }}
[]
{{ end }}`}

	stream, rendered, err := renderOutputStreamTemplate(engine, output, infraTemplate.Data{Request: map[string]any{
		"items": []any{map[string]any{"id": 1}, map[string]any{"id": 2}, map[string]any{"id": 3}},
	}})
	require.NoError(t, err)
	require.True(t, rendered)
	require.Len(t, stream, 3)
}

func TestRenderOutputStreamTemplateRequiresArray(t *testing.T) {
	t.Parallel()

	engine := infraTemplate.New(t.Context(), nil)
	stream, rendered, err := renderOutputStreamTemplate(engine, stuber.Output{StreamTemplate: "value: 1"}, infraTemplate.Data{})
	require.Error(t, err)
	require.True(t, rendered)
	require.Nil(t, stream)
}
