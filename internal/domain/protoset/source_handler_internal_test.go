package protoset

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSourceRejectsUnknownScheme(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{
		"grpc+captur://localhost:4770",
		"http+proxy://localhost:4770",
		"https://example.com/descriptors",
	} {
		t.Run(raw, func(t *testing.T) {
			t.Parallel()

			_, err := ParseSource(raw)
			require.ErrorIs(t, err, errUnknownSourceScheme)
		})
	}
}

func TestParseSourceAcceptsSupportedForms(t *testing.T) {
	t.Parallel()

	for name, testCase := range map[string]struct {
		raw       string
		wantType  SourceType
		wantProxy string
	}{
		"reflection":         {"grpc://localhost:4770", SourceReflect, ""},
		"reflection tls":     {"grpcs://localhost:4770", SourceReflect, ""},
		"proxy mode":         {"grpc+proxy://localhost:4770", SourceProxy, "proxy"},
		"capture mode":       {"grpc+capture://localhost:4770", SourceProxy, "capture"},
		"replay mode tls":    {"grpcs+replay://localhost:4770", SourceProxy, "replay"},
		"plain proto path":   {"examples/service.proto", SourceProto, ""},
		"descriptor set":     {"bundle.protoset", SourceDescriptor, ""},
		"descriptor over ht": {"https://example.com/bundle.protoset", SourceDescriptor, ""},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			source, err := ParseSource(testCase.raw)
			require.NoError(t, err)
			require.Equal(t, testCase.wantType, source.Type)
			require.Equal(t, testCase.wantProxy, source.ProxyMode)
		})
	}
}
