package history_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/domain/history"
)

func responseMap(t *testing.T, response any) map[string]any {
	t.Helper()

	out, ok := response.(map[string]any)
	require.True(t, ok, "response must be an object")

	return out
}

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

	store := history.NewMemoryStore(200)

	for i := range 10 {
		store.Record(history.CallRecord{Service: "svc", Method: "M", Requests: []map[string]any{{"i": i}}})
	}

	require.Less(t, store.Count(), 10)

	all := store.All()
	require.NotEmpty(t, all)

	require.Contains(t, all[len(all)-1].Requests[0], "i")
}

func TestMemoryStoreRedactsPluralMessages(t *testing.T) {
	t.Parallel()

	store := history.NewMemoryStore(0, history.WithRedactKeys([]string{"password"}))
	store.Record(history.CallRecord{
		Service: "svc", Method: "M",
		Requests: []map[string]any{
			{"user": "a", "password": "sekret1"},
			{"user": "b", "password": "sekret2"},
		},
		Responses: []any{
			map[string]any{"token": "t", "password": "resp-secret"},
		},
	})

	rec := store.All()[0]

	require.Equal(t, "[REDACTED]", rec.Requests[0]["password"])
	require.Equal(t, "[REDACTED]", rec.Requests[1]["password"], "every streaming message must be redacted")
	require.Equal(t, "[REDACTED]", responseMap(t, rec.Responses[0])["password"])
	require.Equal(t, "[REDACTED]", rec.Requests[0]["password"])
	require.Equal(t, "a", rec.Requests[0]["user"])
}

func TestMemoryStoreTruncatesPluralMessages(t *testing.T) {
	t.Parallel()

	store := history.NewMemoryStore(0, history.WithMessageMaxBytes(32))
	big := map[string]any{"blob": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}
	store.Record(history.CallRecord{
		Service: "svc", Method: "M",
		Requests:  []map[string]any{big},
		Responses: []any{big},
	})

	rec := store.All()[0]

	require.Equal(t, true, rec.Requests[0]["_truncated"], "oversized request message must be truncated")
	require.Equal(t, true, responseMap(t, rec.Responses[0])["_truncated"])
	require.Equal(t, true, rec.Requests[0]["_truncated"])
}

func TestMemoryStoreKeepsNewestOversizedRecord(t *testing.T) {
	t.Parallel()

	store := history.NewMemoryStore(16)
	store.Record(history.CallRecord{
		Service: "svc", Method: "M",
		Requests: []map[string]any{{"blob": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}},
	})

	require.Equal(t, 1, store.Count(), "newest oversized record must be kept, not evicted to empty")
}

func TestMemoryStoreOwnsRecordedMaps(t *testing.T) {
	t.Parallel()

	store := history.NewMemoryStore(0)

	reqMap := map[string]any{"name": "alice", "nested": map[string]any{"k": "v"}}
	arr := []any{map[string]any{"x": 1}}
	store.Record(history.CallRecord{
		Service: "s", Method: "m",
		Requests:  []map[string]any{reqMap},
		Responses: []any{map[string]any{"list": arr}},
	})

	reqMap["name"] = "mutated"
	reqMap["nested"].(map[string]any)["k"] = "mutated" //nolint:forcetypeassert
	arr[0].(map[string]any)["x"] = 999                 //nolint:forcetypeassert

	rec := store.All()[0]
	require.Equal(t, "alice", rec.Requests[0]["name"], "stored request must not reflect caller mutation")
	require.Equal(t, "v", rec.Requests[0]["nested"].(map[string]any)["k"]) //nolint:forcetypeassert

	list, ok := responseMap(t, rec.Responses[0])["list"].([]any)
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
		Requests: []map[string]any{{
			"user":     "alice",
			"password": "secret123",
			"nested": map[string]any{
				"api_key": "sk-xxx",
				"token":   "jwt-xxx",
			},
		}},
		Responses: []any{map[string]any{
			"Token":  "bearer-xxx",
			"secret": "confidential",
		}},
	})

	all := store.All()
	require.Len(t, all, 1)
	r := all[0]

	require.Equal(t, "alice", r.Requests[0]["user"])
	require.Equal(t, "[REDACTED]", r.Requests[0]["password"])
	require.IsType(t, map[string]any{}, r.Requests[0]["nested"])
	nested, ok := r.Requests[0]["nested"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "sk-xxx", nested["api_key"])
	require.Equal(t, "[REDACTED]", nested["token"])

	require.Equal(t, "[REDACTED]", responseMap(t, r.Responses[0])["Token"])
	require.Equal(t, "[REDACTED]", responseMap(t, r.Responses[0])["secret"])
}

func TestMemoryStoreREcordRedactsInArrays(t *testing.T) {
	t.Parallel()

	store := history.NewMemoryStore(0, history.WithRedactKeys([]string{"password"}))
	store.Record(history.CallRecord{
		Service: "svc",
		Method:  "M",
		Requests: []map[string]any{{
			"items": []any{
				map[string]any{"name": "a", "password": "p1"},
				map[string]any{"name": "b", "password": "p2"},
			},
		}},
	})

	all := store.All()
	require.Len(t, all, 1)
	itemsRaw, ok := all[0].Requests[0]["items"].([]any)
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
	store.Record(history.CallRecord{Service: "svc", Method: "M", Requests: []map[string]any{largeReq}})

	all := store.All()
	require.Len(t, all, 1)
	require.Equal(t, map[string]any{"_truncated": true}, all[0].Requests[0])
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

func TestEvictionAccountingSurvivesChurn(t *testing.T) {
	t.Parallel()

	store := history.NewMemoryStore(2048)

	for i := range 500 {
		store.Record(history.CallRecord{
			Service:  "svc",
			Method:   "M",
			Requests: []map[string]any{{"i": i, "pad": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}},
		})
	}

	kept := store.Count()
	require.Positive(t, kept)
	require.Less(t, kept, 500)

	all := store.All()
	require.Len(t, all, kept)
	require.EqualValues(t, 499, all[len(all)-1].Requests[0]["i"], "the newest record survives eviction")
}

func TestRecordOwnedSkipsCloneButKeepsPolicies(t *testing.T) {
	t.Parallel()

	store := history.NewMemoryStore(0,
		history.WithRedactKeys([]string{"secret"}),
		history.WithMessageMaxBytes(64),
	)

	store.RecordOwned(history.CallRecord{
		Service:  "svc",
		Method:   "M",
		Requests: []map[string]any{{"secret": "hunter2", "ok": "v"}},
	})

	got := store.All()
	require.Len(t, got, 1)
	require.Equal(t, "[REDACTED]", got[0].Requests[0]["secret"])
}

func TestRecordStillProtectsCallerMutation(t *testing.T) {
	t.Parallel()

	store := history.NewMemoryStore(0)
	payload := map[string]any{"name": "before"}

	store.Record(history.CallRecord{Service: "svc", Method: "M", Requests: []map[string]any{payload}})

	payload["name"] = "after"

	require.Equal(t, "before", store.All()[0].Requests[0]["name"])
}

func TestFilterWindowMatchesFilterSemantics(t *testing.T) {
	t.Parallel()

	store := history.NewMemoryStore(0)

	for i := range 25 {
		method := "A"
		if i%3 == 0 {
			method = "B"
		}

		store.Record(history.CallRecord{Service: "svc", Method: method, Requests: []map[string]any{{"i": i}}})
	}

	cases := []struct {
		name   string
		opts   history.FilterOpts
		limit  int
		offset int
	}{
		{"all", history.FilterOpts{}, 0, 0},
		{"limit only", history.FilterOpts{}, 5, 0},
		{"limit and offset", history.FilterOpts{}, 5, 3},
		{"offset only", history.FilterOpts{}, 0, 4},
		{"limit past total", history.FilterOpts{}, 100, 0},
		{"offset past total", history.FilterOpts{}, 5, 100},
		{"filtered", history.FilterOpts{Method: "B"}, 3, 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			all := store.Filter(tc.opts)
			end := max(len(all)-tc.offset, 0)
			start := 0

			if tc.limit > 0 {
				start = max(end-tc.limit, 0)
			}

			want := all[start:end]

			got, total := store.FilterWindow(tc.opts, tc.limit, tc.offset)

			require.Equal(t, len(all), total)
			require.Len(t, got, len(want))

			for i := range want {
				require.Equal(t, want[i].Requests[0]["i"], got[i].Requests[0]["i"])
			}
		})
	}
}

func TestFilterSessionVisibility(t *testing.T) {
	t.Parallel()

	store := history.NewMemoryStore(0)
	store.Record(history.CallRecord{Service: "svc", Method: "M", Session: "a"})
	store.Record(history.CallRecord{Service: "svc", Method: "M", Session: "b"})
	store.Record(history.CallRecord{Service: "svc", Method: "M"})

	scoped := store.Filter(history.FilterOpts{Session: "a"})
	require.Len(t, scoped, 2, "a session sees its own calls plus the global ones")

	for _, call := range scoped {
		require.NotEqual(t, "b", call.Session)
	}

	unscoped := store.Filter(history.FilterOpts{})
	require.Len(t, unscoped, 3,
		"no session means the operator view: every call, including other sessions'. "+
			"Stub visibility is narrower (unscoped sees only global stubs), so this asymmetry is deliberate")
}
