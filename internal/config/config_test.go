package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/config"
)

func TestConfig_OldEnvVarNames(t *testing.T) {
	// Set environment variables
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("STRICT_METHOD_TITLE", "true")
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

	expected := config.Config{
		LogLevel:               "debug",
		StrictMethodTitle:      true,
		GRPCNetwork:            "tcp",
		GRPCHost:               "127.0.0.1",
		GRPCPort:               "8080",
		GRPCAddr:               "127.0.0.1:8080",
		HTTPHost:               "localhost",
		HTTPPort:               "8081",
		HTTPAddr:               "localhost:8081",
		StubWatcherEnabled:     false,
		StubWatcherInterval:    5 * 1000000000, // 5s in nanoseconds
		StubWatcherType:        "polling",
		HistoryEnabled:         false,
		HistoryLimit:           config.ByteSize{Bytes: 128 * 1024 * 1024}, // 128M
		HistoryMessageMaxBytes: 524288,
		HistoryRedactKeys:      []string{"password", "token", "secret"},
	}

	// Load configuration
	cfg := config.Load()

	// Assert values
	assert.Equal(t, expected.LogLevel, cfg.LogLevel)
	assert.Equal(t, expected.StrictMethodTitle, cfg.StrictMethodTitle)
	assert.Equal(t, expected.GRPCNetwork, cfg.GRPCNetwork)
	assert.Equal(t, expected.GRPCHost, cfg.GRPCHost)
	assert.Equal(t, expected.GRPCPort, cfg.GRPCPort)
	assert.Equal(t, expected.GRPCAddr, cfg.GRPCAddr)
	assert.Equal(t, expected.HTTPHost, cfg.HTTPHost)
	assert.Equal(t, expected.HTTPPort, cfg.HTTPPort)
	assert.Equal(t, expected.HTTPAddr, cfg.HTTPAddr)
	assert.Equal(t, expected.StubWatcherEnabled, cfg.StubWatcherEnabled)
	assert.Equal(t, expected.StubWatcherInterval, cfg.StubWatcherInterval)
	assert.Equal(t, expected.StubWatcherType, cfg.StubWatcherType)
	assert.Equal(t, expected.HistoryEnabled, cfg.HistoryEnabled)
	assert.Equal(t, expected.HistoryLimit.Bytes, cfg.HistoryLimit.Bytes)
	assert.Equal(t, expected.HistoryMessageMaxBytes, cfg.HistoryMessageMaxBytes)
	assert.Equal(t, expected.HistoryRedactKeys, cfg.HistoryRedactKeys)
}

func TestConfig_DefaultValues(t *testing.T) {
	t.Parallel()

	expected := config.Config{
		LogLevel:               "info",
		StrictMethodTitle:      false,
		GRPCNetwork:            "tcp",
		GRPCHost:               "0.0.0.0",
		GRPCPort:               "4770",
		GRPCAddr:               "0.0.0.0:4770",
		HTTPHost:               "0.0.0.0",
		HTTPPort:               "4771",
		HTTPAddr:               "0.0.0.0:4771",
		StubWatcherEnabled:     true,
		StubWatcherInterval:    1 * 1000000000, // 1s in nanoseconds
		StubWatcherType:        "fsnotify",
		HistoryEnabled:         true,
		HistoryLimit:           config.ByteSize{Bytes: 64 * 1024 * 1024}, // 64M
		HistoryMessageMaxBytes: 262144,
		HistoryRedactKeys:      nil, // env v11 returns nil for empty strings
	}

	// Load configuration
	cfg := config.Load()

	// Assert values
	assert.Equal(t, expected.LogLevel, cfg.LogLevel)
	assert.Equal(t, expected.StrictMethodTitle, cfg.StrictMethodTitle)
	assert.Equal(t, expected.GRPCNetwork, cfg.GRPCNetwork)
	assert.Equal(t, expected.GRPCHost, cfg.GRPCHost)
	assert.Equal(t, expected.GRPCPort, cfg.GRPCPort)
	assert.Equal(t, expected.GRPCAddr, cfg.GRPCAddr)
	assert.Equal(t, expected.HTTPHost, cfg.HTTPHost)
	assert.Equal(t, expected.HTTPPort, cfg.HTTPPort)
	assert.Equal(t, expected.HTTPAddr, cfg.HTTPAddr)
	assert.Equal(t, expected.StubWatcherEnabled, cfg.StubWatcherEnabled)
	assert.Equal(t, expected.StubWatcherInterval, cfg.StubWatcherInterval)
	assert.Equal(t, expected.StubWatcherType, cfg.StubWatcherType)
	assert.Equal(t, expected.HistoryEnabled, cfg.HistoryEnabled)
	assert.Equal(t, expected.HistoryLimit.Bytes, cfg.HistoryLimit.Bytes)
	assert.Equal(t, expected.HistoryMessageMaxBytes, cfg.HistoryMessageMaxBytes)
	assert.Equal(t, expected.HistoryRedactKeys, cfg.HistoryRedactKeys)
}

func TestConfig_ByteSize(t *testing.T) {
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
			assert.Equal(t, tc.expected, bs.Bytes)
		})
	}
}

func TestConfig_New(t *testing.T) {
	t.Parallel()

	cfg := config.Load()
	assert.NotZero(t, cfg)
}
