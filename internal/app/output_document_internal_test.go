package app

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
)

func documentEngine(t *testing.T) *template.Engine {
	t.Helper()

	return template.New(t.Context(), nil)
}

func documentTestData(t *testing.T) template.Data {
	t.Helper()

	request := map[string]any{
		"user_id": "USER_1",
		"count":   json.Number("2"),
		"items": []any{
			map[string]any{"id": "a", "qty": json.Number("1")},
			map[string]any{"id": "b", "qty": json.Number("0")},
		},
	}

	return newTemplateData(request, nil, 0, time.Now(), []any{request}, nil, 0)
}

func TestRenderDocumentBuildsUnaryPayload(t *testing.T) {
	t.Parallel()

	engine := documentEngine(t)

	output := stuber.Output{Template: true, Data: `{{ dict "items" (extract .Request.items "id") "user" .Request.user_id }}`}

	rendered, err := renderOutput(engine, output, documentTestData(t), renderOptions{})
	require.NoError(t, err)
	require.Empty(t, rendered.Template)

	data, ok := rendered.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "USER_1", data["user"])
	require.Equal(t, []any{"a", "b"}, data["items"])
}

func TestRenderDocumentBuildsStream(t *testing.T) {
	t.Parallel()

	engine := documentEngine(t)

	output := stuber.Output{Template: true, Stream: `{{ range $i := seq .Request.count }}{{ dict "seq" $i }}{{ end }}`}

	rendered, err := renderOutput(engine, output, documentTestData(t),
		renderOptions{skipData: true})
	require.NoError(t, err)
	require.Len(t, rendered.Messages(), 2)
	require.Equal(t, map[string]any{"seq": json.Number("0")}, rendered.Messages()[0])
}

func TestRenderDocumentStreamAcceptsOneArray(t *testing.T) {
	t.Parallel()

	engine := documentEngine(t)

	output := stuber.Output{Template: true, Stream: `{{ .Request.items | where "qty" "gt" 0 }}`}

	rendered, err := renderOutput(engine, output, documentTestData(t),
		renderOptions{skipData: true})
	require.NoError(t, err)
	require.Len(t, rendered.Messages(), 1)
	require.Equal(t, map[string]any{"id": "a", "qty": json.Number("1")}, rendered.Messages()[0])
}

func TestRenderDocumentStreamCanBeEmpty(t *testing.T) {
	t.Parallel()

	engine := documentEngine(t)

	output := stuber.Output{Template: true, Stream: `{{ range $i := seq 0 }}{{ dict "seq" $i }}{{ end }}`}

	rendered, err := renderOutput(engine, output, documentTestData(t),
		renderOptions{skipData: true})
	require.NoError(t, err)
	require.Empty(t, rendered.Messages())
}

func TestRenderDocumentDropsLiteralSiblings(t *testing.T) {
	t.Parallel()

	engine := documentEngine(t)

	dataOutput := stuber.Output{Template: true, Data: `{{ dict "seq" 1 }}`}

	unary, err := renderOutput(engine, dataOutput, documentTestData(t), renderOptions{})
	require.NoError(t, err)
	require.Nil(t, unary.Stream)
	require.Equal(t, map[string]any{"seq": json.Number("1")}, unary.Data)

	streamOutput := stuber.Output{Template: true, Stream: `{{ dict "seq" 1 }}`}

	streamed, err := renderOutput(engine, streamOutput, documentTestData(t),
		renderOptions{skipData: true})
	require.NoError(t, err)
	require.Nil(t, streamed.Data)
	require.Equal(t, []any{map[string]any{"seq": json.Number("1")}}, streamed.Messages())
}

func TestAppendDoesNotMutateRequestData(t *testing.T) {
	t.Parallel()

	engine := documentEngine(t)

	request := map[string]any{"items": []any{
		map[string]any{"id": "a"},
		map[string]any{"id": "b"},
		map[string]any{"id": "c"},
	}}
	templateData := newTemplateData(request, nil, 0, time.Now(), []any{request}, nil, 0)

	output := stuber.Output{Template: true, Data: `{{ $page := page 0 2 .Request.items }}` +
		`{{ $grown := append $page (dict "id" "z") }}` +
		`{{ $direct := append .Request.items (dict "id" "y") }}` +
		`{{ dict "grown" (len $grown) "direct" (len $direct) "items" .Request.items }}`}

	rendered, err := renderOutput(engine, output, templateData, renderOptions{})
	require.NoError(t, err)

	data, ok := rendered.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, json.Number("3"), data["grown"])
	require.Equal(t, json.Number("4"), data["direct"])
	require.Equal(t, []any{
		map[string]any{"id": "a"},
		map[string]any{"id": "b"},
		map[string]any{"id": "c"},
	}, data["items"])
}

func TestRenderDocumentKeepsNumbersExact(t *testing.T) {
	t.Parallel()

	engine := documentEngine(t)

	output := stuber.Output{Template: true, Data: `{{ dict "id" 9223372036854775807 }}`}

	rendered, err := renderOutput(engine, output, documentTestData(t), renderOptions{})
	require.NoError(t, err)

	data, ok := rendered.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, json.Number("9223372036854775807"), data["id"])
}

func TestRenderDocumentDoesNotRenderRequestData(t *testing.T) {
	t.Parallel()

	engine := documentEngine(t)

	request := map[string]any{"payload": "{{ .Request.payload }}"}
	templateData := newTemplateData(request, nil, 0, time.Now(), []any{request}, nil, 0)

	output := stuber.Output{Template: true, Data: `{{ dict "echo" .Request.payload }}`}

	rendered, err := renderOutput(engine, output, templateData, renderOptions{})
	require.NoError(t, err)

	data, ok := rendered.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "{{ .Request.payload }}", data["echo"])
}

func TestRenderDocumentErrors(t *testing.T) {
	t.Parallel()

	engine := documentEngine(t)

	tests := map[string]struct {
		output   stuber.Output
		opts     renderOptions
		contains string
	}{
		"invalid json": {
			output:   stuber.Output{Template: true, Data: `{"items": [ }`},
			contains: "did not render valid JSON",
		},
		"several values for unary": {
			output:   stuber.Output{Template: true, Data: `{"a": 1} {"b": 2}`},
			contains: "must render exactly one JSON value",
		},
		"nothing for unary": {
			output:   stuber.Output{Template: true, Data: `{{ range $i := seq 0 }}{{ dict "seq" $i }}{{ end }}`},
			contains: "must render exactly one JSON value",
		},
		"gripmock key in a unary document": {
			output:   stuber.Output{Template: true, Data: `{{ dict "_gripmock" (dict "delay" "5ms") "id" 1 }}`},
			contains: "_gripmock is only allowed in stream messages",
		},
		"nested loops beyond the byte cap": {
			output: stuber.Output{Template: true, Stream: `{{ range $a := seq 10000 }}{{ range $b := seq 10000 }}` +
				`xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx{{ end }}{{ end }}`},
			opts:     renderOptions{skipData: true},
			contains: "rendered too much text",
		},
		"seq beyond the cap": {
			output:   stuber.Output{Template: true, Stream: `{{ range $i := seq 20000 }}{{ dict "seq" $i }}{{ end }}`},
			opts:     renderOptions{skipData: true},
			contains: "seq supports at most 10000 indexes",
		},
		"template failure": {
			output:   stuber.Output{Template: true, Data: `{{ dict "a" }}`},
			contains: "failed to render output template",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := renderOutput(engine, tt.output, documentTestData(t), tt.opts)
			require.ErrorContains(t, err, tt.contains)
		})
	}
}

func TestRenderDocumentKeepsScalarPayload(t *testing.T) {
	t.Parallel()

	engine := documentEngine(t)

	output := stuber.Output{Template: true, Data: `{{ toJson "2024-01-01T12:00:00Z" }}`}

	rendered, err := renderOutput(engine, output, documentTestData(t), renderOptions{})
	require.NoError(t, err)
	require.Equal(t, "2024-01-01T12:00:00Z", rendered.Data)
}

func TestHealthStubRejectsTemplate(t *testing.T) {
	t.Parallel()

	validate, err := NewStubValidator()
	require.NoError(t, err)

	engine := documentEngine(t)

	healthStub := &stuber.Stub{
		Service: HealthServiceFullName,
		Method:  "Check",
		Input:   stuber.InputData{Equals: map[string]any{"service": ""}},
		Output:  stuber.Output{Template: true, Data: `{{ dict "status" "SERVING" }}`},
	}
	require.ErrorContains(t, checkStub(validate, engine, healthStub), errHealthTemplate)

	regularStub := &stuber.Stub{
		Service: "catalog.CatalogService",
		Method:  "Search",
		Input:   stuber.InputData{Equals: map[string]any{"id": "1"}},
		Output:  stuber.Output{Template: true, Data: `{{ dict "matched" 1 }}`},
	}
	require.NoError(t, checkStub(validate, engine, regularStub))
}

func TestSingleMessageIgnoresStreamOutput(t *testing.T) {
	t.Parallel()

	unary := stuber.Output{Data: map[string]any{"ok": true}}
	require.Equal(t, map[string]any{"ok": true}, singleMessage(unary))

	streamed := stuber.Output{Stream: []any{map[string]any{"ok": true}}}
	require.Equal(t, map[string]any{"ok": true}, singleMessage(streamed))

	require.Nil(t, singleMessage(stuber.Output{}))
}

func TestOutputTemplateValidation(t *testing.T) {
	t.Parallel()

	require.True(t, isValidOutputConfiguration(stuber.Output{Template: true, Data: "{}"}))
	require.False(t, isValidOutputConfiguration(stuber.Output{Template: true, Data: map[string]any{"a": 1}}))
	require.False(t, isValidOutputConfiguration(stuber.Output{Template: true, Data: "{}", Stream: []any{"a"}}))
	require.True(t, isValidOutputConfiguration(stuber.Output{Data: map[string]any{"a": 1}}))
}
