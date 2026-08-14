package config

import (
	"testing"
	"time"

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

//nolint:paralleltest // t.Setenv forbids t.Parallel.
func TestParseGRPCLimitsDefaults(t *testing.T) {
	for _, k := range []string{
		"GRPC_MAX_RECV_MSG_SIZE", "GRPC_MAX_SEND_MSG_SIZE",
		"GRPC_KEEPALIVE_TIME", "GRPC_KEEPALIVE_TIMEOUT",
		"GRPC_KEEPALIVE_MAX_CONNECTION_IDLE", "GRPC_KEEPALIVE_MAX_CONNECTION_AGE",
	} {
		t.Setenv(k, "")
	}

	cfg := Load()
	require.Equal(t, int64(4*1024*1024), cfg.GRPCLimits.MaxRecvMsgSize.Int64())
	require.Equal(t, int64(0), cfg.GRPCLimits.MaxSendMsgSize.Int64(), "0 means unlimited")
	require.Equal(t, 30*time.Second, cfg.GRPCLimits.KeepaliveTime)
	require.Equal(t, 10*time.Second, cfg.GRPCLimits.KeepaliveTimeout)
	require.Equal(t, 5*time.Minute, cfg.GRPCLimits.KeepaliveMaxConnectionIdle)
	require.Equal(t, 30*time.Minute, cfg.GRPCLimits.KeepaliveMaxConnectionAge)
}

func TestParseGRPCLimitsEnv(t *testing.T) {
	// cannot use t.Parallel with t.Setenv
	t.Setenv("GRPC_MAX_RECV_MSG_SIZE", "16M")
	t.Setenv("GRPC_MAX_SEND_MSG_SIZE", "8388608")
	t.Setenv("GRPC_KEEPALIVE_TIME", "5s")
	t.Setenv("GRPC_KEEPALIVE_MAX_CONNECTION_AGE", "1m")

	cfg := Load()
	require.Equal(t, int64(16*1024*1024), cfg.GRPCLimits.MaxRecvMsgSize.Int64())
	require.Equal(t, int64(8*1024*1024), cfg.GRPCLimits.MaxSendMsgSize.Int64())
	require.Equal(t, 5*time.Second, cfg.GRPCLimits.KeepaliveTime)
	require.Equal(t, time.Minute, cfg.GRPCLimits.KeepaliveMaxConnectionAge)
	require.Equal(t, 10*time.Second, cfg.GRPCLimits.KeepaliveTimeout, "untouched var keeps its default")
}
