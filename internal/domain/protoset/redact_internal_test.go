package protoset

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedactURL(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		raw        string
		wantHas    []string
		wantHasNot []string
	}{
		"proxy bearer": {
			raw:        "grpc://upstream.example:8080?bearer=s3cr3t-token&timeout=5s",
			wantHas:    []string{"upstream.example:8080", "timeout=5s", "REDACTED"},
			wantHasNot: []string{"s3cr3t-token"},
		},
		"tls proxy bearer": {
			raw:        "grpcs://upstream.example:443?serverName=svc&bearer=abc",
			wantHas:    []string{"upstream.example:443", "serverName=svc"},
			wantHasNot: []string{"abc"},
		},
		"mixed case parameter": {
			raw:        "grpc://host:1?BeArEr=abc",
			wantHas:    nil,
			wantHasNot: []string{"abc"},
		},
		"token-shaped parameter": {
			raw:        "grpc://host:1?access_token=abc&apiKey=def&clientSecret=ghi",
			wantHas:    nil,
			wantHasNot: []string{"abc", "def", "ghi"},
		},
		"userinfo password": { //nolint:gosec // fixture, not a real credential
			raw:        "grpc://user:hunter2@host:1",
			wantHas:    []string{"host:1", "user"},
			wantHasNot: []string{"hunter2"},
		},
		"unparsable query keeps host": {
			raw:        "grpc://host:1?bearer=abc%zz",
			wantHas:    []string{"host:1"},
			wantHasNot: []string{"abc"},
		},
		"non credential query untouched": {
			raw:        "grpc://host:1?timeout=5s&insecureSkipVerify=true",
			wantHas:    []string{"timeout=5s", "insecureSkipVerify=true"},
			wantHasNot: nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := RedactURL(tc.raw)
			for _, want := range tc.wantHas {
				require.Contains(t, got, want)
			}

			for _, unwanted := range tc.wantHasNot {
				require.NotContains(t, got, unwanted)
			}
		})
	}
}

func TestRedactURLKeepsPlainPaths(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{
		"",
		"api/hello world.proto",
		"/abs/path/to/service.protoset",
		"buf.build/bavix/gripmock",
		"grpc://host:8080",
	} {
		require.Equal(t, raw, RedactURL(raw))
	}
}

func TestRedactURLRefusesToEchoUnparsableSource(t *testing.T) {
	t.Parallel()

	require.Equal(t, redactedSource, RedactURL("grpc://ho\x7fst?bearer=s3cr3t"))
}

func TestRedactURLs(t *testing.T) {
	t.Parallel()

	require.Nil(t, RedactURLs(nil))

	got := RedactURLs([]string{"api/hello.proto", "grpc://host:1?bearer=s3cr3t"})
	require.Equal(t, "api/hello.proto", got[0])
	require.NotContains(t, got[1], "s3cr3t")
}
