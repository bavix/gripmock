package tls

import (
	"crypto/tls"
	"crypto/x509"
	"io/fs"
	"os"
	"strings"

	"github.com/cockroachdb/errors"
)

const (
	MinTLSVersion12 = "1.2"
	MinTLSVersion13 = "1.3"
)

var (
	ErrCertRequired    = errors.New("cert file is required when TLS is enabled")
	ErrKeyRequired     = errors.New("key file is required when TLS is enabled")
	ErrCARequired      = errors.New("CA file is required when client auth is enabled")
	ErrMinVersionValue = errors.New("unsupported TLS min version")
	ErrParseCertFailed = errors.New("failed to parse CA certificate")
	ErrFileNotFound    = errors.New("file does not exist")
	ErrPathIsDirectory = errors.New("path is a directory")
)

type TLSConfig struct {
	CertFile   string
	KeyFile    string
	ClientAuth bool
	CAFile     string
	MinVersion string
}

func (t TLSConfig) BuildTLSConfig() (*tls.Config, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}

	minVersion, err := parseMinVersion(t.MinVersion)
	if err != nil {
		return nil, err
	}

	cert, err := t.LoadCertificates()
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{ //nolint:gosec // TLS 1.2 is intentionally supported
		Certificates: []tls.Certificate{cert},
		MinVersion:   minVersion,
	}

	if t.ClientAuth {
		caPool, loadErr := t.LoadCA()
		if loadErr != nil {
			return nil, loadErr
		}

		tlsConfig.ClientCAs = caPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return tlsConfig, nil
}

func (t TLSConfig) IsEnabled() bool {
	return strings.TrimSpace(t.CertFile) != "" && strings.TrimSpace(t.KeyFile) != ""
}

func (t TLSConfig) LoadCertificates() (tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(t.CertFile, t.KeyFile)
	if err != nil {
		return tls.Certificate{}, errors.Wrap(err, "failed to load certificate pair")
	}

	return cert, nil
}

func (t TLSConfig) LoadCA() (*x509.CertPool, error) {
	if t.CAFile == "" {
		return nil, ErrCARequired
	}

	caPEM, err := os.ReadFile(t.CAFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read CA file")
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, ErrParseCertFailed
	}

	return pool, nil
}

func (t TLSConfig) Validate() error {
	if strings.TrimSpace(t.CertFile) == "" {
		return ErrCertRequired
	}

	if strings.TrimSpace(t.KeyFile) == "" {
		return ErrKeyRequired
	}

	if err := ensureFile(t.CertFile); err != nil {
		return errors.Wrap(err, "invalid cert file")
	}

	if err := ensureFile(t.KeyFile); err != nil {
		return errors.Wrap(err, "invalid key file")
	}

	if t.ClientAuth && strings.TrimSpace(t.CAFile) == "" {
		return ErrCARequired
	}

	if t.CAFile != "" {
		if err := ensureFile(t.CAFile); err != nil {
			return errors.Wrap(err, "invalid CA file")
		}
	}

	_, err := parseMinVersion(t.MinVersion)

	return err
}

func parseMinVersion(raw string) (uint16, error) {
	value := strings.TrimSpace(raw)
	if value == "" || value == MinTLSVersion12 {
		return tls.VersionTLS12, nil
	}

	if value == MinTLSVersion13 {
		return tls.VersionTLS13, nil
	}

	return 0, errors.Wrapf(
		ErrMinVersionValue,
		"%s: %q (supported: %s, %s)",
		ErrMinVersionValue.Error(),
		value,
		MinTLSVersion12,
		MinTLSVersion13,
	)
}

func ensureFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return errors.Wrapf(ErrFileNotFound, "%s: %s", ErrFileNotFound.Error(), path)
		}

		return err
	}

	if info.IsDir() {
		return errors.Wrapf(ErrPathIsDirectory, "%s: %s", ErrPathIsDirectory.Error(), path)
	}

	return nil
}
