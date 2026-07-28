package history_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/domain/history"
)

func TestMemoryStoreREcordUnlimited(t *testing.T) {
	t.Parallel()

	store := history.NewMemoryStore(0)
	store.Record(history.CallRecord{Service: "svc", Method: "M"})
	store.Record(history.CallRecord{Service: "svc", Method: "N"})

	require.Equal(t, 2, store.Count())
	require.Len(t, store.All(), 2)
}

func TestMemoryStoreREcordWithLimit(t *testing.T) {
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

// Regression: redaction covered only the deprecated singular Request/Response,
// so secrets in the primary Requests/Responses slices (all streaming messages,
// and Requests[0] which aliased the same map) leaked unredacted.
func TestMemoryStoreRedactsPluralMessages(t *testing.T) {
	t.Parallel()

	store := history.NewMemoryStore(0, history.WithRedactKeys([]string{"password"}))
	store.Record(history.CallRecord{
		Service: "svc", Method: "M",
		Requests: []map[string]any{
			{"user": "a", "password": "sekret1"},
			{"user": "b", "password": "sekret2"},
		},
		Responses: []map[string]any{
			{"token": "t", "password": "resp-secret"},
		},
	})

	rec := store.All()[0]

	require.Equal(t, "[REDACTED]", rec.Requests[0]["password"])
	require.Equal(t, "[REDACTED]", rec.Requests[1]["password"], "every streaming message must be redacted")
	require.Equal(t, "[REDACTED]", rec.Responses[0]["password"])
	// Singular mirror stays consistent with the redacted slice.
	require.Equal(t, "[REDACTED]", rec.Request["password"])
	// Non-secret fields survive.
	require.Equal(t, "a", rec.Requests[0]["user"])
}

// Regression: truncation covered only the singular fields, so messageMaxBytes
// never truncated the primary Requests/Responses slices.
func TestMemoryStoreTruncatesPluralMessages(t *testing.T) {
	t.Parallel()

	store := history.NewMemoryStore(0, history.WithMessageMaxBytes(32))
	big := map[string]any{"blob": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}
	store.Record(history.CallRecord{
		Service: "svc", Method: "M",
		Requests:  []map[string]any{big},
		Responses: []map[string]any{big},
	})

	rec := store.All()[0]

	require.Equal(t, true, rec.Requests[0]["_truncated"], "oversized request message must be truncated")
	require.Equal(t, true, rec.Responses[0]["_truncated"])
	require.Equal(t, true, rec.Request["_truncated"])
}

// Regression: an eviction loop bounded by len(calls) > 0 could evict the just-
// appended record when a single call alone exceeded the byte limit, leaving the
// store empty. The newest record must always survive.
func TestMemoryStoreKeepsNewestOversizedRecord(t *testing.T) {
	t.Parallel()

	store := history.NewMemoryStore(16) // tiny limit; one record alone exceeds it
	store.Record(history.CallRecord{
		Service: "svc", Method: "M",
		Request: map[string]any{"blob": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"},
	})

	require.Equal(t, 1, store.Count(), "newest oversized record must be kept, not evicted to empty")
}

// Regression: the store aliased caller-owned message maps, so a caller mutating
// its map after recording would corrupt stored history. Record must deep-copy.
func TestMemoryStoreOwnsRecordedMaps(t *testing.T) {
	t.Parallel()

	store := history.NewMemoryStore(0)

	reqMap := map[string]any{"name": "alice", "nested": map[string]any{"k": "v"}}
	arr := []any{map[string]any{"x": 1}}
	store.Record(history.CallRecord{
		Service: "s", Method: "m",
		Requests:  []map[string]any{reqMap},
		Responses: []map[string]any{{"list": arr}},
	})

	// Mutate the caller's maps AFTER recording.
	reqMap["name"] = "mutated"
	reqMap["nested"].(map[string]any)["k"] = "mutated" //nolint:forcetypeassert
	arr[0].(map[string]any)["x"] = 999                 //nolint:forcetypeassert

	rec := store.All()[0]
	require.Equal(t, "alice", rec.Requests[0]["name"], "stored request must not reflect caller mutation")
	require.Equal(t, "v", rec.Requests[0]["nested"].(map[string]any)["k"]) //nolint:forcetypeassert

	list, ok := rec.Responses[0]["list"].([]any)
	require.True(t, ok)
	require.Equal(t, 1, list[0].(map[string]any)["x"], "nested slice elements must be cloned too") //nolint:forcetypeassert
}

func TestMemoryStoreFilterErrorOnly(t *testing.T) {
	t.Parallel()

	store := &history.MemoryStore{}
	store.Record(history.CallRecord{Service: "a", Method: "ok", Code: 0})
	store.Record(history.CallRecord{Service: "a", Method: "boom", Code: 5, Error: "not found"})
	store.Record(history.CallRecord{Service: "a", Method: "msg-only", Error: "boom"})

	errs := store.Filter(history.FilterOpts{ErrorOnly: true})
	require.Len(t, errs, 2, "only errored calls (non-zero code or error message)")

	for _, c := range errs {
		require.NotEqual(t, "ok", c.Method)
	}

	require.Len(t, store.Filter(history.FilterOpts{}), 3, "no filter returns all")
}

func TestMemoryStoreFIlterCombined(t *testing.T) {
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

func TestMemoryStoreFilterSeq(t *testing.T) {
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

func TestMemoryStoreREcordRedactsSensitiveKeys(t *testing.T) {
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

func TestMemoryStoreREcordRedactsInArrays(t *testing.T) {
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

func TestMemoryStoreREcordTruncatesLargeMessages(t *testing.T) {
	t.Parallel()

	store := history.NewMemoryStore(0, history.WithMessageMaxBytes(50))
	largeReq := map[string]any{"data": string(make([]byte, 200))}
	store.Record(history.CallRecord{Service: "svc", Method: "M", Request: largeReq})

	all := store.All()
	require.Len(t, all, 1)
	require.Equal(t, map[string]any{"_truncated": true}, all[0].Request)
}

func TestMemoryStoreFIlterByMethodBackwardCompat(t *testing.T) {
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
