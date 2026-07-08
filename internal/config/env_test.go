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

	require.Equal(t, "0.0.0.0", conf.GRPC.Host)
	require.Equal(t, "4770", conf.GRPC.Port)
	require.Equal(t, "0.0.0.0:4770", conf.GRPC.Addr)

	require.Equal(t, "0.0.0.0", conf.HTTP.Host)
	require.Equal(t, "4771", conf.HTTP.Port)
	require.Equal(t, "0.0.0.0:4771", conf.HTTP.Addr)

	require.Empty(t, conf.GRPCTLS.CertFile)
	require.Empty(t, conf.GRPCTLS.KeyFile)
	require.False(t, conf.GRPCTLS.ClientAuth)
	require.Empty(t, conf.GRPCTLS.CAFile)
	require.Equal(t, "1.2", conf.GRPCTLS.MinVersion)

	require.Empty(t, conf.HTTPTLS.CertFile)
	require.Empty(t, conf.HTTPTLS.KeyFile)
	require.False(t, conf.HTTPTLS.ClientAuth)
	require.Empty(t, conf.HTTPTLS.CAFile)
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
	require.Equal(t, "111.111.111.111", conf.GRPC.Host)
	require.Equal(t, "1111", conf.GRPC.Port)
	require.Equal(t, "111.111.111.111:1111", conf.GRPC.Addr)

	require.Equal(t, "192.168.1.2", conf.HTTP.Host)
	require.Equal(t, "2000", conf.HTTP.Port)
	require.Equal(t, "192.168.1.2:2000", conf.HTTP.Addr)

	require.Equal(t, "grpc-cert.pem", conf.GRPCTLS.CertFile)
	require.Equal(t, "grpc-key.pem", conf.GRPCTLS.KeyFile)
	require.True(t, conf.GRPCTLS.ClientAuth)
	require.Equal(t, "grpc-ca.pem", conf.GRPCTLS.CAFile)
	require.Equal(t, "1.3", conf.GRPCTLS.MinVersion)

	require.Equal(t, "http-cert.pem", conf.HTTPTLS.CertFile)
	require.Equal(t, "http-key.pem", conf.HTTPTLS.KeyFile)
	require.True(t, conf.HTTPTLS.ClientAuth)
	require.Equal(t, "http-ca.pem", conf.HTTPTLS.CAFile)
}
