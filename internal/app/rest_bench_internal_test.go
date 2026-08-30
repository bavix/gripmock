package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/domain/rest"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func benchServerWith(b *testing.B, stubs, calls int) *RestServer {
	b.Helper()

	budgerigar := stuber.NewBudgerigar()
	store := history.NewMemoryStore(0)

	for i := range stubs {
		budgerigar.PutMany(&stuber.Stub{
			ID:      uuid.New(),
			Service: "svc.Service",
			Method:  "Method",
			Input:   stuber.InputData{Equals: map[string]any{"id": i, "name": "filler"}},
			Output:  stuber.Output{Data: map[string]any{"ok": true, "index": i}},
		})
	}

	for i := range calls {
		store.Record(history.CallRecord{
			Service:   "svc.Service",
			Method:    "Method",
			Requests:  []map[string]any{{"id": i}},
			Responses: []any{map[string]any{"ok": true}},
		})
	}

	server, err := NewRestServer(b.Context(), budgerigar, &mockExtender{}, store, nil, nil, nil)
	if err != nil {
		b.Fatal(err)
	}

	return server
}

func BenchmarkRestListStubs(b *testing.B) {
	server := benchServerWith(b, 1000, 0)
	limit := 50

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(b.Context(), http.MethodGet, "/api/stubs", nil)

		server.ListStubs(rec, req, rest.ListStubsParams{Limit: &limit})
	}
}

func BenchmarkRestHistory(b *testing.B) {
	server := benchServerWith(b, 0, 10000)
	limit := 50

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(b.Context(), http.MethodGet, "/api/history", nil)

		server.ListHistory(rec, req, rest.ListHistoryParams{Limit: &limit})
	}
}

func BenchmarkRestDashboard(b *testing.B) {
	server := benchServerWith(b, 1000, 10000)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(b.Context(), http.MethodGet, "/api/dashboard", nil)

		server.Dashboard(rec, req)
	}
}
