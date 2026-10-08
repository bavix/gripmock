package stuber

import (
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func cachedSessions(t *testing.T, budgerigar *Budgerigar) int {
	t.Helper()

	budgerigar.searcher.lookupMu.RLock()
	defer budgerigar.searcher.lookupMu.RUnlock()

	return len(budgerigar.searcher.lookupCache)
}

func hasCachedSession(t *testing.T, budgerigar *Budgerigar, session string) bool {
	t.Helper()

	budgerigar.searcher.lookupMu.RLock()
	defer budgerigar.searcher.lookupMu.RUnlock()

	_, ok := budgerigar.searcher.lookupCache[session]

	return ok
}

func searchInSession(t *testing.T, budgerigar *Budgerigar, session string) {
	t.Helper()

	_, _ = budgerigar.FindByQuery(Query{
		Service: "svc",
		Method:  "M",
		Session: session,
		Input:   []map[string]any{{"k": "v"}},
	})
}

func TestDeleteSessionEvictsLookupCache(t *testing.T) {
	t.Parallel()

	budgerigar := NewBudgerigar()
	budgerigar.PutMany(&Stub{
		ID:      uuid.New(),
		Service: "svc",
		Method:  "M",
		Session: "s1",
		Input:   InputData{Glob: map[string]any{"k": "v*"}},
		Output:  Output{Data: map[string]any{"ok": true}},
	})

	searchInSession(t, budgerigar, "s1")
	require.True(t, hasCachedSession(t, budgerigar, "s1"), "search must populate the lookup cache")

	budgerigar.DeleteSession("s1")
	require.False(t, hasCachedSession(t, budgerigar, "s1"), "DeleteSession must evict the session's lookup cache entry")
}

func TestDeleteSessionEvictsLookupCacheForGlobalOnlySessions(t *testing.T) {
	t.Parallel()

	budgerigar := NewBudgerigar()
	budgerigar.PutMany(&Stub{
		ID:      uuid.New(),
		Service: "svc",
		Method:  "M",
		Input:   InputData{Glob: map[string]any{"k": "v*"}},
		Output:  Output{Data: map[string]any{"ok": true}},
	})

	const sessions = 50

	for i := range sessions {
		searchInSession(t, budgerigar, "session-"+strconv.Itoa(i))
	}

	require.Equal(t, sessions, cachedSessions(t, budgerigar))

	for i := range sessions {
		budgerigar.DeleteSession("session-" + strconv.Itoa(i))
	}

	require.Zero(t, cachedSessions(t, budgerigar), "lookup cache must not outlive the sessions that created it")
}
