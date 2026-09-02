package app

import (
	"bytes"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	protoloc "github.com/bavix/gripmock/v3/internal/domain/proto"
)

func TestBuildProxiesWithBindingsRedactsBearer(t *testing.T) {
	t.Parallel()

	const (
		proxyToken  = "proxy-s3cr3t"
		sourceToken = "source-s3cr3t"
	)

	var logs bytes.Buffer

	ctx := zerolog.New(&logs).With().Logger().WithContext(t.Context())

	server := &GRPCServer{
		params: protoloc.NewWithBindings(nil, nil, []protoloc.ProxySourceBinding{{
			ProxyURL: "grpc+proxy://upstream.example:8080/nope?bearer=" + proxyToken,
			Sources:  []string{"grpc://source.example:9090?bearer=" + sourceToken},
		}}),
	}

	_, _, err := server.buildProxiesWithBindings(ctx, nil)

	require.NotContains(t, logs.String(), proxyToken)
	require.NotContains(t, logs.String(), sourceToken)
	require.Contains(t, logs.String(), "upstream.example:8080")
	require.Contains(t, logs.String(), "source.example:9090")

	require.Error(t, err)
	require.NotContains(t, err.Error(), proxyToken)
	require.Contains(t, err.Error(), "upstream.example:8080")
}
