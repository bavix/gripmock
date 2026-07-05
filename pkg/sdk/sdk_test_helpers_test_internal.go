package sdk

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/domain/protoset"
)

func sdkProtoPath(project string) string {
	return filepath.Join("..", "..", "examples", "projects", project, "service.proto")
}

func mustBuildFDS(t *testing.T, protoPath string) *descriptorpb.FileDescriptorSet {
	t.Helper()

	ctx := t.Context()
	fdsSlice, err := protoset.Build(ctx, nil, []string{protoPath}, nil)
	require.NoError(t, err)
	require.NotEmpty(t, fdsSlice)

	return fdsSlice[0]
}

// mustRunWithProto builds descriptors from protoPath and runs mock via Run(t, ...) (auto cleanup).
func mustRunWithProto(t *testing.T, protoPath string, opts ...Option) Mock {
	t.Helper()

	fds := mustBuildFDS(t, protoPath)
	allOpts := append([]Option{WithDescriptors(fds)}, opts...)
	mock, err := Run(t, allOpts...)
	require.NoError(t, err)
	require.NotNil(t, mock)

	return mock
}

// mustRunWithProtoAndReg returns mock and protodesc registry. Uses Run(t, ...) (auto cleanup).
func mustRunWithProtoAndReg(t *testing.T, protoPath string, opts ...Option) (Mock, *protoregistry.Files) {
	t.Helper()

	fds := mustBuildFDS(t, protoPath)
	allOpts := append([]Option{WithDescriptors(fds)}, opts...)
	mock, err := Run(t, allOpts...)
	require.NoError(t, err)
	require.NotNil(t, mock)

	reg, err := protodesc.NewFiles(fds)
	require.NoError(t, err)

	return mock, reg
}

func mustBuildRegistryFromProto(t *testing.T, protoPath string) *protoregistry.Files {
	t.Helper()

	fds := mustBuildFDS(t, protoPath)
	reg, err := protodesc.NewFiles(fds)
	require.NoError(t, err)

	return reg
}

func invokeGreeterSayHello(t *testing.T, conn *grpc.ClientConn, reg *protoregistry.Files, ctx context.Context, name string) *dynamicpb.Message {
	t.Helper()
	inDesc, err := reg.FindDescriptorByName("helloworld.HelloRequest")
	require.NoError(t, err)
	outDesc, err := reg.FindDescriptorByName("helloworld.HelloReply")
	require.NoError(t, err)

	in := dynamicpb.NewMessage(inDesc.(protoreflect.MessageDescriptor))
	fd := inDesc.(protoreflect.MessageDescriptor).Fields().ByName("name")
	in.Set(fd, protoreflect.ValueOfString(name))

	out := dynamicpb.NewMessage(outDesc.(protoreflect.MessageDescriptor))
	err = conn.Invoke(ctx, "/helloworld.Greeter/SayHello", in, out)
	require.NoError(t, err)
	return out
}

func getMessageField(t *testing.T, msg *dynamicpb.Message, field string) string {
	t.Helper()
	fd := msg.Descriptor().Fields().ByName(protoreflect.Name(field))
	require.NotNil(t, fd)
	return msg.Get(fd).String()
}

func createGreeterRequest(t *testing.T, reg *protoregistry.Files, name string) *dynamicpb.Message {
	t.Helper()
	inDesc, err := reg.FindDescriptorByName("helloworld.HelloRequest")
	require.NoError(t, err)
	in := dynamicpb.NewMessage(inDesc.(protoreflect.MessageDescriptor))
	fd := inDesc.(protoreflect.MessageDescriptor).Fields().ByName("name")
	in.Set(fd, protoreflect.ValueOfString(name))
	return in
}
