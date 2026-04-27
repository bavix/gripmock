package sdk_test

import (
	"io"
	"testing"
	"time"

	chatpb "github.com/bavix/gripmock/v3/pkg/sdk/internal/examplefds/gen/examples/projects/chat"
	multiversepb "github.com/bavix/gripmock/v3/pkg/sdk/internal/examplefds/gen/examples/projects/multiverse"
	"github.com/bavix/gripmock/v3/pkg/sdk/internal/fdstest"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/pkg/sdk"
)

func mustRunMockWithDescriptors(t *testing.T, fds *descriptorpb.FileDescriptorSet) sdk.Mock {
	t.Helper()

	mock, err := sdk.Run(t, sdk.WithDescriptors(fds), sdk.WithHealthCheckTimeout(3*time.Second))
	require.NoError(t, err)

	return mock
}

func TestExmpEmbeddedChatStreaming(t *testing.T) {
	t.Parallel()

	fds := fdstest.DescriptorSetFromFile(chatpb.File_examples_projects_chat_service_proto)
	mock := mustRunMockWithDescriptors(t, fds)

	stubChatService(mock)

	client := chatpb.NewChatServiceClient(mock.Conn())
	ctx := t.Context()

	sendStream, err := client.SendMessage(ctx)
	require.NoError(t, err)
	require.NoError(t, sendStream.Send(&chatpb.Message{User: "alice", Text: "one"}))
	require.NoError(t, sendStream.Send(&chatpb.Message{User: "alice", Text: "two"}))
	clientResp, err := sendStream.CloseAndRecv()
	require.NoError(t, err)
	require.Equal(t, "stored", clientResp.GetMessage())
	require.True(t, clientResp.GetSuccess())

	serverStream, err := client.ReceiveMessages(ctx, &chatpb.UserRequest{User: "alice"})
	require.NoError(t, err)

	msg1, err := serverStream.Recv()
	require.NoError(t, err)
	msg2, err := serverStream.Recv()
	require.NoError(t, err)
	_, err = serverStream.Recv()
	require.ErrorIs(t, err, io.EOF)
	require.Equal(t, "hi", msg1.GetText())
	require.Equal(t, "welcome", msg2.GetText())

	bidi, err := client.Chat(ctx)
	require.NoError(t, err)
	require.NoError(t, bidi.Send(&chatpb.Message{User: "alice", Text: "hello"}))
	require.NoError(t, bidi.CloseSend())

	bidiMsg, err := bidi.Recv()
	require.NoError(t, err)
	require.Equal(t, "ack", bidiMsg.GetText())

	mock.Verify().Total(t, 3)
}

func TestExmpEmbeddedMultiverseUnaryAndStreams(t *testing.T) {
	t.Parallel()

	fds := fdstest.DescriptorSetFromFile(multiversepb.File_examples_projects_multiverse_service_proto)
	mock := mustRunMockWithDescriptors(t, fds)

	stubMultiverseService(mock)

	client := multiversepb.NewMultiverseServiceClient(mock.Conn())
	ctx := t.Context()

	var head metadata.MD
	pingResp, err := client.Ping(
		ctx,
		&multiversepb.PingRequest{Message: "ping", UserId: "u1"},
		grpc.Header(&head),
	)
	require.NoError(t, err)
	require.Equal(t, "pong", pingResp.GetReply())
	require.Equal(t, []string{"ok"}, head.Get("x-sdk"))

	upload, err := client.UploadData(ctx)
	require.NoError(t, err)
	require.NoError(t, upload.Send(&multiversepb.DataChunk{ChunkId: "c1", Sequence: 1}))
	require.NoError(t, upload.Send(&multiversepb.DataChunk{ChunkId: "c2", Sequence: 2}))
	uploadResp, err := upload.CloseAndRecv()
	require.NoError(t, err)
	require.Equal(t, "up-1", uploadResp.GetUploadId())
	require.True(t, uploadResp.GetSuccess())

	stream, err := client.StreamData(ctx, &multiversepb.StreamRequest{StreamId: "s1"})
	require.NoError(t, err)
	chunk1, err := stream.Recv()
	require.NoError(t, err)
	chunk2, err := stream.Recv()
	require.NoError(t, err)
	require.Equal(t, "c1", chunk1.GetChunkId())
	require.Equal(t, "c2", chunk2.GetChunkId())

	mock.Verify().Method(sdk.By(multiversepb.MultiverseService_Ping_FullMethodName)).Called(t, 1)
}

func TestExmpEmbeddedMergedGeneratedDescriptors(t *testing.T) {
	t.Parallel()

	chat := fdstest.DescriptorSetFromFile(chatpb.File_examples_projects_chat_service_proto)
	multiverse := fdstest.DescriptorSetFromFile(multiversepb.File_examples_projects_multiverse_service_proto)
	merged := &descriptorpb.FileDescriptorSet{File: append([]*descriptorpb.FileDescriptorProto{}, chat.GetFile()...)}
	merged.File = append(merged.File, multiverse.GetFile()...)

	mock, err := sdk.Run(t,
		sdk.WithDescriptors(merged),
		sdk.WithHealthCheckTimeout(3*time.Second),
	)
	require.NoError(t, err)

	mock.Stub(sdk.By(chatpb.ChatService_SendMessage_FullMethodName)).
		Reply(sdk.Data("success", true, "message", "ok")).
		Commit()

	mock.Stub(sdk.By(multiversepb.MultiverseService_Ping_FullMethodName)).
		Reply(sdk.Data("reply", "pong")).
		Commit()

	mock.Verify().Total(t, 0)
}

func stubChatService(mock sdk.Mock) {
	mock.Stub(sdk.By(chatpb.ChatService_SendMessage_FullMethodName)).
		Reply(sdk.Data("success", true, "message", "stored")).
		Commit()

	mock.Stub(sdk.By(chatpb.ChatService_ReceiveMessages_FullMethodName)).
		When(sdk.Equals("user", "alice")).
		ReplyStream(
			sdk.Data("user", "bob", "text", "hi"),
			sdk.Data("user", "carol", "text", "welcome"),
		).
		Commit()

	mock.Stub(sdk.By(chatpb.ChatService_Chat_FullMethodName)).
		ReplyStream(sdk.Data("user", "server", "text", "ack")).
		Commit()
}

func stubMultiverseService(mock sdk.Mock) {
	mock.Stub(sdk.By(multiversepb.MultiverseService_Ping_FullMethodName)).
		When(sdk.Equals("message", "ping")).
		Reply(sdk.Data("reply", "pong", "user_id", "u1")).
		ReplyHeaderPairs("x-sdk", "ok").
		Commit()

	mock.Stub(sdk.By(multiversepb.MultiverseService_UploadData_FullMethodName)).
		Reply(sdk.Data("upload_id", "up-1", "success", true, "total_chunks", 2)).
		Commit()

	mock.Stub(sdk.By(multiversepb.MultiverseService_StreamData_FullMethodName)).
		When(sdk.Equals("stream_id", "s1")).
		ReplyStream(
			sdk.Data("chunk_id", "c1", "sequence", 1),
			sdk.Data("chunk_id", "c2", "sequence", 2),
		).
		Commit()
}
