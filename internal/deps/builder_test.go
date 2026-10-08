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
		Service:   "svc",
		Method:    "M",
		Requests:  []map[string]any{{"user": "alice", "password": "secret"}},
		Responses: []any{map[string]any{"ok": true}},
	})

	all := store.All()
	require.Len(t, all, 1)
	require.Equal(t, "alice", all[0].Requests[0]["user"])
	require.Equal(t, "[REDACTED]", all[0].Requests[0]["password"])
}

func TestBuilderLogOptionsFollowConfig(t *testing.T) {
	t.Parallel()

	defaults := deps.NewBuilder(deps.WithConfig(config.Config{
		LogRedactMetadata: true,
		LogMessageContent: true,
	})).LogOptions()

	require.True(t, defaults.RedactMetadata)
	require.True(t, defaults.MessageContent)
	require.Empty(t, defaults.RedactKeys)

	custom := deps.NewBuilder(deps.WithConfig(config.Config{
		LogRedactKeys: []string{"x-tenant"},
	})).LogOptions()

	require.False(t, custom.RedactMetadata)
	require.False(t, custom.MessageContent)
	require.Equal(t, []string{"x-tenant"}, custom.RedactKeys)
}
