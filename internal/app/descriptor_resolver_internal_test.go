package app

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/internal/domain/protoset"
)

func TestFallbackResolver_UsesPrimaryThenFallback(t *testing.T) {
	t.Parallel()

	greeterFDS := mustSingleDescriptorSet(t, filepath.Join("..", "..", "examples", "projects", "greeter", "service.proto"))
	calculatorFDS := mustSingleDescriptorSet(t, filepath.Join("..", "..", "examples", "projects", "calculator", "service.proto"))

	primary, err := protodesc.NewFiles(greeterFDS)
	require.NoError(t, err)

	fallback, err := protodesc.NewFiles(calculatorFDS)
	require.NoError(t, err)

	resolver := &fallbackResolver{Primary: primary, Fallback: fallback}

	greeterDesc, err := resolver.FindDescriptorByName(protoreflect.FullName("helloworld.Greeter"))
	require.NoError(t, err)
	require.Equal(t, protoreflect.FullName("helloworld.Greeter"), greeterDesc.FullName())

	calcDesc, err := resolver.FindDescriptorByName(protoreflect.FullName("calculator.CalculatorService"))
	require.NoError(t, err)
	require.Equal(t, protoreflect.FullName("calculator.CalculatorService"), calcDesc.FullName())
}

func TestFallbackResolver_NotFoundWhenNoResolvers(t *testing.T) {
	t.Parallel()

	resolver := &fallbackResolver{}
	_, err := resolver.FindDescriptorByName(protoreflect.FullName("unknown.Service"))
	require.Error(t, err)
}

func mustSingleDescriptorSet(t *testing.T, protoPath string) *descriptorpb.FileDescriptorSet {
	t.Helper()

	fds, err := protoset.Build(t.Context(), nil, []string{protoPath})
	require.NoError(t, err)
	require.Len(t, fds, 1)

	return fds[0]
}
