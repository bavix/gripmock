package protoset

import (
	"net/url"

	"github.com/cockroachdb/errors"
)

var (
	errClientCertPairIncomplete = errors.New("upstream mTLS needs both clientCert and clientKey")
	errClientTLSWithoutTLS      = errors.New("clientCert, clientKey and caFile require a TLS scheme (grpcs)")
)

// upstreamTLSFiles carries the certificate material a URL declares for reaching
// an upstream: the client pair for mTLS and the CA that signs the upstream.
type upstreamTLSFiles struct {
	ClientCert string
	ClientKey  string
	CAFile     string
}

// parseUpstreamTLSFiles reads clientCert/clientKey/caFile from the query. The
// pair must be complete, and none of them make sense without TLS — a plaintext
// upstream that silently ignored a client certificate would look secure and not be.
func parseUpstreamTLSFiles(query url.Values, tlsEnabled bool) (upstreamTLSFiles, error) {
	files := upstreamTLSFiles{
		ClientCert: query.Get("clientCert"),
		ClientKey:  query.Get("clientKey"),
		CAFile:     query.Get("caFile"),
	}

	if (files.ClientCert == "") != (files.ClientKey == "") {
		return upstreamTLSFiles{}, errClientCertPairIncomplete
	}

	if !tlsEnabled && (files.ClientCert != "" || files.CAFile != "") {
		return upstreamTLSFiles{}, errClientTLSWithoutTLS
	}

	return files, nil
}
