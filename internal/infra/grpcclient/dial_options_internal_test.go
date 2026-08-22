package grpcclient

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDialOptionsWithoutTLS(t *testing.T) {
	t.Parallel()

	options, err := DialOptions(Upstream{Address: "upstream:4770", Timeout: time.Second})
	require.NoError(t, err)
	require.NotEmpty(t, options)
}

func TestDialOptionsLoadsClientCertificate(t *testing.T) {
	t.Parallel()

	certFile, keyFile := selfSignedPair(t)

	options, err := DialOptions(Upstream{
		Address:        "upstream:8443",
		Timeout:        time.Second,
		TLS:            true,
		ClientCertFile: certFile,
		ClientKeyFile:  keyFile,
	})
	require.NoError(t, err)
	require.NotEmpty(t, options)
}

func TestDialOptionsReportsUnreadableCertificate(t *testing.T) {
	t.Parallel()

	// A typo in a path must surface at startup, not as a handshake failure on the
	// first proxied call.
	_, err := DialOptions(Upstream{
		Address:        "upstream:8443",
		TLS:            true,
		ClientCertFile: filepath.Join(t.TempDir(), "missing.pem"),
		ClientKeyFile:  filepath.Join(t.TempDir(), "missing.key"),
	})
	require.Error(t, err)
}

func TestUpstreamUsesCustomTLS(t *testing.T) {
	t.Parallel()

	require.False(t, Upstream{TLS: true}.UsesCustomTLS())
	require.True(t, Upstream{TLS: true, CAFile: "/ca.pem"}.UsesCustomTLS())
	require.True(t, Upstream{TLS: true, ClientCertFile: "/c.pem"}.UsesCustomTLS())
}

func selfSignedPair(t *testing.T) (string, string) {
	t.Helper()

	dir := t.TempDir()
	certFile := filepath.Join(dir, "client.pem")
	keyFile := filepath.Join(dir, "client.key")

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	derKey, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(7),
		Subject:      pkix.Name{CommonName: "gripmock-client"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	derCert, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, key.Public(), key)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derCert}), 0o600))
	require.NoError(t, os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: derKey}), 0o600))

	return certFile, keyFile
}
