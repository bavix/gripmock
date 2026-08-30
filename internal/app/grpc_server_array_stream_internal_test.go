package app

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/types"
)

type mockArrayStreamServerStream struct {
	grpc.ServerStream

	ctx              context.Context //nolint:containedctx
	sentMessages     []*dynamicpb.Message
	sendMsgError     error
	recvMsgError     error
	contextCancelled bool
	headers          metadata.MD
	trailers         metadata.MD
}

func (m *mockArrayStreamServerStream) Context() context.Context {
	if m.contextCancelled {
		ctx, cancel := context.WithCancel(m.ctx)
		cancel()

		return ctx
	}

	return m.ctx
}

func (m *mockArrayStreamServerStream) SendMsg(msg any) error {
	if m.sendMsgError != nil {
		return m.sendMsgError
	}

	if dynamicMsg, ok := msg.(*dynamicpb.Message); ok {
		m.sentMessages = append(m.sentMessages, dynamicMsg)
	}

	return nil
}

func (m *mockArrayStreamServerStream) RecvMsg(msg any) error {
	return m.recvMsgError
}

func (m *mockArrayStreamServerStream) SetHeader(md metadata.MD) error {
	m.headers = metadata.Join(m.headers, md)

	return nil
}

func (m *mockArrayStreamServerStream) SendHeader(md metadata.MD) error {
	return nil
}

func (m *mockArrayStreamServerStream) SetTrailer(md metadata.MD) {
	m.trailers = metadata.Join(m.trailers, md)
}

func TestHandleArrayStreamDataSendsAllMessages(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	stream := &mockArrayStreamServerStream{
		ctx:          t.Context(),
		sentMessages: make([]*dynamicpb.Message, 0),
	}

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "TestService",
		Method:  "TestMethod",
		Output: stuber.Output{
			Stream: []any{
				map[string]any{"value": "message1"},
				map[string]any{"value": "message2"},
				map[string]any{"value": "message3"},
			},
		},
	}

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)

	_, err := mocker.handleArrayStreamData(stream, stub, stub.Output.Messages(), inputMsg, time.Now(), 1, false)
	require.NoError(t, err)
	require.Len(t, stream.sentMessages, 3)
}

func TestHandleArrayStreamDataEmptyStream(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	stream := &mockArrayStreamServerStream{
		ctx:          t.Context(),
		sentMessages: make([]*dynamicpb.Message, 0),
	}

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "TestService",
		Method:  "TestMethod",
		Output: stuber.Output{
			Stream: []any{},
		},
	}

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)
	_, err := mocker.handleArrayStreamData(stream, stub, stub.Output.Messages(), inputMsg, time.Now(), 1, false)
	require.NoError(t, err)
	require.Empty(t, stream.sentMessages)
}

func TestHandleArrayStreamDataWithDelay(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	stream := &mockArrayStreamServerStream{
		ctx:          t.Context(),
		sentMessages: make([]*dynamicpb.Message, 0),
	}

	delay := types.NewDelay(10 * time.Millisecond)
	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "TestService",
		Method:  "TestMethod",
		Output: stuber.Output{
			Delay: delay,
			Stream: []any{
				map[string]any{"value": "message1"},
				map[string]any{"value": "message2"},
			},
		},
	}

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)
	start := time.Now()
	_, err := mocker.handleArrayStreamData(stream, stub, stub.Output.Messages(), inputMsg, time.Now(), 1, false)
	duration := time.Since(start)

	require.NoError(t, err)
	require.Len(t, stream.sentMessages, 2)
	require.GreaterOrEqual(t, duration, time.Duration(delay.Static()))
}

func TestHandleArrayStreamDataWithTemplates(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	stream := &mockArrayStreamServerStream{
		ctx:          t.Context(),
		sentMessages: make([]*dynamicpb.Message, 0),
	}

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "TestService",
		Method:  "TestMethod",
		Output: stuber.Output{
			Stream: []any{
				map[string]any{"value": "{{.Request.value}}_0"},
				map[string]any{"value": "{{.Request.value}}_1"},
			},
		},
	}

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)

	_, err := mocker.handleArrayStreamData(stream, stub, stub.Output.Messages(), inputMsg, time.Now(), 1, false)
	require.NoError(t, err)
	require.Len(t, stream.sentMessages, 2)

	msg1Map := convertToMapWithDepth(stream.sentMessages[0], defaultConvertDepth)
	require.NotNil(t, msg1Map)
	require.NotNil(t, msg1Map)

	msg2Map := convertToMapWithDepth(stream.sentMessages[1], defaultConvertDepth)
	require.NotNil(t, msg2Map)
	require.NotNil(t, msg2Map)
}

func TestHandleArrayStreamDataInvalidDataType(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	stream := &mockArrayStreamServerStream{
		ctx:          t.Context(),
		sentMessages: make([]*dynamicpb.Message, 0),
	}

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "TestService",
		Method:  "TestMethod",
		Output: stuber.Output{
			Stream: []any{
				map[string]any{"value": "message1"},
				"invalid_string",
			},
		},
	}

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)
	_, err := mocker.handleArrayStreamData(stream, stub, stub.Output.Messages(), inputMsg, time.Now(), 1, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to convert response to dynamic message")
}

func TestHandleArrayStreamDataScalarElementsForWrapperOutput(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	mocker.outputDesc = (&wrapperspb.StringValue{}).ProtoReflect().Descriptor()

	stream := &mockArrayStreamServerStream{
		ctx:          t.Context(),
		sentMessages: make([]*dynamicpb.Message, 0),
	}

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "TestService",
		Method:  "TestMethod",
		Output:  stuber.Output{Stream: []any{"first", "second"}},
	}

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)

	sent, err := mocker.handleArrayStreamData(stream, stub, stub.Output.Messages(), inputMsg, time.Now(), 1, false)
	require.NoError(t, err)
	require.Equal(t, 2, sent)
	require.Len(t, stream.sentMessages, 2)
}

func TestHandleArrayStreamDataSendMsgError(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	expectedError := status.Error(codes.Internal, "send error")
	stream := &mockArrayStreamServerStream{
		ctx:          t.Context(),
		sentMessages: make([]*dynamicpb.Message, 0),
		sendMsgError: expectedError,
	}

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "TestService",
		Method:  "TestMethod",
		Output: stuber.Output{
			Stream: []any{
				map[string]any{"value": "message1"},
			},
		},
	}

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)
	_, err := mocker.handleArrayStreamData(stream, stub, stub.Output.Messages(), inputMsg, time.Now(), 1, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "send error")
}

func TestHandleArrayStreamDataContextCancelled(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	stream := &mockArrayStreamServerStream{
		ctx:              t.Context(),
		sentMessages:     make([]*dynamicpb.Message, 0),
		contextCancelled: true,
	}

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "TestService",
		Method:  "TestMethod",
		Output: stuber.Output{
			Stream: []any{
				map[string]any{"value": "message1"},
			},
		},
	}

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)
	_, err := mocker.handleArrayStreamData(stream, stub, stub.Output.Messages(), inputMsg, time.Now(), 1, false)
	require.Error(t, err)
	require.Equal(t, context.Canceled, err)
}

func TestHandleArrayStreamDataMessageIndexInTemplates(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	stream := &mockArrayStreamServerStream{
		ctx:          t.Context(),
		sentMessages: make([]*dynamicpb.Message, 0),
	}

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "TestService",
		Method:  "TestMethod",
		Output: stuber.Output{
			Stream: []any{
				map[string]any{"value": "index_{{.MessageIndex}}"},
				map[string]any{"value": "index_{{.MessageIndex}}"},
				map[string]any{"value": "index_{{.MessageIndex}}"},
			},
		},
	}

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)
	_, err := mocker.handleArrayStreamData(stream, stub, stub.Output.Messages(), inputMsg, time.Now(), 1, false)
	require.NoError(t, err)
	require.Len(t, stream.sentMessages, 3)

	for _, msg := range stream.sentMessages {
		msgMap := convertToMapWithDepth(msg, defaultConvertDepth)
		require.NotNil(t, msgMap)
	}
}

func TestHandleArrayStreamDataWithHeaders(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	md := metadata.New(map[string]string{"x-user-id": "123"})
	ctx := metadata.NewIncomingContext(t.Context(), md)
	stream := &mockArrayStreamServerStream{
		ctx:          ctx,
		sentMessages: make([]*dynamicpb.Message, 0),
	}

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "TestService",
		Method:  "TestMethod",
		Output: stuber.Output{
			Stream: []any{
				map[string]any{"value": "{{index .Headers \"x-user-id\"}}"},
			},
		},
	}

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)
	_, err := mocker.handleArrayStreamData(stream, stub, stub.Output.Messages(), inputMsg, time.Now(), 1, false)
	require.NoError(t, err)
	require.Len(t, stream.sentMessages, 1)

	msgMap := convertToMapWithDepth(stream.sentMessages[0], defaultConvertDepth)
	require.NotNil(t, msgMap)
}

func TestHandleArrayStreamDataEOFError(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	stream := &mockArrayStreamServerStream{
		ctx:          t.Context(),
		sentMessages: make([]*dynamicpb.Message, 0),
		sendMsgError: io.EOF,
	}

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "TestService",
		Method:  "TestMethod",
		Output: stuber.Output{
			Stream: []any{
				map[string]any{"value": "message1"},
			},
		},
	}

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)
	_, err := mocker.handleArrayStreamData(stream, stub, stub.Output.Messages(), inputMsg, time.Now(), 1, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to send response")
}

func TestHandleArrayStreamDataComputesDelayPerElement(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	stream := &mockArrayStreamServerStream{
		ctx:          t.Context(),
		sentMessages: make([]*dynamicpb.Message, 0),
	}

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "TestService",
		Method:  "TestMethod",
		Output: stuber.Output{
			Delay: "{{ .MessageIndex }}00ms",
			Stream: []any{
				map[string]any{"value": "message1"},
				map[string]any{"value": "message2"},
				map[string]any{
					stuber.GripMockKey: map[string]any{"delay": "{{ 30 }}ms"},
					"value":            "message3",
				},
			},
		},
	}

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)

	started := time.Now()

	_, err := mocker.handleArrayStreamData(stream, stub, stub.Output.Messages(), inputMsg, time.Now(), 1, false)
	elapsed := time.Since(started)

	require.NoError(t, err)
	require.Len(t, stream.sentMessages, 3)
	require.GreaterOrEqual(t, elapsed, 130*time.Millisecond)
	require.Less(t, elapsed, 400*time.Millisecond)
}

func TestNonArrayStreamDataRendersDataTemplate(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	stream := &mockArrayStreamServerStream{
		ctx:          t.Context(),
		sentMessages: make([]*dynamicpb.Message, 0),
		recvMsgError: io.EOF,
	}

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: "TestService",
		Method:  "TestMethod",
		Output: stuber.Output{
			Template: true,
			Data:     `{"value": "{{ .Request.value }}"}`,
		},
	}

	err := mocker.handleNonArrayStreamData(stream, stub, stub.Output,
		map[string]any{"value": "rendered"}, time.Now(), 0)
	require.NoError(t, err)
	require.Len(t, stream.sentMessages, 1)
	raw, err := protojson.Marshal(stream.sentMessages[0])
	require.NoError(t, err)
	require.JSONEq(t, `{"value":"rendered"}`, string(raw))
}
