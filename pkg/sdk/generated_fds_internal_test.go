package sdk

import (
	"testing"

	chatpb "github.com/bavix/gripmock/v3/pkg/sdk/internal/examplefds/gen/examples/projects/chat"
	multiversepb "github.com/bavix/gripmock/v3/pkg/sdk/internal/examplefds/gen/examples/projects/multiverse"
	"github.com/bavix/gripmock/v3/pkg/sdk/internal/fdstest"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestGeneratedDescriptorsCoverageScenarios(t *testing.T) {
	t.Parallel()

	// Arrange
	chat := fdstest.DescriptorSetFromFile(chatpb.File_examples_projects_chat_service_proto)
	multiverse := fdstest.DescriptorSetFromFile(multiversepb.File_examples_projects_multiverse_service_proto)

	// Assert: dedicated streaming fixture
	require.True(t, hasMethod(chat, "chat.ChatService", "SendMessage", true, false))
	require.True(t, hasMethod(chat, "chat.ChatService", "ReceiveMessages", false, true))
	require.True(t, hasMethod(chat, "chat.ChatService", "Chat", true, true))

	// Assert: unary + all streaming modes in one project
	require.True(t, hasMethod(multiverse, "multiverse.v1.MultiverseService", "Ping", false, false))
	require.True(t, hasMethod(multiverse, "multiverse.v1.MultiverseService", "UploadData", true, false))
	require.True(t, hasMethod(multiverse, "multiverse.v1.MultiverseService", "StreamData", false, true))
	require.True(t, hasMethod(multiverse, "multiverse.v1.MultiverseService", "Chat", true, true))

	// Assert: WKT import presence (google/protobuf/timestamp.proto)
	require.True(t, hasFile(multiverse, "google/protobuf/timestamp.proto"))
}

func hasFile(fds *descriptorpb.FileDescriptorSet, name string) bool {
	for _, file := range fds.GetFile() {
		if file.GetName() == name {
			return true
		}
	}

	return false
}

func hasMethod(fds *descriptorpb.FileDescriptorSet, service, method string, clientStreaming, serverStreaming bool) bool {
	for _, file := range fds.GetFile() {
		pkg := file.GetPackage()
		for _, svc := range file.GetService() {
			svcName := svc.GetName()
			if pkg != "" {
				svcName = pkg + "." + svcName
			}
			if svcName != service {
				continue
			}

			for _, m := range svc.GetMethod() {
				if m.GetName() == method && m.GetClientStreaming() == clientStreaming && m.GetServerStreaming() == serverStreaming {
					return true
				}
			}
		}
	}

	return false
}
