package app

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
)

func benchRequest(size int) map[string]any {
	items := make([]any, 0, size)
	for i := range size {
		items = append(items, map[string]any{
			"id":  "item-" + json.Number(string(rune('a'+i%26))).String(),
			"qty": json.Number("2"),
		})
	}

	return map[string]any{"user_id": "USER_1", "count": json.Number("3"), "items": items}
}

func benchData(size int) map[string]any {
	items := make([]any, 0, size)
	for range size {
		items = append(items, map[string]any{
			"id":  "{{ .Request.user_id }}",
			"qty": "{{ .Request.count }}",
		})
	}

	return map[string]any{"user": "{{ .Request.user_id }}", "items": items}
}

const benchDocument = `{{ dict "user" .Request.user_id "items" (.Request.items | where "qty" "gt" 0) }}`

const benchStreamDocument = `{{ range $item := .Request.items }}{{ dict "id" $item.id "qty" $item.qty }}{{ end }}`

func benchmarkRender(b *testing.B, output stuber.Output, size int, opts renderOptions) {
	b.Helper()

	engine := template.New(b.Context(), nil)
	request := benchRequest(size)
	templateData := newTemplateData(request, nil, 0, time.Now(), []any{request}, nil, 0)

	b.ReportAllocs()

	for b.Loop() {
		if _, err := renderOutput(engine, output, templateData, opts); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderLiteralLeaves(b *testing.B) {
	for _, size := range []int{1, 10, 100} {
		b.Run(sizeName(size), func(b *testing.B) {
			benchmarkRender(b, stuber.Output{Data: benchData(size)}, size, renderOptions{})
		})
	}
}

func BenchmarkRenderDocument(b *testing.B) {
	for _, size := range []int{1, 10, 100} {
		b.Run(sizeName(size), func(b *testing.B) {
			benchmarkRender(b, stuber.Output{Template: true, Data: benchDocument}, size, renderOptions{})
		})
	}
}

func BenchmarkRenderDocumentStream(b *testing.B) {
	for _, size := range []int{1, 10, 100} {
		b.Run(sizeName(size), func(b *testing.B) {
			benchmarkRender(b, stuber.Output{Template: true, Stream: benchStreamDocument}, size,
				renderOptions{skipData: true})
		})
	}
}

func BenchmarkRenderStaticOutput(b *testing.B) {
	for _, size := range []int{1, 10, 100} {
		b.Run(sizeName(size), func(b *testing.B) {
			items := make([]any, 0, size)
			for range size {
				items = append(items, map[string]any{"id": "static", "qty": json.Number("1")})
			}

			benchmarkRender(b, stuber.Output{Data: map[string]any{"items": items}}, size, renderOptions{})
		})
	}
}

func sizeName(size int) string {
	return "items=" + json.Number(itoa(size)).String()
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}

	digits := ""
	for v > 0 {
		digits = string(rune('0'+v%10)) + digits
		v /= 10
	}

	return digits
}
