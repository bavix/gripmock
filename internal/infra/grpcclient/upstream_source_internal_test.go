package grpcclient

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	protosetdom "github.com/bavix/gripmock/v3/internal/domain/protoset"
)

func TestUpstreamFromSourceCarriesEveryReflectField(t *testing.T) {
	t.Parallel()

	source := &protosetdom.Source{
		ReflectAddress:    "upstream:9090",
		ReflectTimeout:    7 * time.Second,
		ReflectTLS:        true,
		ReflectServerName: "sni.example.com",
		ReflectInsecure:   true,
		ReflectClientCert: "/certs/client.pem",
		ReflectClientKey:  "/certs/client-key.pem",
		ReflectCAFile:     "/certs/ca.pem",
		ReflectBearer:     "token",
	}

	upstream := UpstreamFromSource(source)

	require.Equal(t, Upstream{
		Address:            "upstream:9090",
		Timeout:            7 * time.Second,
		TLS:                true,
		ServerName:         "sni.example.com",
		InsecureSkipVerify: true,
		ClientCertFile:     "/certs/client.pem",
		ClientKeyFile:      "/certs/client-key.pem",
		CAFile:             "/certs/ca.pem",
		Bearer:             "token",
	}, upstream)
}

// A field added to Upstream without a line in UpstreamFromSource would be silently
// left at its zero value — for the TLS fields that means dialing without the client
// certificate the URL asked for.
func TestUpstreamFromSourceLeavesNoFieldUnset(t *testing.T) {
	t.Parallel()

	upstream := UpstreamFromSource(&protosetdom.Source{
		ReflectAddress:    "upstream:9090",
		ReflectTimeout:    time.Second,
		ReflectTLS:        true,
		ReflectServerName: "sni",
		ReflectInsecure:   true,
		ReflectClientCert: "cert",
		ReflectClientKey:  "key",
		ReflectCAFile:     "ca",
		ReflectBearer:     "bearer",
	})

	value := reflect.ValueOf(upstream)
	for i := range value.NumField() {
		require.Falsef(t, value.Field(i).IsZero(),
			"Upstream.%s stayed at its zero value", value.Type().Field(i).Name)
	}
}
