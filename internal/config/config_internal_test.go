package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseHistoryDefaults(t *testing.T) {
	// clear env via Setenv to ensure defaults (cannot use t.Parallel with t.Setenv)
	t.Setenv("HISTORY_LIMIT", "")
	t.Setenv("HISTORY_REDACT_KEYS", "")
	t.Setenv("HISTORY_MESSAGE_MAX_BYTES", "")
	t.Setenv("HISTORY_ENABLED", "")

	cfg := Load()
	require.True(t, cfg.HistoryEnabled, "history should be enabled by default")
	require.Equal(t, int64(64*1024*1024), cfg.HistoryLimit.Int64(), "unexpected default limit")
	require.EqualValues(t, 262144, cfg.HistoryMessageMaxBytes, "unexpected default max bytes")
	require.Empty(t, cfg.HistoryRedactKeys, "expected no redact keys by default")
}

func TestParseHistoryEnv(t *testing.T) {
	// uses t.Setenv, cannot use t.Parallel
	toGiB := int64(1024 * 1024 * 1024)

	t.Setenv("HISTORY_LIMIT", "1G")
	t.Setenv("HISTORY_REDACT_KEYS", "password,token,secret")
	t.Setenv("HISTORY_MESSAGE_MAX_BYTES", "1024")
	t.Setenv("HISTORY_ENABLED", "false")

	cfg := Load()
	require.False(t, cfg.HistoryEnabled, "history should be disabled by env")
	require.Equal(t, toGiB, cfg.HistoryLimit.Int64(), "unexpected limit")
	require.EqualValues(t, 1024, cfg.HistoryMessageMaxBytes, "unexpected max bytes")
	require.Equal(t, []string{"password", "token", "secret"}, cfg.HistoryRedactKeys, "unexpected redact keys")
}
