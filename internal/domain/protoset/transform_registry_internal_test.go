package protoset

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

const clashProbeFile = "gripmock_registry_clash_probe.proto"

func probeDescriptor(t *testing.T, serviceName string) *descriptorpb.FileDescriptorProto {
	t.Helper()

	return &descriptorpb.FileDescriptorProto{
		Name:    new(clashProbeFile),
		Package: new("gripmock.clashprobe"),
		Syntax:  new("proto3"),
		Service: []*descriptorpb.ServiceDescriptorProto{{Name: new(serviceName)}},
	}
}

//nolint:paralleltest
func TestRegisteredFileState(t *testing.T) {
	require.Equal(t, fileNotRegistered, registeredFileState(clashProbeFile, probeDescriptor(t, "First")),
		"probe name must be unused before the test registers it")

	protoRegistryMu.Lock()

	file, err := protodesc.NewFile(probeDescriptor(t, "First"), protoregistry.GlobalFiles)
	require.NoError(t, err)
	require.NoError(t, protoregistry.GlobalFiles.RegisterFile(file))

	protoRegistryMu.Unlock()

	require.Equal(t, fileRegisteredIdentical, registeredFileState(clashProbeFile, probeDescriptor(t, "First")),
		"the identical file arriving twice is routine, not a clash")

	require.Equal(t, fileRegisteredDifferent, registeredFileState(clashProbeFile, probeDescriptor(t, "Second")),
		"a different file under the same name costs the caller its services")
}
