package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/config"
)

func TestConfig_Defaults(t *testing.T) {
	t.Parallel()

	conf := config.Load()

	require.Equal(t, "info", conf.LogLevel)

	require.False(t, conf.StrictMethodTitle)

	require.Equal(t, "tcp", conf.GRPCNetwork)
	require.Equal(t, "0.0.0.0", conf.GRPCHost)
	require.Equal(t, "4770", conf.GRPCPort)
	require.Equal(t, "0.0.0.0:4770", conf.GRPCAddr)

	require.Equal(t, "0.0.0.0", conf.HTTPHost)
	require.Equal(t, "4771", conf.HTTPPort)
	require.Equal(t, "0.0.0.0:4771", conf.HTTPAddr)
}

//nolint:paralleltest
func TestConfig_Override(t *testing.T) {
	env := map[string]string{
		"LOG_LEVEL":           "trace",
		"PACKAGE_SIMPLER":     "false",
		"STRICT_METHOD_TITLE": "false",
		"GRPC_NETWORK":        "udp",
		"GRPC_HOST":           "111.111.111.111",
		"GRPC_PORT":           "1111",
		"HTTP_HOST":           "192.168.1.2",
		"HTTP_PORT":           "2000",
	}

	for k, v := range env {
		t.Setenv(k, v)
	}

	conf := config.Load()

	require.Equal(t, "trace", conf.LogLevel)

	require.False(t, conf.StrictMethodTitle)

	require.Equal(t, "udp", conf.GRPCNetwork)
	require.Equal(t, "111.111.111.111", conf.GRPCHost)
	require.Equal(t, "1111", conf.GRPCPort)
	require.Equal(t, "111.111.111.111:1111", conf.GRPCAddr)

	require.Equal(t, "192.168.1.2", conf.HTTPHost)
	require.Equal(t, "2000", conf.HTTPPort)
	require.Equal(t, "192.168.1.2:2000", conf.HTTPAddr)
}
