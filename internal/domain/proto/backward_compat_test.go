package proto_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/domain/proto"
)

// TestBackwardCompatibility_LegacyMode verifies that legacy mode (without per-proxy bindings)
// still works as before the per-proxy binding feature was added.
func TestBackwardCompatibility_LegacyMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      []string
		imports   []string
		wantProto []string
		wantSrc   []string
	}{
		{
			name:      "no sources",
			args:      []string{"a.proto", "b.proto"},
			imports:   []string{"./imports"},
			wantProto: []string{"a.proto", "b.proto"},
			wantSrc:   []string{},
		},
		{
			name:      "with sources",
			args:      []string{"-S", "service.proto", "common.proto"},
			imports:   []string{"./imports"},
			wantProto: []string{"common.proto"},
			wantSrc:   []string{"service.proto"},
		},
		{
			name:      "multiple sources",
			args:      []string{"-S", "a.proto", "-S", "b.proto", "common.proto"},
			imports:   []string{},
			wantProto: []string{"common.proto"},
			wantSrc:   []string{"a.proto", "b.proto"},
		},
		{
			name:      "sources only",
			args:      []string{"-S", "service.proto"},
			imports:   []string{},
			wantProto: []string{},
			wantSrc:   []string{"service.proto"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := proto.ParseArgumentsWithBindings(tt.args, tt.imports)

			require.False(t, result.HasProxyBindings(), "legacy mode should not have proxy bindings")
			require.Nil(t, result.ProxyBindings())
			require.Equal(t, tt.wantProto, result.ProtoPath())
			require.Equal(t, tt.wantSrc, result.Sources())
			require.Equal(t, tt.imports, result.Imports())
		})
	}
}

// TestBackwardCompatibility_OldProxyBehavior verifies that when using old-style
// global -S flags with a single proxy, they still work (legacy behavior).
func TestBackwardCompatibility_OldProxyBehavior(t *testing.T) {
	t.Parallel()

	// Old behavior: -S flags before proxy were global
	args := []string{"-S", "service.proto", "grpc+proxy://upstream:4111"}
	result := proto.ParseArgumentsWithBindings(args, nil)

	// This should now be treated as per-proxy binding
	require.True(t, result.HasProxyBindings())
	require.Len(t, result.ProxyBindings(), 1)
	require.Equal(t, "grpc+proxy://upstream:4111", result.ProxyBindings()[0].ProxyURL)
	require.Equal(t, []string{"service.proto"}, result.ProxyBindings()[0].Sources)
}

// TestBackwardCompatibility_MultipleProxiesOldWay verifies that old-style usage
// with multiple proxies gets new behavior (per-proxy binding).
func TestBackwardCompatibility_MultipleProxiesOldWay(t *testing.T) {
	t.Parallel()

	// Old way (that didn't work correctly): all -S flags were global
	args := []string{"-S", "a.proto", "-S", "b.proto", "grpc+proxy://up1:4111", "grpc+proxy://up2:4222"}
	result := proto.ParseArgumentsWithBindings(args, nil)

	// New behavior: -S flags before first proxy are bound to first proxy
	require.True(t, result.HasProxyBindings())
	require.Len(t, result.ProxyBindings(), 2)

	// First proxy gets the sources
	require.Equal(t, "grpc+proxy://up1:4111", result.ProxyBindings()[0].ProxyURL)
	require.Equal(t, []string{"a.proto", "b.proto"}, result.ProxyBindings()[0].Sources)

	// Second proxy has no sources (uses reflection)
	require.Equal(t, "grpc+proxy://up2:4222", result.ProxyBindings()[1].ProxyURL)
	require.Empty(t, result.ProxyBindings()[1].Sources)
}

// TestBackwardCompatibility_MixedProtoAndProxy verifies that proto files
// and proxy URLs can be mixed on command line.
func TestBackwardCompatibility_MixedProtoAndProxy(t *testing.T) {
	t.Parallel()

	args := []string{"common.proto", "-S", "service.proto", "grpc+proxy://upstream:4111"}
	result := proto.ParseArgumentsWithBindings(args, nil)

	// common.proto goes to protoPath (for local mock server)
	require.Equal(t, []string{"common.proto"}, result.ProtoPath())

	// Proxy binding
	require.True(t, result.HasProxyBindings())
	require.Len(t, result.ProxyBindings(), 1)
	require.Equal(t, []string{"service.proto"}, result.ProxyBindings()[0].Sources)
}

// TestBackwardCompatibility_NoProxyURLs verifies that without proxy URLs,
// everything falls back to legacy global mode.
func TestBackwardCompatibility_NoProxyURLs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{"regular proto files", []string{"a.proto", "b.proto"}},
		{"with -S flags", []string{"-S", "service.proto", "common.proto"}},
		{"directory paths", []string{"./examples", "./protos"}},
		{"grpc reflection (no +mode)", []string{"grpc://upstream:4111"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := proto.ParseArgumentsWithBindings(tt.args, nil)

			require.False(t, result.HasProxyBindings(), "should use legacy mode")
			require.Nil(t, result.ProxyBindings())
		})
	}
}

// TestBackwardCompatibility_EmptyArgs verifies empty arguments don't panic.
func TestBackwardCompatibility_EmptyArgs(t *testing.T) {
	t.Parallel()

	result := proto.ParseArgumentsWithBindings([]string{}, nil)

	require.NotNil(t, result)
	require.False(t, result.HasProxyBindings())
	require.Empty(t, result.ProtoPath())
	require.Empty(t, result.Sources())
	require.Empty(t, result.Imports())
}

// TestBackwardCompatibility_AllThreeProxyModes verifies all proxy modes work.
func TestBackwardCompatibility_AllThreeProxyModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		mode string
		url  string
	}{
		{"proxy", "grpc+proxy://upstream:4111"},
		{"capture", "grpc+capture://upstream:4222"},
		{"replay", "grpc+replay://upstream:4333"},
		{"proxy-tls", "grpcs+proxy://upstream:4444"},
		{"capture-tls", "grpcs+capture://upstream:4555"},
		{"replay-tls", "grpcs+replay://upstream:4666"},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			t.Parallel()

			result := proto.ParseArgumentsWithBindings([]string{tt.url}, nil)

			require.True(t, result.HasProxyBindings())
			require.Len(t, result.ProxyBindings(), 1)
			require.Equal(t, tt.url, result.ProxyBindings()[0].ProxyURL)
		})
	}
}
