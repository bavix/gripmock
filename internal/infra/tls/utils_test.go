package tls_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	infraTLS "github.com/bavix/gripmock/v3/internal/infra/tls"
)

func TestBuildTLSConfigDisabled(t *testing.T) {
	t.Parallel()

	cfg := infraTLS.TLSConfig{}
	require.False(t, cfg.IsEnabled())
}

func TestBuildTLSConfigMissingFiles(t *testing.T) {
	t.Parallel()

	t.Run("missing cert", func(t *testing.T) {
		t.Parallel()

		cfg := infraTLS.TLSConfig{KeyFile: "k.pem"}
		_, err := cfg.BuildTLSConfig()
		require.ErrorIs(t, err, infraTLS.ErrCertRequired)
	})

	t.Run("missing key", func(t *testing.T) {
		t.Parallel()

		cfg := infraTLS.TLSConfig{CertFile: "c.pem"}
		_, err := cfg.BuildTLSConfig()
		require.ErrorIs(t, err, infraTLS.ErrKeyRequired)
	})
}

func TestBuildTLSConfigUnsupportedMinVersion(t *testing.T) {
	t.Parallel()

	certFile, keyFile := mustSelfSignedServerCert(t)

	cfg := infraTLS.TLSConfig{
		CertFile:   certFile,
		KeyFile:    keyFile,
		MinVersion: "2.0",
	}

	_, err := cfg.BuildTLSConfig()
	require.ErrorIs(t, err, infraTLS.ErrMinVersionValue)
}

func TestBuildTLSConfigTLS12AndTLS13(t *testing.T) {
	t.Parallel()

	certFile, keyFile := mustSelfSignedServerCert(t)

	t.Run("tls12 default", func(t *testing.T) {
		t.Parallel()

		cfg := infraTLS.TLSConfig{CertFile: certFile, KeyFile: keyFile}
		tlsCfg, err := cfg.BuildTLSConfig()
		require.NoError(t, err)
		require.EqualValues(t, tls.VersionTLS12, tlsCfg.MinVersion)
	})

	t.Run("tls13", func(t *testing.T) {
		t.Parallel()

		cfg := infraTLS.TLSConfig{CertFile: certFile, KeyFile: keyFile, MinVersion: infraTLS.MinTLSVersion13}
		tlsCfg, err := cfg.BuildTLSConfig()
		require.NoError(t, err)
		require.EqualValues(t, tls.VersionTLS13, tlsCfg.MinVersion)
	})
}

func TestBuildTLSConfigClientAuth(t *testing.T) {
	t.Parallel()

	certFile, keyFile := mustSelfSignedServerCert(t)

	t.Run("missing ca", func(t *testing.T) {
		t.Parallel()

		cfg := infraTLS.TLSConfig{CertFile: certFile, KeyFile: keyFile, ClientAuth: true}
		_, err := cfg.BuildTLSConfig()
		require.ErrorIs(t, err, infraTLS.ErrCARequired)
	})

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		_, _, caFile := mustServerAndCA(t)
		cfg := infraTLS.TLSConfig{
			CertFile:   certFile,
			KeyFile:    keyFile,
			ClientAuth: true,
			CAFile:     caFile,
		}

		tlsCfg, err := cfg.BuildTLSConfig()
		require.NoError(t, err)
		require.Equal(t, tls.RequireAndVerifyClientCert, tlsCfg.ClientAuth)
		require.NotNil(t, tlsCfg.ClientCAs)
	})
}

func TestLoadCAInvalidPEM(t *testing.T) {
	t.Parallel()

	caFile := filepath.Join(t.TempDir(), "bad-ca.pem")
	require.NoError(t, os.WriteFile(caFile, []byte("not-a-pem"), 0o600))

	cfg := infraTLS.TLSConfig{CertFile: "unused", KeyFile: "unused", CAFile: caFile}
	_, err := cfg.LoadCA()
	require.ErrorIs(t, err, infraTLS.ErrParseCertFailed)
}

func TestValidatePathIsDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := infraTLS.TLSConfig{CertFile: dir, KeyFile: dir}
	err := cfg.Validate()
	require.Error(t, err)
}

func TestBuildClientTLSConfig(t *testing.T) {
	t.Parallel()

	t.Run("missing key with cert", func(t *testing.T) {
		t.Parallel()

		certFile, _ := mustSelfSignedServerCert(t)

		cfg := infraTLS.TLSConfig{CertFile: certFile}
		_, err := cfg.BuildClientTLSConfig("localhost:4770")
		require.ErrorIs(t, err, infraTLS.ErrKeyRequired)
	})

	t.Run("missing cert with key", func(t *testing.T) {
		t.Parallel()

		_, keyFile := mustSelfSignedServerCert(t)

		cfg := infraTLS.TLSConfig{KeyFile: keyFile}
		_, err := cfg.BuildClientTLSConfig("localhost:4770")
		require.ErrorIs(t, err, infraTLS.ErrCertRequired)
	})

	t.Run("sets defaults for wildcard host", func(t *testing.T) {
		t.Parallel()

		cfg := infraTLS.TLSConfig{}
		tlsCfg, err := cfg.BuildClientTLSConfig("0.0.0.0:4770")
		require.NoError(t, err)
		require.Equal(t, "localhost", tlsCfg.ServerName)
		require.EqualValues(t, tls.VersionTLS12, tlsCfg.MinVersion)
	})

	t.Run("loads client certificate pair", func(t *testing.T) {
		t.Parallel()

		certFile, keyFile := mustSelfSignedServerCert(t)

		cfg := infraTLS.TLSConfig{CertFile: certFile, KeyFile: keyFile, MinVersion: infraTLS.MinTLSVersion13}
		tlsCfg, err := cfg.BuildClientTLSConfig("localhost:4770")
		require.NoError(t, err)
		require.Len(t, tlsCfg.Certificates, 1)
		require.EqualValues(t, tls.VersionTLS13, tlsCfg.MinVersion)
	})

	t.Run("fails on invalid CA", func(t *testing.T) {
		t.Parallel()

		caFile := filepath.Join(t.TempDir(), "bad-ca.pem")
		require.NoError(t, os.WriteFile(caFile, []byte("not-a-pem"), 0o600))

		cfg := infraTLS.TLSConfig{CAFile: caFile}
		_, err := cfg.BuildClientTLSConfig("localhost:4770")
		require.ErrorIs(t, err, infraTLS.ErrParseCertFailed)
	})
}

func TestTLSConfigIsClientEnabled(t *testing.T) {
	t.Parallel()

	require.False(t, (infraTLS.TLSConfig{}).IsClientEnabled())
	require.True(t, (infraTLS.TLSConfig{CAFile: "ca.crt"}).IsClientEnabled())
	require.True(t, (infraTLS.TLSConfig{CertFile: "cert.crt", KeyFile: "key.key"}).IsClientEnabled())
}

func mustSelfSignedServerCert(t *testing.T) (string, string) {
	t.Helper()

	tmp := t.TempDir()
	certFile := filepath.Join(tmp, "server.crt")
	keyFile := filepath.Join(tmp, "server.key")

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	derKey, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(10),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
	}

	derCert, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, key.Public(), key)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derCert}), 0o600))
	require.NoError(t, os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: derKey}), 0o600))

	return certFile, keyFile
}

func mustServerAndCA(t *testing.T) (string, string, string) {
	t.Helper()

	tmp := t.TempDir()
	caFile := filepath.Join(tmp, "ca.crt")
	serverFile := filepath.Join(tmp, "server.crt")
	serverKeyFile := filepath.Join(tmp, "server.key")

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(20),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, caKey.Public(), caKey)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(caFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}), 0o600))

	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	serverKeyDER, err := x509.MarshalECPrivateKey(serverKey)
	require.NoError(t, err)

	serverTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(21),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
	}

	serverDER, err := x509.CreateCertificate(rand.Reader, serverTmpl, caTmpl, serverKey.Public(), caKey)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(serverFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverDER}), 0o600))
	require.NoError(t, os.WriteFile(serverKeyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: serverKeyDER}), 0o600))

	return serverFile, serverKeyFile, caFile
}
