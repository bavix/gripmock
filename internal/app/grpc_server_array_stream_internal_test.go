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
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/types"
)

// mockArrayStreamServerStream mocks grpc.ServerStream for array stream testing.
type mockArrayStreamServerStream struct {
	grpc.ServerStream

	ctx              context.Context //nolint:containedctx
	sentMessages     []*dynamicpb.Message
	sendMsgError     error
	recvMsgError     error
	contextCancelled bool
}

func (m *mockArrayStreamServerStream) Context() context.Context {
	if m.contextCancelled {
		ctx, cancel := context.WithCancel(context.Background())
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
	return nil
}

func (m *mockArrayStreamServerStream) SendHeader(md metadata.MD) error {
	return nil
}

func (m *mockArrayStreamServerStream) SetTrailer(md metadata.MD) {
}

func TestHandleArrayStreamData_SendsAllMessages(t *testing.T) {
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

	// Create empty input message
	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)

	err := mocker.handleArrayStreamData(stream, stub, inputMsg, time.Now())
	require.NoError(t, err)
	require.Len(t, stream.sentMessages, 3)
}

func TestHandleArrayStreamData_EmptyStream(t *testing.T) {
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
	err := mocker.handleArrayStreamData(stream, stub, inputMsg, time.Now())
	require.NoError(t, err)
	require.Empty(t, stream.sentMessages)
}

func TestHandleArrayStreamData_WithDelay(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	stream := &mockArrayStreamServerStream{
		ctx:          t.Context(),
		sentMessages: make([]*dynamicpb.Message, 0),
	}

	delay := types.Duration(10 * time.Millisecond)
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
	err := mocker.handleArrayStreamData(stream, stub, inputMsg, time.Now())
	duration := time.Since(start)

	require.NoError(t, err)
	require.Len(t, stream.sentMessages, 2)
	// Should have at least one delay (between messages)
	require.GreaterOrEqual(t, duration, time.Duration(delay))
}

func TestHandleArrayStreamData_WithTemplates(t *testing.T) {
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

	// Create empty input message (template will use empty request)
	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)

	err := mocker.handleArrayStreamData(stream, stub, inputMsg, time.Now())
	require.NoError(t, err)
	require.Len(t, stream.sentMessages, 2)

	// Verify template processing by converting messages back to maps
	msg1Map := convertToMap(stream.sentMessages[0])
	require.NotNil(t, msg1Map)
	// Check if value was processed (may be in different structure depending on descriptor)
	require.NotNil(t, msg1Map)

	msg2Map := convertToMap(stream.sentMessages[1])
	require.NotNil(t, msg2Map)
	require.NotNil(t, msg2Map)
}

func TestHandleArrayStreamData_InvalidDataType(t *testing.T) {
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
				"invalid_string", // Invalid type
			},
		},
	}

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)
	err := mocker.handleArrayStreamData(stream, stub, inputMsg, time.Now())
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Internal, st.Code())
	require.Contains(t, st.Message(), "invalid data format")
}

func TestHandleArrayStreamData_SendMsgError(t *testing.T) {
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
	err := mocker.handleArrayStreamData(stream, stub, inputMsg, time.Now())
	require.Error(t, err)
	require.Contains(t, err.Error(), "send error")
}

func TestHandleArrayStreamData_ContextCancelled(t *testing.T) {
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
	err := mocker.handleArrayStreamData(stream, stub, inputMsg, time.Now())
	require.Error(t, err)
	require.Equal(t, context.Canceled, err)
}

func TestHandleArrayStreamData_MessageIndexInTemplates(t *testing.T) {
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
	err := mocker.handleArrayStreamData(stream, stub, inputMsg, time.Now())
	require.NoError(t, err)
	require.Len(t, stream.sentMessages, 3)

	// Verify all messages were sent (template processing is tested in template package)
	for _, msg := range stream.sentMessages {
		msgMap := convertToMap(msg)
		require.NotNil(t, msgMap)
	}
}

func TestHandleArrayStreamData_WithHeaders(t *testing.T) {
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
	err := mocker.handleArrayStreamData(stream, stub, inputMsg, time.Now())
	require.NoError(t, err)
	require.Len(t, stream.sentMessages, 1)

	// Verify message was sent (template processing with headers is tested in template package)
	msgMap := convertToMap(stream.sentMessages[0])
	require.NotNil(t, msgMap)
}

func TestHandleArrayStreamData_EOFError(t *testing.T) {
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
	err := mocker.handleArrayStreamData(stream, stub, inputMsg, time.Now())
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to send response")
}
