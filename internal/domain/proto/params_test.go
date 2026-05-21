package proto_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/domain/proto"
)

func TestArguments_New_Legacy(t *testing.T) {
	t.Parallel()

	protoPath := []string{"a.proto", "b.proto"}
	imports := []string{"./imports"}
	sources := []string{"c.proto", "d.proto"}

	args := proto.New(protoPath, imports, sources)

	require.NotNil(t, args)
	require.False(t, args.HasProxyBindings(), "legacy mode should not have proxy bindings")
	require.Nil(t, args.ProxyBindings(), "legacy mode should return nil for proxy bindings")
	require.Equal(t, protoPath, args.ProtoPath())
	require.Equal(t, imports, args.Imports())
	require.Equal(t, sources, args.Sources())
}

func TestArguments_NewWithBindings_Empty(t *testing.T) {
	t.Parallel()

	args := proto.NewWithBindings(nil, nil, nil)

	require.NotNil(t, args)
	require.True(t, args.HasProxyBindings(), "should have proxy bindings even if empty")
	require.Empty(t, args.ProxyBindings())
	require.Empty(t, args.ProtoPath())
	require.Empty(t, args.Imports())
	require.Empty(t, args.Sources(), "sources should be empty in binding mode")
}

func TestArguments_NewWithBindings_SingleProxy(t *testing.T) {
	t.Parallel()

	protoPath := []string{"common.proto"}
	imports := []string{"./imports"}
	bindings := []proto.ProxySourceBinding{
		{
			ProxyURL: "grpc+proxy://upstream1:4111",
			Sources:  []string{"service_a.proto"},
		},
	}

	args := proto.NewWithBindings(protoPath, imports, bindings)

	require.NotNil(t, args)
	require.True(t, args.HasProxyBindings())
	require.Equal(t, bindings, args.ProxyBindings())
	require.Equal(t, protoPath, args.ProtoPath())
	require.Equal(t, imports, args.Imports())
	require.Empty(t, args.Sources(), "sources should be empty in binding mode")
}

func TestArguments_NewWithBindings_MultipleProxies(t *testing.T) {
	t.Parallel()

	bindings := []proto.ProxySourceBinding{
		{
			ProxyURL: "grpc+proxy://upstream1:4111",
			Sources:  []string{"service_a.proto", "service_b.proto"},
		},
		{
			ProxyURL: "grpcs+capture://upstream2:4222",
			Sources:  []string{"service_c.proto"},
		},
		{
			ProxyURL: "grpc+replay://upstream3:4333",
			Sources:  []string{}, // Empty - should use reflection
		},
	}

	args := proto.NewWithBindings([]string{"common.proto"}, []string{"./imports"}, bindings)

	require.True(t, args.HasProxyBindings())
	require.Len(t, args.ProxyBindings(), 3)
	require.Equal(t, bindings, args.ProxyBindings())
}

func TestArguments_NewWithBindings_ProxyWithoutSources(t *testing.T) {
	t.Parallel()

	bindings := []proto.ProxySourceBinding{
		{
			ProxyURL: "grpc+proxy://upstream1:4111",
			Sources:  []string{}, // No sources - should trigger reflection
		},
	}

	args := proto.NewWithBindings(nil, nil, bindings)

	require.True(t, args.HasProxyBindings())
	require.Len(t, args.ProxyBindings(), 1)
	require.Empty(t, args.ProxyBindings()[0].Sources)
}

func TestArguments_ImmutabilityOfBindings(t *testing.T) {
	t.Parallel()

	originalBindings := []proto.ProxySourceBinding{
		{
			ProxyURL: "grpc+proxy://upstream1:4111",
			Sources:  []string{"a.proto"},
		},
	}

	args := proto.NewWithBindings(nil, nil, originalBindings)

	// Modify the original
	originalBindings[0].Sources = append(originalBindings[0].Sources, "b.proto")

	// Should not affect args
	retrieved := args.ProxyBindings()
	require.Len(t, retrieved[0].Sources, 1)
	require.Equal(t, "a.proto", retrieved[0].Sources[0])
}

func TestArguments_GettersReturnCorrectValues(t *testing.T) {
	t.Parallel()

	protoPath := []string{"path1.proto", "path2.proto"}
	imports := []string{"import1", "import2"}
	sources := []string{"source1.proto"}

	args := proto.New(protoPath, imports, sources)

	require.Equal(t, protoPath, args.ProtoPath())
	require.Equal(t, imports, args.Imports())
	require.Equal(t, sources, args.Sources())
}

func TestArguments_LegacyModeVsBindingMode(t *testing.T) {
	t.Parallel()

	legacyArgs := proto.New(
		[]string{"common.proto"},
		[]string{"./imports"},
		[]string{"service.proto"},
	)

	bindingArgs := proto.NewWithBindings(
		[]string{"common.proto"},
		[]string{"./imports"},
		[]proto.ProxySourceBinding{
			{ProxyURL: "grpc+proxy://up:4111", Sources: []string{"service.proto"}},
		},
	)

	// Legacy mode
	require.False(t, legacyArgs.HasProxyBindings())
	require.Nil(t, legacyArgs.ProxyBindings())
	require.Equal(t, []string{"service.proto"}, legacyArgs.Sources())

	// Binding mode
	require.True(t, bindingArgs.HasProxyBindings())
	require.NotNil(t, bindingArgs.ProxyBindings())
	require.Empty(t, bindingArgs.Sources())
}

func TestProxySourceBinding_Structure(t *testing.T) {
	t.Parallel()

	binding := proto.ProxySourceBinding{
		ProxyURL: "grpcs+capture://upstream:5000?timeout=10s",
		Sources:  []string{"service_a.proto", "service_b.proto"},
	}

	require.Equal(t, "grpcs+capture://upstream:5000?timeout=10s", binding.ProxyURL)
	require.Len(t, binding.Sources, 2)
	require.Contains(t, binding.Sources, "service_a.proto")
	require.Contains(t, binding.Sources, "service_b.proto")
}

func TestArguments_EmptySlicesNotNil(t *testing.T) {
	t.Parallel()

	args := proto.New(nil, nil, nil)

	// Ensure empty slices are returned, not nil
	require.NotNil(t, args.ProtoPath())
	require.NotNil(t, args.Imports())
	require.NotNil(t, args.Sources())
	require.Empty(t, args.ProtoPath())
	require.Empty(t, args.Imports())
	require.Empty(t, args.Sources())
}
