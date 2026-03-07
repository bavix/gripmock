package deps_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/config"
	"github.com/bavix/gripmock/v3/internal/deps"
	"github.com/bavix/gripmock/v3/internal/domain/history"
)

func TestBuilderHIstoryStoreWithRedactKeys(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		HistoryEnabled:         true,
		HistoryRedactKeys:      []string{"password"},
		HistoryLimit:           config.ByteSize{Bytes: 1 << 20},
		HistoryMessageMaxBytes: 262144,
	}
	builder := deps.NewBuilder(deps.WithConfig(cfg))
	store := builder.HistoryStore()
	require.NotNil(t, store)

	store.Record(history.CallRecord{
		Service:  "svc",
		Method:   "M",
		Request:  map[string]any{"user": "alice", "password": "secret"},
		Response: map[string]any{"ok": true},
	})

	all := store.All()
	require.Len(t, all, 1)
	require.Equal(t, "alice", all[0].Request["user"])
	require.Equal(t, "[REDACTED]", all[0].Request["password"])
}
