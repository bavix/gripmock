package app

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/bavix/features"
	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	"github.com/bavix/gripmock/v3/internal/domain/protoset"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

//nolint:paralleltest
func TestBuildFromDescriptorSet_Greeter(t *testing.T) {
	ctx := t.Context()
	protoPath := filepath.Join("..", "..", "examples", "projects", "greeter", "service.proto")
	fdsSlice, err := protoset.Build(ctx, nil, []string{protoPath})
	require.NoError(t, err)
	require.NotEmpty(t, fdsSlice)

	budgerigar := stuber.NewBudgerigar(features.New())
	waiter := NewInstantExtender()

	server, err := BuildFromDescriptorSet(ctx, fdsSlice[0], budgerigar, waiter, nil)
	require.NoError(t, err)
	require.NotNil(t, server)

	defer server.GracefulStop()
}

//nolint:paralleltest
func TestGRPCServerBuild_WithoutStartupDescriptors(t *testing.T) {
	ctx := t.Context()

	server := NewGRPCServer(
		"tcp",
		":0",
		nil,
		stuber.NewBudgerigar(features.New()),
		NewInstantExtender(),
		nil,
		descriptors.NewRegistry(),
	)

	grpcServer, err := server.Build(ctx)
	require.NoError(t, err)
	require.NotNil(t, grpcServer)

	defer grpcServer.GracefulStop()
}

//nolint:paralleltest
func TestGRPCServerFindMethodDescriptor_FromDynamicRegistry(t *testing.T) {
	ctx := t.Context()
	protoPath := filepath.Join("..", "..", "examples", "projects", "greeter", "service.proto")
	fdsSlice, err := protoset.Build(ctx, nil, []string{protoPath})
	require.NoError(t, err)

	files, err := protodesc.NewFiles(fdsSlice[0])
	require.NoError(t, err)

	registry := descriptors.NewRegistry()

	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		registry.Register(fd)

		return true
	})

	server := NewGRPCServer(
		"tcp",
		":0",
		nil,
		stuber.NewBudgerigar(features.New()),
		NewInstantExtender(),
		nil,
		registry,
	)

	method, err := server.findMethodDescriptor("helloworld.Greeter", "SayHello")
	require.NoError(t, err)
	require.Equal(t, "helloworld.HelloRequest", string(method.Input().FullName()))
	require.Equal(t, "helloworld.HelloReply", string(method.Output().FullName()))
}
