package history_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/domain/history"
)

func TestMemoryStore_Record_Unlimited(t *testing.T) {
	t.Parallel()

	store := history.NewMemoryStore(0)
	store.Record(history.CallRecord{Service: "svc", Method: "M"})
	store.Record(history.CallRecord{Service: "svc", Method: "N"})

	require.Equal(t, 2, store.Count())
	require.Len(t, store.All(), 2)
}

func TestMemoryStore_Record_WithLimit(t *testing.T) {
	t.Parallel()

	// Each record is ~80-120 bytes as JSON; 200 total limit => expect 1-2 records after eviction
	store := history.NewMemoryStore(200)

	for i := range 10 {
		store.Record(history.CallRecord{Service: "svc", Method: "M", Request: map[string]any{"i": i}})
	}

	// Should evict older records to stay under limit
	require.Less(t, store.Count(), 10)

	all := store.All()
	require.NotEmpty(t, all)

	// Newest records should remain (FIFO eviction)
	require.Contains(t, all[len(all)-1].Request, "i")
}

func TestMemoryStore_Filter_Combined(t *testing.T) {
	t.Parallel()

	store := &history.MemoryStore{}
	store.Record(history.CallRecord{Service: "a", Method: "M1", Session: ""})
	store.Record(history.CallRecord{Service: "a", Method: "M2", Session: ""})
	store.Record(history.CallRecord{Service: "b", Method: "M1", Session: "s1"})
	store.Record(history.CallRecord{Service: "a", Method: "M1", Session: "s1"})
	store.Record(history.CallRecord{Service: "a", Method: "M1", Session: "s2"})

	got := store.Filter(history.FilterOpts{Service: "a", Method: "M1"})
	require.Len(t, got, 3)

	for _, c := range got {
		require.Equal(t, "a", c.Service)
		require.Equal(t, "M1", c.Method)
	}

	got = store.Filter(history.FilterOpts{Service: "a", Method: "M1", Session: "s1"})
	require.Len(t, got, 2)

	for _, c := range got {
		require.True(t, c.Session == "" || c.Session == "s1")
	}
}

func TestMemoryStore_FilterSeq(t *testing.T) {
	t.Parallel()

	store := &history.MemoryStore{}
	store.Record(history.CallRecord{Service: "a", Method: "M"})
	store.Record(history.CallRecord{Service: "b", Method: "M"})
	store.Record(history.CallRecord{Service: "a", Method: "M"})

	var count int

	for range store.FilterSeq(history.FilterOpts{Service: "a", Method: "M"}) {
		count++
	}

	require.Equal(t, 2, count)
}

func TestMemoryStore_Record_RedactsSensitiveKeys(t *testing.T) {
	t.Parallel()

	store := history.NewMemoryStore(0, history.WithRedactKeys([]string{"password", "token", "secret"}))
	store.Record(history.CallRecord{
		Service: "svc",
		Method:  "M",
		Request: map[string]any{
			"user":     "alice",
			"password": "secret123",
			"nested": map[string]any{
				"api_key": "sk-xxx",
				"token":   "jwt-xxx",
			},
		},
		Response: map[string]any{
			"Token":  "bearer-xxx",
			"secret": "confidential",
		},
	})

	all := store.All()
	require.Len(t, all, 1)
	r := all[0]

	require.Equal(t, "alice", r.Request["user"])
	require.Equal(t, "[REDACTED]", r.Request["password"])
	require.IsType(t, map[string]any{}, r.Request["nested"])
	nested, ok := r.Request["nested"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "sk-xxx", nested["api_key"]) // api_key not in redact list
	require.Equal(t, "[REDACTED]", nested["token"])

	require.Equal(t, "[REDACTED]", r.Response["Token"]) // case-insensitive
	require.Equal(t, "[REDACTED]", r.Response["secret"])
}

func TestMemoryStore_Record_RedactsInArrays(t *testing.T) {
	t.Parallel()

	store := history.NewMemoryStore(0, history.WithRedactKeys([]string{"password"}))
	store.Record(history.CallRecord{
		Service: "svc",
		Method:  "M",
		Request: map[string]any{
			"items": []any{
				map[string]any{"name": "a", "password": "p1"},
				map[string]any{"name": "b", "password": "p2"},
			},
		},
	})

	all := store.All()
	require.Len(t, all, 1)
	itemsRaw, ok := all[0].Request["items"].([]any)
	require.True(t, ok)
	require.Len(t, itemsRaw, 2)
	m0, ok := itemsRaw[0].(map[string]any)
	require.True(t, ok)
	m1, ok := itemsRaw[1].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "a", m0["name"])
	require.Equal(t, "[REDACTED]", m0["password"])
	require.Equal(t, "b", m1["name"])
	require.Equal(t, "[REDACTED]", m1["password"])
}

func TestMemoryStore_Record_TruncatesLargeMessages(t *testing.T) {
	t.Parallel()

	store := history.NewMemoryStore(0, history.WithMessageMaxBytes(50))
	largeReq := map[string]any{"data": string(make([]byte, 200))}
	store.Record(history.CallRecord{Service: "svc", Method: "M", Request: largeReq})

	all := store.All()
	require.Len(t, all, 1)
	require.Equal(t, map[string]any{"_truncated": true}, all[0].Request)
}

func TestMemoryStore_FilterByMethod_BackwardCompat(t *testing.T) {
	t.Parallel()

	store := &history.MemoryStore{}
	store.Record(history.CallRecord{Service: "svc", Method: "M"})
	store.Record(history.CallRecord{Service: "svc", Method: "N"})
	store.Record(history.CallRecord{Service: "oth", Method: "M"})

	got := store.FilterByMethod("svc", "M")
	require.Len(t, got, 1)
	require.Equal(t, "svc", got[0].Service)
	require.Equal(t, "M", got[0].Method)
}
