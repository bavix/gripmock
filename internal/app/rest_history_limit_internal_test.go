package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/domain/rest"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// ListHistory supports ?limit=N (most recent N), ?offset=M (skip newest M),
// ?service/?method filters, and an X-Total-Count header.
func TestListHistoryLimitAndFilter(t *testing.T) {
	t.Parallel()

	store := history.NewMemoryStore(0)
	for range 5 {
		store.Record(history.CallRecord{Service: "svc", Method: "GetProduct"})
	}

	store.Record(history.CallRecord{Service: "svc", Method: "Other"})

	srv, err := NewRestServer(t.Context(), stuber.NewBudgerigar(), &mockExtender{}, store, nil, nil, nil)
	require.NoError(t, err)

	get := func(params rest.ListHistoryParams) (rest.HistoryList, string) {
		rec := httptest.NewRecorder()
		srv.ListHistory(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/history", nil), params)
		require.Equal(t, http.StatusOK, rec.Code)

		var out rest.HistoryList
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))

		return out, rec.Header().Get("X-Total-Count")
	}

	cases := []struct {
		name      string
		params    rest.ListHistoryParams
		wantLen   int
		wantTotal string
	}{
		{"no filter", rest.ListHistoryParams{}, 6, "6"},
		{"limit=2", rest.ListHistoryParams{Limit: new(2)}, 2, "6"},
		// offset skips the newest record; total header still reflects the full set.
		{"limit=2&offset=2", rest.ListHistoryParams{Limit: new(2), Offset: new(2)}, 2, "6"},
		// offset past the end yields an empty slice, not a panic.
		{"offset past end", rest.ListHistoryParams{Limit: new(2), Offset: new(100)}, 0, "6"},
		{"method filter", rest.ListHistoryParams{Method: new("GetProduct")}, 5, "5"},
		{"method+limit", rest.ListHistoryParams{Method: new("GetProduct"), Limit: new(3)}, 3, "5"},
	}

	for _, tc := range cases {
		out, total := get(tc.params)
		require.Lenf(t, out, tc.wantLen, "%s: length", tc.name)
		require.Equalf(t, tc.wantTotal, total, "%s: total", tc.name)
	}
}
