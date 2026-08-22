package protoset

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProxySourceCarriesClientCertificate(t *testing.T) {
	t.Parallel()

	handler := &ProxyHandler{}

	source, err := handler.Parse("grpcs+proxy://upstream:8443?clientCert=/c.pem&clientKey=/c.key&caFile=/ca.pem")
	require.NoError(t, err)
	require.Equal(t, "/c.pem", source.ReflectClientCert)
	require.Equal(t, "/c.key", source.ReflectClientKey)
	require.Equal(t, "/ca.pem", source.ReflectCAFile)
	require.True(t, source.ReflectTLS)
}

func TestReflectSourceCarriesClientCertificate(t *testing.T) {
	t.Parallel()

	handler := &GRPCHandler{}

	source, err := handler.Parse("grpcs://upstream:8443?clientCert=/c.pem&clientKey=/c.key")
	require.NoError(t, err)
	require.Equal(t, "/c.pem", source.ReflectClientCert)
	require.Equal(t, "/c.key", source.ReflectClientKey)
}

func TestUpstreamTLSFilesRejectHalfAPair(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{
		"grpcs+proxy://upstream:8443?clientCert=/c.pem",
		"grpcs+proxy://upstream:8443?clientKey=/c.key",
	} {
		_, err := (&ProxyHandler{}).Parse(raw)
		require.ErrorIs(t, err, errClientCertPairIncomplete, raw)
	}
}

func TestUpstreamTLSFilesRejectPlaintextScheme(t *testing.T) {
	t.Parallel()

	// A plaintext upstream that quietly ignored a client certificate would look
	// authenticated without being it.
	_, err := (&ProxyHandler{}).Parse("grpc+proxy://upstream:8443?clientCert=/c.pem&clientKey=/c.key")
	require.ErrorIs(t, err, errClientTLSWithoutTLS)

	_, err = (&GRPCHandler{}).Parse("grpc://upstream:8443?caFile=/ca.pem")
	require.ErrorIs(t, err, errClientTLSWithoutTLS)
}

func TestEachUpstreamKeepsItsOwnCertificates(t *testing.T) {
	t.Parallel()

	handler := &ProxyHandler{}

	first, err := handler.Parse("grpcs+proxy://a:8443?clientCert=/a.pem&clientKey=/a.key&caFile=/ca-a.pem")
	require.NoError(t, err)

	second, err := handler.Parse("grpcs+proxy://b:8443?clientCert=/b.pem&clientKey=/b.key&caFile=/ca-b.pem")
	require.NoError(t, err)

	// Two upstreams on one command line are a normal setup, and they rarely share
	// a PKI: the material must stay attached to its own URL.
	require.Equal(t, "/a.pem", first.ReflectClientCert)
	require.Equal(t, "/ca-a.pem", first.ReflectCAFile)
	require.Equal(t, "/b.pem", second.ReflectClientCert)
	require.Equal(t, "/ca-b.pem", second.ReflectCAFile)
}
