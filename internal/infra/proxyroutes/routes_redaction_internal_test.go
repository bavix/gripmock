package proxyroutes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

//nolint:gosec // fixture, not a real credential
const proxyURLWithToken = "grpc+proxy://upstream.example:8080/nope?bearer=s3cr3t-token"

func TestNewDoesNotLeakBearerOnParseFailure(t *testing.T) {
	t.Parallel()

	_, err := New(t.Context(), []string{proxyURLWithToken}, nil, nil)

	require.Error(t, err)
	require.NotContains(t, err.Error(), "s3cr3t-token")
	require.Contains(t, err.Error(), "failed to parse source")
	require.Contains(t, err.Error(), "upstream.example:8080")
}

func TestNewWithPerProxyDescriptorsDoesNotLeakBearerOnParseFailure(t *testing.T) {
	t.Parallel()

	_, err := NewWithPerProxyDescriptors(t.Context(),
		[]ProxyDescriptorBinding{{ProxyURL: proxyURLWithToken, Descriptors: nil}}, nil)

	require.Error(t, err)
	require.NotContains(t, err.Error(), "s3cr3t-token")
	require.Contains(t, err.Error(), "failed to parse proxy source")
	require.Contains(t, err.Error(), "upstream.example:8080")
}

func TestNewDoesNotLeakBearerWhenReflectionFails(t *testing.T) {
	t.Parallel()

	client := &fakeRemoteClient{sets: nil, calls: 0, failAll: true}

	_, err := New(t.Context(),
		[]string{"grpc+proxy://upstream.example:8080?bearer=s3cr3t-token"}, client, nil)

	require.Error(t, err)
	require.NotContains(t, err.Error(), "s3cr3t-token")
	require.Contains(t, err.Error(), "failed to fetch proxy descriptors")
	require.Contains(t, err.Error(), "upstream.example:8080")
}
