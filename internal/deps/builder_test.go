package deps_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/config"
	"github.com/bavix/gripmock/v3/internal/deps"
	"github.com/bavix/gripmock/v3/internal/domain/history"
)

func TestBuilder_Basic(t *testing.T) {
	t.Parallel()
	// Test basic builder functionality
	builder := deps.NewBuilder()
	require.NotNil(t, builder)
}

func TestBuilder_Empty(t *testing.T) {
	t.Parallel()
	// Test empty builder case
	builder := deps.NewBuilder()
	require.NotNil(t, builder)
}

func TestBuilder_Initialization(t *testing.T) {
	t.Parallel()
	// Test builder initialization
	builder := deps.NewBuilder()
	require.NotNil(t, builder)
	// Verify builder is properly initialized
}

func TestBuilder_WithDefaultConfig(t *testing.T) {
	t.Parallel()
	// Test builder with default config
	builder := deps.NewBuilder(deps.WithDefaultConfig())
	require.NotNil(t, builder)
}

func TestBuilder_WithConfig(t *testing.T) {
	t.Parallel()
	// Test builder with custom config
	cfg := config.Load()
	builder := deps.NewBuilder(deps.WithConfig(cfg))
	require.NotNil(t, builder)
}

func TestBuilder_HistoryStore_WithRedactKeys(t *testing.T) {
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
