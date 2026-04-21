package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/config"
)

func TestConfigDefaults(t *testing.T) {
	t.Parallel()

	conf := config.Load()

	require.Equal(t, "info", conf.LogLevel)

	require.Equal(t, "0.0.0.0", conf.HTTPHost)
	require.Equal(t, "4771", conf.HTTPPort)
	require.Equal(t, "0.0.0.0:4771", conf.HTTPAddr)

	require.Empty(t, conf.GRPCTLSCertFile)
	require.Empty(t, conf.GRPCTLSKeyFile)
	require.False(t, conf.GRPCTLSClientAuth)
	require.Empty(t, conf.GRPCTLSCAFile)
	require.Equal(t, "1.2", conf.GRPCTLSMinVersion)

	require.Empty(t, conf.HTTPTLSCertFile)
	require.Empty(t, conf.HTTPTLSKeyFile)
	require.False(t, conf.HTTPTLSClientAuth)
	require.Empty(t, conf.HTTPTLSCAFile)
}

//nolint:paralleltest
func TestConfigOverride(t *testing.T) {
	env := map[string]string{
		"LOG_LEVEL":            "trace",
		"PACKAGE_SIMPLER":      "false",
		"GRPC_NETWORK":         "udp",
		"GRPC_HOST":            "111.111.111.111",
		"GRPC_PORT":            "1111",
		"HTTP_HOST":            "192.168.1.2",
		"HTTP_PORT":            "2000",
		"GRPC_TLS_CERT_FILE":   "grpc-cert.pem",
		"GRPC_TLS_KEY_FILE":    "grpc-key.pem",
		"GRPC_TLS_CLIENT_AUTH": "true",
		"GRPC_TLS_CA_FILE":     "grpc-ca.pem",
		"GRPC_TLS_MIN_VERSION": "1.3",
		"HTTP_TLS_CERT_FILE":   "http-cert.pem",
		"HTTP_TLS_KEY_FILE":    "http-key.pem",
		"HTTP_TLS_CLIENT_AUTH": "true",
		"HTTP_TLS_CA_FILE":     "http-ca.pem",
	}

	for k, v := range env {
		t.Setenv(k, v)
	}

	conf := config.Load()

	require.Equal(t, "trace", conf.LogLevel)

	require.Equal(t, "udp", conf.GRPCNetwork)
	require.Equal(t, "111.111.111.111", conf.GRPCHost)
	require.Equal(t, "1111", conf.GRPCPort)
	require.Equal(t, "111.111.111.111:1111", conf.GRPCAddr)

	require.Equal(t, "192.168.1.2", conf.HTTPHost)
	require.Equal(t, "2000", conf.HTTPPort)
	require.Equal(t, "192.168.1.2:2000", conf.HTTPAddr)

	require.Equal(t, "grpc-cert.pem", conf.GRPCTLSCertFile)
	require.Equal(t, "grpc-key.pem", conf.GRPCTLSKeyFile)
	require.True(t, conf.GRPCTLSClientAuth)
	require.Equal(t, "grpc-ca.pem", conf.GRPCTLSCAFile)
	require.Equal(t, "1.3", conf.GRPCTLSMinVersion)

	require.Equal(t, "http-cert.pem", conf.HTTPTLSCertFile)
	require.Equal(t, "http-key.pem", conf.HTTPTLSKeyFile)
	require.True(t, conf.HTTPTLSClientAuth)
	require.Equal(t, "http-ca.pem", conf.HTTPTLSCAFile)
}
