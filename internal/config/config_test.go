package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/config"
)

func assertConfig(t *testing.T, expected config.Config) {
	t.Helper()

	cfg := config.Load()

	require.Equal(t, expected.LogLevel, cfg.LogLevel)
	require.Equal(t, expected.GRPCNetwork, cfg.GRPCNetwork)
	require.Equal(t, expected.GRPC.Host, cfg.GRPC.Host)
	require.Equal(t, expected.GRPC.Port, cfg.GRPC.Port)
	require.Equal(t, expected.GRPC.Addr, cfg.GRPC.Addr)
	require.Equal(t, expected.HTTP.Host, cfg.HTTP.Host)
	require.Equal(t, expected.HTTP.Port, cfg.HTTP.Port)
	require.Equal(t, expected.HTTP.Addr, cfg.HTTP.Addr)
	require.Equal(t, expected.StubWatcherEnabled, cfg.StubWatcherEnabled)
	require.Equal(t, expected.StubWatcherInterval, cfg.StubWatcherInterval)
	require.Equal(t, expected.StubWatcherType, cfg.StubWatcherType)
	require.Equal(t, expected.HistoryEnabled, cfg.HistoryEnabled)
	require.Equal(t, expected.HistoryLimit.Bytes, cfg.HistoryLimit.Bytes)
	require.Equal(t, expected.HistoryMessageMaxBytes, cfg.HistoryMessageMaxBytes)
	require.Equal(t, expected.HistoryRedactKeys, cfg.HistoryRedactKeys)
}

func TestConfigOldEnvVarNames(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")

	t.Setenv("GRPC_NETWORK", "tcp")
	t.Setenv("GRPC_HOST", "127.0.0.1")
	t.Setenv("GRPC_PORT", "8080")
	t.Setenv("HTTP_HOST", "localhost")
	t.Setenv("HTTP_PORT", "8081")
	t.Setenv("STUB_WATCHER_ENABLED", "false")
	t.Setenv("STUB_WATCHER_INTERVAL", "5s")
	t.Setenv("STUB_WATCHER_TYPE", "polling")
	t.Setenv("HISTORY_ENABLED", "false")
	t.Setenv("HISTORY_LIMIT", "128M")
	t.Setenv("HISTORY_MESSAGE_MAX_BYTES", "524288")
	t.Setenv("HISTORY_REDACT_KEYS", "password,token,secret")

	assertConfig(t, config.Config{
		LogLevel:               "debug",
		GRPCNetwork:            "tcp",
		GRPC:                   config.ServerConfig{Host: "127.0.0.1", Port: "8080", Addr: "127.0.0.1:8080"},
		HTTP:                   config.ServerConfig{Host: "localhost", Port: "8081", Addr: "localhost:8081"},
		StubWatcherEnabled:     false,
		StubWatcherInterval:    5 * 1000000000, // 5s in nanoseconds
		StubWatcherType:        "polling",
		HistoryEnabled:         false,
		HistoryLimit:           config.ByteSize{Bytes: 128 * 1024 * 1024}, // 128M
		HistoryMessageMaxBytes: 524288,
		HistoryRedactKeys:      []string{"password", "token", "secret"},
	})
}

func TestConfigDefaultValues(t *testing.T) {
	t.Parallel()

	assertConfig(t, config.Config{
		LogLevel:               "info",
		GRPCNetwork:            "tcp",
		GRPC:                   config.ServerConfig{Host: "0.0.0.0", Port: "4770", Addr: "0.0.0.0:4770"},
		HTTP:                   config.ServerConfig{Host: "0.0.0.0", Port: "4771", Addr: "0.0.0.0:4771"},
		StubWatcherEnabled:     true,
		StubWatcherInterval:    1 * 1000000000, // 1s in nanoseconds
		StubWatcherType:        "fsnotify",
		HistoryEnabled:         true,
		HistoryLimit:           config.ByteSize{Bytes: 64 * 1024 * 1024}, // 64M
		HistoryMessageMaxBytes: 262144,
		HistoryRedactKeys:      nil,
	})
}

func TestConfigByteSize(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		input    string
		expected int64
	}{
		{"1024", 1024},
		{"1K", 1024},
		{"1M", 1024 * 1024},
		{"1G", 1024 * 1024 * 1024},
		{"64M", 64 * 1024 * 1024},
		{"128K", 128 * 1024},
		{"2G", 2 * 1024 * 1024 * 1024},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			var bs config.ByteSize

			err := bs.UnmarshalText([]byte(tc.input))
			require.NoError(t, err)
			require.Equal(t, tc.expected, bs.Bytes)
		})
	}
}

func TestConfigNew(t *testing.T) {
	t.Parallel()

	cfg := config.Load()
	require.NotZero(t, cfg)
}
