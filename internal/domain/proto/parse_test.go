package proto_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/domain/proto"
)

func TestParseArgumentsWithBindings_SingleProxyWithSources(t *testing.T) {
	t.Parallel()

	args := []string{"-S", "a.proto", "-S", "b.proto", "grpc+proxy://up1:4111"}
	params := proto.ParseArgumentsWithBindings(args, args, nil, nil)

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
	params := proto.ParseArgumentsWithBindings(args, args, nil, nil)

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
	params := proto.ParseArgumentsWithBindings(args, args, nil, nil)

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
	params := proto.ParseArgumentsWithBindings(args, args, nil, nil)

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
	params := proto.ParseArgumentsWithBindings(args, args, nil, nil)

	require.True(t, params.HasProxyBindings())
	require.Len(t, params.ProxyBindings(), 3)

	require.Equal(t, "grpc+proxy://up1:4111", params.ProxyBindings()[0].ProxyURL)
	require.Equal(t, "grpcs+capture://up2:4222", params.ProxyBindings()[1].ProxyURL)
	require.Equal(t, "grpc+replay://up3:4333", params.ProxyBindings()[2].ProxyURL)
}

func TestParseArgumentsWithBindings_SourceEqualsSyntax(t *testing.T) {
	t.Parallel()

	args := []string{"-S=a.proto", "--source=b.proto", "grpc+proxy://up1:4111"}
	params := proto.ParseArgumentsWithBindings(args, args, nil, nil)

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
	params := proto.ParseArgumentsWithBindings(args, args, nil, nil)

	require.True(t, params.HasProxyBindings())
	require.Equal(t, []string{"examples/greeter.proto", "examples/orders"}, params.ProtoPath())
	require.Len(t, params.ProxyBindings(), 1)
	require.Equal(t, []string{"local.proto"}, params.ProxyBindings()[0].Sources)
}

func TestParseArgumentsWithBindings_NoProxiesAllSources(t *testing.T) {
	t.Parallel()

	args := []string{"-S", "a.proto", "-S", "b.proto", "examples/greeter.proto"}
	params := proto.ParseArgumentsWithBindings(args, args, nil, nil)

	require.False(t, params.HasProxyBindings())
	require.Equal(t, []string{"a.proto", "b.proto"}, params.Sources())
	require.Equal(t, []string{"examples/greeter.proto"}, params.ProtoPath())
}

func TestParseArgumentsWithBindings_EmptyArgs(t *testing.T) {
	t.Parallel()

	params := proto.ParseArgumentsWithBindings([]string{}, []string{}, nil, nil)

	require.False(t, params.HasProxyBindings())
	require.Empty(t, params.ProtoPath())
	require.Empty(t, params.Sources())
}

func TestParseArgumentsWithBindings_WithImports(t *testing.T) {
	t.Parallel()

	args := []string{"-S", "a.proto", "grpc+proxy://up1:4111"}
	imports := []string{"./proto", "./vendor"}
	params := proto.ParseArgumentsWithBindings(args, args, imports, nil)

	require.Equal(t, imports, params.Imports())
}

func TestParseArgumentsWithBindings_SecureGRPC(t *testing.T) {
	t.Parallel()

	args := []string{
		"-S", "a.proto", "grpcs+proxy://up1:4111",
		"-S", "b.proto", "grpcs+capture://up2:4222",
	}
	params := proto.ParseArgumentsWithBindings(args, args, nil, nil)

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

// Cobra hands the command its positional arguments with every -S already pulled
// out into a flag slice. Binding a source to a proxy is positional, so the parser
// has to read the untouched command line — the shape production actually passes.
func TestParseArgumentsWithBindings_CobraStrippedFlags(t *testing.T) {
	t.Parallel()

	rawArgs := []string{
		"-S", "orders.proto", "grpc+proxy://up1:4111",
		"-S", "users.proto", "grpc+replay://up2:4222",
	}
	// What Cobra leaves for the command, plus what it collected.
	positional := []string{"grpc+proxy://up1:4111", "grpc+replay://up2:4222"}
	cmdSources := []string{"orders.proto", "users.proto"}

	params := proto.ParseArgumentsWithBindings(positional, rawArgs, nil, cmdSources)

	require.True(t, params.HasProxyBindings())
	require.Len(t, params.ProxyBindings(), 2)
	require.Equal(t, []string{"orders.proto"}, params.ProxyBindings()[0].Sources)
	require.Equal(t, []string{"users.proto"}, params.ProxyBindings()[1].Sources)
}

// The documented case where the first proxy has no source of its own: it uses
// reflection, and the flag slice must not be dumped onto it.
func TestParseArgumentsWithBindings_SourceAfterFirstProxy(t *testing.T) {
	t.Parallel()

	rawArgs := []string{"grpc+proxy://up1:4111", "-S", "a.proto", "grpc+proxy://up2:4222"}
	positional := []string{"grpc+proxy://up1:4111", "grpc+proxy://up2:4222"}

	params := proto.ParseArgumentsWithBindings(positional, rawArgs, nil, []string{"a.proto"})

	require.Len(t, params.ProxyBindings(), 2)
	require.Empty(t, params.ProxyBindings()[0].Sources, "up1 was given no source and must use reflection")
	require.Equal(t, []string{"a.proto"}, params.ProxyBindings()[1].Sources)
}

// Protos named positionally stay protos; the proxy URL and the -S values do not
// leak into that list.
func TestParseArgumentsWithBindings_ProtoPathFromPositionalOnly(t *testing.T) {
	t.Parallel()

	params := proto.ParseArgumentsWithBindings(
		[]string{"local.proto", "grpc+proxy://up1:4111"},
		[]string{"local.proto", "-S", "a.proto", "grpc+proxy://up1:4111"},
		nil, []string{"a.proto"},
	)

	require.Equal(t, []string{"local.proto"}, params.ProtoPath())
	require.Equal(t, []string{"a.proto"}, params.ProxyBindings()[0].Sources)
}

// Cobra collects every -S into the flag slice, and the raw command line still
// shows the same flags. A source must not be built twice because it was seen
// from both sides.
func TestParseArgumentsWithBindings_SourceNotDuplicatedWithoutProxy(t *testing.T) {
	t.Parallel()

	params := proto.ParseArgumentsWithBindings(
		[]string{}, []string{"-S", "a.proto"}, nil, []string{"a.proto"},
	)

	require.Equal(t, []string{"a.proto"}, params.Sources())
	require.False(t, params.HasProxyBindings())
}

// pflag accepts the shorthand with its value attached. Written that way the
// source still binds to the proxy that follows it.
func TestParseArgumentsWithBindings_AttachedShorthand(t *testing.T) {
	t.Parallel()

	params := proto.ParseArgumentsWithBindings(
		[]string{"grpc+proxy://up1:4111", "grpc+proxy://up2:4222"},
		[]string{"-Sa.proto", "grpc+proxy://up1:4111", "-Sb.proto", "grpc+proxy://up2:4222"},
		nil, []string{"a.proto", "b.proto"},
	)

	require.Len(t, params.ProxyBindings(), 2)
	require.Equal(t, []string{"a.proto"}, params.ProxyBindings()[0].Sources)
	require.Equal(t, []string{"b.proto"}, params.ProxyBindings()[1].Sources)
}
