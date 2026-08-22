package grpcclient

import (
	protosetdom "github.com/bavix/gripmock/v3/internal/domain/protoset"
)

// UpstreamFromSource maps a parsed source URL onto the dial settings, so the
// reflection client and the proxy routes reach an upstream the same way.
func UpstreamFromSource(source *protosetdom.Source) Upstream {
	return Upstream{
		Address:            source.ReflectAddress,
		Timeout:            source.ReflectTimeout,
		TLS:                source.ReflectTLS,
		ServerName:         source.ReflectServerName,
		InsecureSkipVerify: source.ReflectInsecure,
		ClientCertFile:     source.ReflectClientCert,
		ClientKeyFile:      source.ReflectClientKey,
		CAFile:             source.ReflectCAFile,
		Bearer:             source.ReflectBearer,
	}
}
