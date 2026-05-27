package proto_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/domain/proto"
)

func TestParseArgumentsWithBindings_SingleProxyWithSources(t *testing.T) {
	t.Parallel()

	args := []string{"-S", "a.proto", "-S", "b.proto", "grpc+proxy://up1:4111"}
	params := proto.ParseArgumentsWithBindings(args, nil, nil)

	require.True(t, params.HasProxyBindings())
	require.Len(t, params.ProxyBindings(), 1)
	require.Equal(t, "grpc+proxy://up1:4111", params.ProxyBindings()[0].ProxyURL)
	require.Equal(t, []string{"a.proto", "b.proto"}, params.ProxyBindings()[0].Sources)
}

func TestParseArgumentsWithBindings_MultipleProxiesWithDistinctSources(t *testing.T) {
	t.Parallel()

	args := []string{
		"-S", "a.proto",
		"grpc+proxy://up1:4111",
		"-S", "b.proto",
		"grpc+capture://up2:4222",
	}
	params := proto.ParseArgumentsWithBindings(args, nil, nil)

	require.True(t, params.HasProxyBindings())
	require.Len(t, params.ProxyBindings(), 2)

	require.Equal(t, "grpc+proxy://up1:4111", params.ProxyBindings()[0].ProxyURL)
	require.Equal(t, []string{"a.proto"}, params.ProxyBindings()[0].Sources)

	require.Equal(t, "grpc+capture://up2:4222", params.ProxyBindings()[1].ProxyURL)
	require.Equal(t, []string{"b.proto"}, params.ProxyBindings()[1].Sources)
}

func TestParseArgumentsWithBindings_MultipleSourcesForFirstProxy(t *testing.T) {
	t.Parallel()

	args := []string{
		"-S", "a.proto", "-S", "b.proto",
		"grpc+proxy://up1:4111",
		"grpc+proxy://up2:4222",
	}
	params := proto.ParseArgumentsWithBindings(args, nil, nil)

	require.True(t, params.HasProxyBindings())
	require.Len(t, params.ProxyBindings(), 2)

	require.Equal(t, "grpc+proxy://up1:4111", params.ProxyBindings()[0].ProxyURL)
	require.Equal(t, []string{"a.proto", "b.proto"}, params.ProxyBindings()[0].Sources)

	require.Equal(t, "grpc+proxy://up2:4222", params.ProxyBindings()[1].ProxyURL)
	require.Empty(t, params.ProxyBindings()[1].Sources)
}

func TestParseArgumentsWithBindings_ProxyWithoutSources(t *testing.T) {
	t.Parallel()

	args := []string{"grpc+proxy://up1:4111", "-S", "a.proto", "grpc+proxy://up2:4222"}
	params := proto.ParseArgumentsWithBindings(args, nil, nil)

	require.True(t, params.HasProxyBindings())
	require.Len(t, params.ProxyBindings(), 2)

	require.Equal(t, "grpc+proxy://up1:4111", params.ProxyBindings()[0].ProxyURL)
	require.Empty(t, params.ProxyBindings()[0].Sources)

	require.Equal(t, "grpc+proxy://up2:4222", params.ProxyBindings()[1].ProxyURL)
	require.Equal(t, []string{"a.proto"}, params.ProxyBindings()[1].Sources)
}

func TestParseArgumentsWithBindings_AllProxyModes(t *testing.T) {
	t.Parallel()

	args := []string{
		"-S", "a.proto", "grpc+proxy://up1:4111",
		"-S", "b.proto", "grpcs+capture://up2:4222",
		"-S", "c.proto", "grpc+replay://up3:4333",
	}
	params := proto.ParseArgumentsWithBindings(args, nil, nil)

	require.True(t, params.HasProxyBindings())
	require.Len(t, params.ProxyBindings(), 3)

	require.Equal(t, "grpc+proxy://up1:4111", params.ProxyBindings()[0].ProxyURL)
	require.Equal(t, "grpcs+capture://up2:4222", params.ProxyBindings()[1].ProxyURL)
	require.Equal(t, "grpc+replay://up3:4333", params.ProxyBindings()[2].ProxyURL)
}

func TestParseArgumentsWithBindings_SourceEqualsSyntax(t *testing.T) {
	t.Parallel()

	args := []string{"-S=a.proto", "--source=b.proto", "grpc+proxy://up1:4111"}
	params := proto.ParseArgumentsWithBindings(args, nil, nil)

	require.True(t, params.HasProxyBindings())
	require.Len(t, params.ProxyBindings(), 1)
	require.Equal(t, []string{"a.proto", "b.proto"}, params.ProxyBindings()[0].Sources)
}

func TestParseArgumentsWithBindings_MixedProtoPathsAndProxies(t *testing.T) {
	t.Parallel()

	args := []string{
		"examples/greeter.proto",
		"-S", "local.proto",
		"grpc+proxy://up1:4111",
		"examples/orders",
	}
	params := proto.ParseArgumentsWithBindings(args, nil, nil)

	require.True(t, params.HasProxyBindings())
	require.Equal(t, []string{"examples/greeter.proto", "examples/orders"}, params.ProtoPath())
	require.Len(t, params.ProxyBindings(), 1)
	require.Equal(t, []string{"local.proto"}, params.ProxyBindings()[0].Sources)
}

func TestParseArgumentsWithBindings_NoProxiesAllSources(t *testing.T) {
	t.Parallel()

	args := []string{"-S", "a.proto", "-S", "b.proto", "examples/greeter.proto"}
	params := proto.ParseArgumentsWithBindings(args, nil, nil)

	require.False(t, params.HasProxyBindings())
	require.Equal(t, []string{"a.proto", "b.proto"}, params.Sources())
	require.Equal(t, []string{"examples/greeter.proto"}, params.ProtoPath())
}

func TestParseArgumentsWithBindings_EmptyArgs(t *testing.T) {
	t.Parallel()

	params := proto.ParseArgumentsWithBindings([]string{}, nil, nil)

	require.False(t, params.HasProxyBindings())
	require.Empty(t, params.ProtoPath())
	require.Empty(t, params.Sources())
}

func TestParseArgumentsWithBindings_WithImports(t *testing.T) {
	t.Parallel()

	args := []string{"-S", "a.proto", "grpc+proxy://up1:4111"}
	imports := []string{"./proto", "./vendor"}
	params := proto.ParseArgumentsWithBindings(args, imports, nil)

	require.Equal(t, imports, params.Imports())
}

func TestParseArgumentsWithBindings_SecureGRPC(t *testing.T) {
	t.Parallel()

	args := []string{
		"-S", "a.proto", "grpcs+proxy://up1:4111",
		"-S", "b.proto", "grpcs+capture://up2:4222",
	}
	params := proto.ParseArgumentsWithBindings(args, nil, nil)

	require.True(t, params.HasProxyBindings())
	require.Len(t, params.ProxyBindings(), 2)

	require.Equal(t, "grpcs+proxy://up1:4111", params.ProxyBindings()[0].ProxyURL)
	require.Equal(t, []string{"a.proto"}, params.ProxyBindings()[0].Sources)

	require.Equal(t, "grpcs+capture://up2:4222", params.ProxyBindings()[1].ProxyURL)
	require.Equal(t, []string{"b.proto"}, params.ProxyBindings()[1].Sources)
}

func TestIsProxyURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"grpc proxy", "grpc+proxy://host:123", true},
		{"grpcs proxy", "grpcs+proxy://host:123", true},
		{"grpc capture", "grpc+capture://host:123", true},
		{"grpcs capture", "grpcs+capture://host:123", true},
		{"grpc replay", "grpc+replay://host:123", true},
		{"grpcs replay", "grpcs+replay://host:123", true},
		{"plain grpc", "grpc://host:123", false},
		{"plain grpcs", "grpcs://host:123", false},
		{"regular path", "examples/greeter.proto", false},
		{"http url", "http://example.com", false},
		{"file with plus", "file+name.proto", false},
		{"grpc without ://", "grpc+proxy", false},
		{"invalid mode", "grpc+invalid://host:123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, proto.IsProxyURL(tt.url))
		})
	}
}
