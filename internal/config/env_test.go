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
	require.Equal(t, "0.0.0.0:4770", conf.GRPCAddr)

	require.Equal(t, "0.0.0.0:4771", conf.HTTPAddr)
}

//nolint:paralleltest
func TestConfig_Override(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv() - this is intentional
	env := map[string]string{
		"LOG_LEVEL":           "info", // simplified version only supports basic fields
		"PACKAGE_SIMPLER":     "false",
		"STRICT_METHOD_TITLE": "false",
		"GRPC_NETWORK":        "tcp", // simplified version uses default
		"GRPC_HOST":           "192.168.1.1",
		"GRPC_PORT":           "1000",
		"HTTP_HOST":           "192.168.1.2",
		"HTTP_PORT":           "2000",
	}

	for k, v := range env {
		t.Setenv(k, v)
	}

	conf := config.Load()

	require.Equal(t, "info", conf.LogLevel) // simplified version uses default

	require.False(t, conf.StrictMethodTitle)

	require.Equal(t, "tcp", conf.GRPCNetwork)           // simplified version uses default
	require.Equal(t, "192.168.1.1:1000", conf.GRPCAddr) // computed from GRPC_HOST and GRPC_PORT

	require.Equal(t, "192.168.1.2:2000", conf.HTTPAddr) // computed from HTTP_HOST and HTTP_PORT
}
