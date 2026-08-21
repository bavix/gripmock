package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func renderTestData() map[string]any {
	return map[string]any{
		"user_id": "USER_1",
		"count":   3,
	}
}

func TestRenderOutputRendersLeaves(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	request := renderTestData()
	templateData := newTemplateData(request, map[string]any{"x-role": "admin"}, 0, time.Now(), []any{request}, nil, 0)

	output := stuber.Output{
		Data: map[string]any{
			"user":   "{{ .Request.user_id }}",
			"nested": map[string]any{"role": `{{ index .Headers "x-role" }}`},
			"list":   []any{"{{ .Request.user_id }}"},
		},
		Headers: map[string]string{"x-user": "{{ .Request.user_id }}"},
		Error:   "boom {{ .Request.user_id }}",
	}

	rendered, err := renderOutput(mocker.templateEngine, output, templateData, renderOptions{})
	require.NoError(t, err)

	data, ok := rendered.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "USER_1", data["user"])
	require.Equal(t, map[string]any{"role": "admin"}, data["nested"])
	require.Equal(t, []any{"USER_1"}, data["list"])
	require.Equal(t, map[string]string{"x-user": "USER_1"}, rendered.Headers)
	require.Equal(t, "boom USER_1", rendered.Error)
}

func TestRenderOutputKeepsLeafValuesAsText(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	request := renderTestData()
	templateData := newTemplateData(request, nil, 0, time.Now(), []any{request}, nil, 0)

	output := stuber.Output{Data: map[string]any{"text": "{{ .Request.count }}"}}

	rendered, err := renderOutput(mocker.templateEngine, output, templateData, renderOptions{})
	require.NoError(t, err)

	data, ok := rendered.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "3", data["text"])
}

func TestRenderOutputSkipsDataWhenAsked(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	templateData := newTemplateData(renderTestData(), nil, 0, time.Now(), nil, nil, 0)

	output := stuber.Output{Data: map[string]any{"user": "{{ .Request.user_id }}"}}

	rendered, err := renderOutput(mocker.templateEngine, output, templateData, renderOptions{skipData: true})
	require.NoError(t, err)
	require.Equal(t, output.Data, rendered.Data)
}

func TestRenderOutputRendersStreamOnDemand(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	request := renderTestData()
	templateData := newTemplateData(request, nil, 0, time.Now(), []any{request}, nil, 0)

	output := stuber.Output{Stream: []any{
		map[string]any{"id": "{{ .Request.user_id }}"},
		"static",
	}}

	rendered, err := renderOutput(mocker.templateEngine, output, templateData, renderOptions{renderStream: true})
	require.NoError(t, err)
	require.Len(t, rendered.Messages(), 2)

	first, ok := rendered.Messages()[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "USER_1", first["id"])
	require.Equal(t, "static", rendered.Messages()[1])
}

func TestRenderOutputLeavesStaticOutputUntouched(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	templateData := newTemplateData(nil, nil, 0, time.Now(), nil, nil, 0)

	data := map[string]any{"id": "static"}
	output := stuber.Output{Data: data, Headers: map[string]string{"x": "y"}}

	rendered, err := renderOutput(mocker.templateEngine, output, templateData, renderOptions{})
	require.NoError(t, err)
	require.Equal(t, data, rendered.Data)
	require.Equal(t, map[string]string{"x": "y"}, rendered.Headers)
}
