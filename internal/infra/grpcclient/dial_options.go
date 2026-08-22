package grpcclient

import (
	"crypto/tls"
	"time"

	"github.com/cockroachdb/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	infraTLS "github.com/bavix/gripmock/v3/internal/infra/tls"
)

const dialOptionCapacity = 5

// Upstream describes how to reach an upstream gRPC server: transport security,
// who we present ourselves as, and the per-call budget.
type Upstream struct {
	Address string
	Timeout time.Duration

	TLS                bool
	ServerName         string
	InsecureSkipVerify bool

	// Client certificate presented to an upstream that asks for one (mTLS), and
	// the CA that signs the upstream certificate.
	ClientCertFile string
	ClientKeyFile  string
	CAFile         string

	Bearer string
}

// UsesCustomTLS reports whether the upstream needs more than the system trust
// store: a private CA, a client certificate, or both.
func (u Upstream) UsesCustomTLS() bool {
	return u.CAFile != "" || u.ClientCertFile != "" || u.ClientKeyFile != ""
}

// DialOptions builds the dial options for an upstream. It fails when the TLS
// material cannot be loaded, so a typo in a certificate path surfaces at startup
// rather than as a handshake error on the first proxied call.
func DialOptions(upstream Upstream) ([]grpc.DialOption, error) {
	options := make([]grpc.DialOption, 0, dialOptionCapacity)

	if upstream.TLS {
		tlsConfig, err := upstreamTLSConfig(upstream)
		if err != nil {
			return nil, err
		}

		options = append(options, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		options = append(options, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	unaryInterceptors := []grpc.UnaryClientInterceptor{UnaryTimeoutInterceptor(upstream.Timeout)}
	streamInterceptors := []grpc.StreamClientInterceptor{StreamTimeoutInterceptor(upstream.Timeout)}

	if upstream.Bearer != "" {
		unaryInterceptors = append(unaryInterceptors, UnaryBearerInterceptor(upstream.Bearer))
		streamInterceptors = append(streamInterceptors, StreamBearerInterceptor(upstream.Bearer))
	}

	options = append(options,
		grpc.WithChainUnaryInterceptor(unaryInterceptors...),
		grpc.WithChainStreamInterceptor(streamInterceptors...),
	)

	return options, nil
}

func upstreamTLSConfig(upstream Upstream) (*tls.Config, error) {
	if !upstream.UsesCustomTLS() {
		cfg := &tls.Config{MinVersion: tls.VersionTLS12}
		applyTLSOverrides(cfg, upstream)

		return cfg, nil
	}

	cfg, err := infraTLS.TLSConfig{
		CertFile: upstream.ClientCertFile,
		KeyFile:  upstream.ClientKeyFile,
		CAFile:   upstream.CAFile,
	}.BuildClientTLSConfig(upstream.Address)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build upstream TLS config")
	}

	applyTLSOverrides(cfg, upstream)

	return cfg, nil
}

// applyTLSOverrides puts the explicit SNI and the skip-verify switch on top of a
// built config, so both plain and certificate-backed upstreams honour them.
func applyTLSOverrides(cfg *tls.Config, upstream Upstream) {
	if upstream.ServerName != "" {
		cfg.ServerName = upstream.ServerName
	}

	cfg.InsecureSkipVerify = upstream.InsecureSkipVerify
}
