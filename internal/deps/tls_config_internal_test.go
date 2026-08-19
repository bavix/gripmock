package deps

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/config"
	infraTLS "github.com/bavix/gripmock/v3/internal/infra/tls"
)

func TestHTTPGatewayTLSMinVersionFromConfig(t *testing.T) {
	t.Parallel()

	cfg := config.Config{}
	cfg.HTTPTLS.MinVersion = infraTLS.MinTLSVersion13
	cfg.GatewayTLS.MinVersion = infraTLS.MinTLSVersion13

	b := NewBuilder(WithConfig(cfg))

	require.Equal(t, infraTLS.MinTLSVersion13, b.httpTLSConfig().MinVersion)
	require.Equal(t, infraTLS.MinTLSVersion13, b.gatewayTLSConfig().MinVersion)
}
