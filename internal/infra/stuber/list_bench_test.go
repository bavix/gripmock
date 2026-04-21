package stuber_test

import (
	"strconv"
	"testing"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func BenchmarkListFilterSortPaginate(b *testing.B) {
	budgerigar := stuber.NewBudgerigar()

	for i := range 10000 {
		source := "rest"
		if i%4 == 0 {
			source = "proxy"
		}

		budgerigar.PutMany(&stuber.Stub{
			Service:  "svc." + strconv.Itoa(i%200),
			Method:   "Method" + strconv.Itoa(i%20),
			Priority: i % 17,
			Session:  "s" + strconv.Itoa(i%10),
			Source:   source,
			Input: stuber.InputData{
				Equals: map[string]any{"id": i},
			},
			Output: stuber.Output{Data: map[string]any{"ok": true}},
		})
	}

	options := stuber.ListOptions{
		Source:     "proxy",
		Service:    "svc.8",
		Method:     "Method8",
		SessionSet: true,
		Session:    "s8",
		Sort:       stuber.ListSortPriorityDesc,
		Limit:      20,
		Offset:     10,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = budgerigar.List(options)
	}
}

func BenchmarkListDefaultSortLargeSet(b *testing.B) {
	budgerigar := stuber.NewBudgerigar()

	for i := range 20000 {
		budgerigar.PutMany(&stuber.Stub{
			Service:  "svc." + strconv.Itoa(i%400),
			Method:   "M" + strconv.Itoa(i%40),
			Priority: i % 31,
			Source:   "rest",
			Input: stuber.InputData{
				Equals: map[string]any{"id": i},
			},
			Output: stuber.Output{Data: map[string]any{"ok": true}},
		})
	}

	options := stuber.ListOptions{Sort: stuber.ListSortPriorityDesc, Limit: 100}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = budgerigar.List(options)
	}
}

func BenchmarkAllImmutableCloneDeepPayload(b *testing.B) {
	budgerigar := stuber.NewBudgerigar()

	for i := range 5000 {
		budgerigar.PutMany(&stuber.Stub{
			Service: "svc.deep",
			Method:  "Clone",
			Source:  "rest",
			Input: stuber.InputData{
				Equals: map[string]any{
					"id": i,
					"nested": map[string]any{
						"items": []any{i, i + 1, map[string]any{"x": "y"}},
					},
				},
			},
			Output: stuber.Output{
				Data: map[string]any{
					"status": "ok",
					"payload": []any{
						map[string]any{"k": "v"},
						[]any{1, 2, 3},
					},
				},
			},
		})
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = budgerigar.All()
	}
}

func BenchmarkAllImmutableCloneSimplePayload(b *testing.B) {
	budgerigar := stuber.NewBudgerigar()

	for i := range 20000 {
		budgerigar.PutMany(&stuber.Stub{
			Service:  "svc.simple",
			Method:   "Clone",
			Priority: i % 5,
			Source:   "rest",
			Input:    stuber.InputData{},
			Output:   stuber.Output{},
		})
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = budgerigar.All()
	}
}
