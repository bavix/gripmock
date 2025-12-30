package app

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/bavix/features"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
	"github.com/bavix/gripmock/v3/internal/infra/types"
	"github.com/bavix/gripmock/v3/pkg/plugintest"
)

const (
	testServiceName = "TestService"
	testMethodName  = "TestMethod"
)

// mockFullServerStream mocks grpc.ServerStream for testing with full functionality.
type mockFullServerStream struct {
	grpc.ServerStream

	ctx              context.Context //nolint:containedctx // Mock for testing
	sentMessages     []*dynamicpb.Message
	receivedMessages []*dynamicpb.Message
	sendMsgError     error
	recvMsgError     error
	recvMsgCount     int
	recvMsgLimit     int
	contextCancelled bool
	headers          metadata.MD
}

func (m *mockFullServerStream) Context() context.Context {
	if m.contextCancelled {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		return ctx
	}

	if m.ctx == nil {
		m.ctx = context.Background()
	}

	return m.ctx
}

func (m *mockFullServerStream) SendMsg(msg any) error {
	if m.sendMsgError != nil {
		return m.sendMsgError
	}

	if dynamicMsg, ok := msg.(*dynamicpb.Message); ok {
		m.sentMessages = append(m.sentMessages, dynamicMsg)
	}

	return nil
}

func (m *mockFullServerStream) RecvMsg(msg any) error {
	if m.recvMsgLimit > 0 && m.recvMsgCount >= m.recvMsgLimit {
		return io.EOF
	}

	if m.recvMsgError != nil {
		return m.recvMsgError
	}

	if dynamicMsg, ok := msg.(*dynamicpb.Message); ok && len(m.receivedMessages) > m.recvMsgCount {
		*dynamicMsg = *m.receivedMessages[m.recvMsgCount]
		m.recvMsgCount++

		return nil
	}

	return io.EOF
}

func (m *mockFullServerStream) SetHeader(md metadata.MD) error {
	m.headers = md

	return nil
}

func (m *mockFullServerStream) SendHeader(md metadata.MD) error {
	return nil
}

func (m *mockFullServerStream) SetTrailer(md metadata.MD) {
}

func createTestMockerForStream() *grpcMocker {
	structDesc := (&structpb.Struct{}).ProtoReflect().Descriptor()
	testRegistry := plugintest.NewRegistry()
	templateEngine := template.New(context.Background(), testRegistry)

	return &grpcMocker{
		budgerigar:     stuber.NewBudgerigar(features.New()),
		templateEngine: templateEngine,
		inputDesc:      structDesc,
		outputDesc:     structDesc,
	}
}

func TestHandleServerStream_WithArrayStream(t *testing.T) {
	t.Parallel()

	mocker := createTestMockerForStream()
	mocker.fullMethod = testServiceName + "/" + testMethodName
	mocker.fullServiceName = testServiceName
	mocker.serviceName = testServiceName
	mocker.methodName = testMethodName

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)
	stream := &mockFullServerStream{
		ctx:              context.Background(),
		sentMessages:     make([]*dynamicpb.Message, 0),
		receivedMessages: []*dynamicpb.Message{inputMsg},
		recvMsgLimit:     1,
	}

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: testServiceName,
		Method:  testMethodName,
		Input: stuber.InputData{
			Contains: map[string]any{},
		},
		Output: stuber.Output{
			Stream: []any{
				map[string]any{"message": "test1"},
				map[string]any{"message": "test2"},
			},
		},
	}

	mocker.budgerigar.PutMany(stub)

	err := mocker.handleServerStream(stream)
	require.NoError(t, err)
	assert.Len(t, stream.sentMessages, 2)
}

func TestHandleServerStream_WithNonArrayStream(t *testing.T) {
	t.Parallel()

	mocker := createTestMockerForStream()
	mocker.fullMethod = testServiceName + "/" + testMethodName
	mocker.fullServiceName = testServiceName
	mocker.serviceName = testServiceName
	mocker.methodName = testMethodName

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)
	stream := &mockFullServerStream{
		ctx:              context.Background(),
		sentMessages:     make([]*dynamicpb.Message, 0),
		receivedMessages: []*dynamicpb.Message{inputMsg},
		recvMsgLimit:     1,
	}

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: testServiceName,
		Method:  testMethodName,
		Input: stuber.InputData{
			Contains: map[string]any{},
		},
		Output: stuber.Output{
			Data: map[string]any{"message": "test"},
		},
	}

	mocker.budgerigar.PutMany(stub)

	err := mocker.handleServerStream(stream)
	require.NoError(t, err)
	assert.Len(t, stream.sentMessages, 1)
}

func TestHandleServerStream_WithHeaders(t *testing.T) {
	t.Parallel()

	mocker := createTestMockerForStream()
	mocker.fullMethod = testServiceName + "/" + testMethodName
	mocker.fullServiceName = testServiceName
	mocker.serviceName = testServiceName
	mocker.methodName = testMethodName

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
		"x-user": "testuser",
	}))
	stream := &mockFullServerStream{
		ctx:              ctx,
		sentMessages:     make([]*dynamicpb.Message, 0),
		receivedMessages: []*dynamicpb.Message{inputMsg},
		recvMsgLimit:     1,
	}

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: testServiceName,
		Method:  testMethodName,
		Input: stuber.InputData{
			Contains: map[string]any{},
		},
		Output: stuber.Output{
			Stream: []any{
				map[string]any{"message": "test"},
			},
			Headers: map[string]string{
				"x-response": "test",
			},
		},
	}

	mocker.budgerigar.PutMany(stub)

	err := mocker.handleServerStream(stream)
	require.NoError(t, err)
	assert.NotNil(t, stream.headers)
	assert.Equal(t, "test", stream.headers.Get("x-response")[0])
}

func TestHandleServerStream_WithError(t *testing.T) {
	t.Parallel()

	mocker := createTestMockerForStream()
	mocker.fullMethod = testServiceName + "/" + testMethodName
	mocker.fullServiceName = testServiceName
	mocker.serviceName = testServiceName
	mocker.methodName = testMethodName

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)
	stream := &mockFullServerStream{
		ctx:              context.Background(),
		sentMessages:     make([]*dynamicpb.Message, 0),
		receivedMessages: []*dynamicpb.Message{inputMsg},
		recvMsgLimit:     1,
	}

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: testServiceName,
		Method:  testMethodName,
		Input: stuber.InputData{
			Contains: map[string]any{},
		},
		Output: stuber.Output{
			Stream: []any{
				map[string]any{"message": "test"},
			},
			Error: "test error",
		},
	}

	mocker.budgerigar.PutMany(stub)

	err := mocker.handleServerStream(stream)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "test error")
}

func TestHandleServerStream_EOF(t *testing.T) {
	t.Parallel()

	mocker := createTestMockerForStream()
	stream := &mockFullServerStream{
		ctx:          context.Background(),
		sentMessages: make([]*dynamicpb.Message, 0),
		recvMsgError: io.EOF,
	}

	err := mocker.handleServerStream(stream)
	require.NoError(t, err)
}

func TestHandleServerStream_RecvError(t *testing.T) {
	t.Parallel()

	mocker := createTestMockerForStream()
	stream := &mockFullServerStream{
		ctx:          context.Background(),
		sentMessages: make([]*dynamicpb.Message, 0),
		recvMsgError: status.Error(codes.Internal, "receive error"),
	}

	err := mocker.handleServerStream(stream)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to receive message")
}

func TestHandleServerStream_NotFound(t *testing.T) {
	t.Parallel()

	mocker := createTestMockerForStream()
	mocker.fullMethod = testServiceName + "/" + testMethodName
	mocker.fullServiceName = testServiceName
	mocker.serviceName = testServiceName
	mocker.methodName = testMethodName

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)
	stream := &mockFullServerStream{
		ctx:              context.Background(),
		sentMessages:     make([]*dynamicpb.Message, 0),
		receivedMessages: []*dynamicpb.Message{inputMsg},
		recvMsgLimit:     1,
	}

	err := mocker.handleServerStream(stream)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find response")
}

func TestHandleNonArrayStreamData_SendsMessages(t *testing.T) {
	t.Parallel()

	mocker := createTestMockerForStream()
	stream := &mockFullServerStream{
		ctx:          context.Background(),
		sentMessages: make([]*dynamicpb.Message, 0),
		recvMsgLimit: 1,
	}

	stub := &stuber.Stub{
		ID: uuid.New(),
		Output: stuber.Output{
			Data: map[string]any{"message": "test"},
		},
	}

	err := mocker.handleNonArrayStreamData(stream, stub)
	require.NoError(t, err)
	assert.Len(t, stream.sentMessages, 1)
}

func TestHandleNonArrayStreamData_WithDelay(t *testing.T) {
	t.Parallel()

	mocker := createTestMockerForStream()
	stream := &mockFullServerStream{
		ctx:          context.Background(),
		sentMessages: make([]*dynamicpb.Message, 0),
		recvMsgLimit: 1,
	}

	stub := &stuber.Stub{
		ID: uuid.New(),
		Output: stuber.Output{
			Data:  map[string]any{"message": "test"},
			Delay: types.Duration(10 * time.Millisecond),
		},
	}

	start := time.Now()
	err := mocker.handleNonArrayStreamData(stream, stub)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, duration, 10*time.Millisecond)
}

func TestHandleNonArrayStreamData_WithTemplates(t *testing.T) {
	t.Parallel()

	mocker := createTestMockerForStream()
	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)
	stream := &mockFullServerStream{
		ctx:              context.Background(),
		sentMessages:     make([]*dynamicpb.Message, 0),
		receivedMessages: []*dynamicpb.Message{inputMsg},
		recvMsgLimit:     1,
	}

	stub := &stuber.Stub{
		ID: uuid.New(),
		Output: stuber.Output{
			Data: map[string]any{"message": "Hello, {{.Request.name}}!"},
		},
	}

	err := mocker.handleNonArrayStreamData(stream, stub)
	require.NoError(t, err)
	assert.Len(t, stream.sentMessages, 1)
}

func TestHandleNonArrayStreamData_ContextCancelled(t *testing.T) {
	t.Parallel()

	mocker := createTestMockerForStream()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stream := &mockFullServerStream{
		ctx:          ctx,
		sentMessages: make([]*dynamicpb.Message, 0),
		recvMsgLimit: 0,
	}

	stub := &stuber.Stub{
		ID: uuid.New(),
		Output: stuber.Output{
			Data: map[string]any{"message": "test"},
		},
	}

	err := mocker.handleNonArrayStreamData(stream, stub)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestHandleNonArrayStreamData_WithError(t *testing.T) {
	t.Parallel()

	mocker := createTestMockerForStream()
	stream := &mockFullServerStream{
		ctx:          context.Background(),
		sentMessages: make([]*dynamicpb.Message, 0),
		recvMsgLimit: 0,
	}

	stub := &stuber.Stub{
		ID: uuid.New(),
		Output: stuber.Output{
			Data:  map[string]any{"message": "test"},
			Error: "test error",
		},
	}

	err := mocker.handleNonArrayStreamData(stream, stub)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "test error")
}

func TestReceiveStreamMessage_Success(t *testing.T) {
	t.Parallel()

	msg := dynamicpb.NewMessage((&structpb.Struct{}).ProtoReflect().Descriptor())
	stream := &mockFullServerStream{
		ctx:              context.Background(),
		receivedMessages: []*dynamicpb.Message{msg},
		recvMsgLimit:     1,
	}

	err := receiveStreamMessage(stream, msg)
	require.NoError(t, err)
}

func TestReceiveStreamMessage_Error(t *testing.T) {
	t.Parallel()

	msg := dynamicpb.NewMessage((&structpb.Struct{}).ProtoReflect().Descriptor())
	stream := &mockFullServerStream{
		ctx:          context.Background(),
		recvMsgError: status.Error(codes.Internal, "receive error"),
	}

	err := receiveStreamMessage(stream, msg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to receive message")
}

func TestProcessHeaders_EmptyMetadata(t *testing.T) {
	t.Parallel()

	md := metadata.New(map[string]string{})
	result := processHeaders(md)
	assert.Nil(t, result)
}

func TestProcessHeaders_WithHeaders(t *testing.T) {
	t.Parallel()

	md := metadata.New(map[string]string{
		"x-user":     "testuser",
		"x-request":  "test",
		":authority": "localhost",
	})
	result := processHeaders(md)
	assert.NotNil(t, result)
	assert.Equal(t, "testuser", result["x-user"])
	assert.Equal(t, "test", result["x-request"])
	assert.NotContains(t, result, ":authority")
}

func TestProcessHeaders_ExcludedHeaders(t *testing.T) {
	t.Parallel()

	md := metadata.New(map[string]string{
		"content-type":         "application/grpc",
		"grpc-accept-encoding": "gzip",
		"user-agent":           "test",
		"accept-encoding":      "gzip",
		"x-custom":             "value",
	})
	result := processHeaders(md)
	assert.NotNil(t, result)
	assert.NotContains(t, result, "content-type")
	assert.NotContains(t, result, "grpc-accept-encoding")
	assert.NotContains(t, result, "user-agent")
	assert.NotContains(t, result, "accept-encoding")
	assert.Equal(t, "value", result["x-custom"])
}

func TestProcessHeaders_MultipleValues(t *testing.T) {
	t.Parallel()

	md := metadata.Pairs(
		"x-header", "value1",
		"x-header", "value2",
		"x-header", "value3",
	)
	result := processHeaders(md)
	assert.NotNil(t, result)
	assert.Equal(t, "value1;value2;value3", result["x-header"])
}
