package app

import (
	"context"
	"errors"
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
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/types"
)

const (
	testServiceName = "TestService"
	testMethodName  = "TestMethod"
)

var errTestSendFailed = errors.New("send failed")

type mockFullServerStream struct {
	grpc.ServerStream

	ctx              context.Context //nolint:containedctx
	sentMessages     []*dynamicpb.Message
	receivedMessages []*dynamicpb.Message
	sendMsgError     error
	sendMsgFailAfter int
	recvMsgError     error
	recvMsgCount     int
	recvMsgLimit     int
	contextCancelled bool
	headers          metadata.MD
	trailers         metadata.MD
}

func (m *mockFullServerStream) Context() context.Context {
	if m.contextCancelled {
		ctx, cancel := context.WithCancel(m.ctx)
		cancel()

		return ctx
	}

	return m.ctx
}

func (m *mockFullServerStream) SendMsg(msg any) error {
	if m.sendMsgFailAfter > 0 && len(m.sentMessages) >= m.sendMsgFailAfter {
		return errTestSendFailed
	}

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
	m.headers = metadata.Join(m.headers, md)

	return nil
}

func (m *mockFullServerStream) SendHeader(md metadata.MD) error {
	return nil
}

func (m *mockFullServerStream) SetTrailer(md metadata.MD) {
	m.trailers = metadata.Join(m.trailers, md)
}

func TestHandleServerStreamWithArrayStream(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	mocker.fullMethod = testServiceName + "/" + testMethodName
	mocker.fullServiceName = testServiceName
	mocker.serviceName = testServiceName
	mocker.methodName = testMethodName

	stream := createTestStream(t, mocker)

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
	require.Len(t, stream.sentMessages, 2)
}

func TestHandleServerStreamWithNonArrayStream(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	mocker.fullMethod = testServiceName + "/" + testMethodName
	mocker.fullServiceName = testServiceName
	mocker.serviceName = testServiceName
	mocker.methodName = testMethodName

	stream := createTestStream(t, mocker)

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
	require.Len(t, stream.sentMessages, 1)
}

func TestHandleServerStreamWithHeaders(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	mocker.fullMethod = testServiceName + "/" + testMethodName
	mocker.fullServiceName = testServiceName
	mocker.serviceName = testServiceName
	mocker.methodName = testMethodName

	ctx := metadata.NewIncomingContext(t.Context(), metadata.New(map[string]string{
		"x-user": "testuser",
	}))
	stream := createTestStream(t, mocker)
	stream.ctx = ctx

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
	require.NotNil(t, stream.headers)
	require.Equal(t, "test", stream.headers.Get("x-response")[0])
}

func TestHandleServerStreamWithError(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	mocker.fullMethod = testServiceName + "/" + testMethodName
	mocker.fullServiceName = testServiceName
	mocker.serviceName = testServiceName
	mocker.methodName = testMethodName

	stream := createTestStream(t, mocker)

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
	require.Contains(t, err.Error(), "test error")
}

func TestHandleServerStreamRendersErrorTemplate(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	mocker.fullMethod = testServiceName + "/" + testMethodName
	mocker.fullServiceName = testServiceName
	mocker.serviceName = testServiceName
	mocker.methodName = testMethodName

	stream := createTestStream(t, mocker)

	id := uuid.New()
	stub := &stuber.Stub{
		ID:      id,
		Service: testServiceName,
		Method:  testMethodName,
		Input: stuber.InputData{
			Contains: map[string]any{},
		},
		Output: stuber.Output{
			Stream: []any{
				map[string]any{"message": "test"},
			},
			Error: "boom {{.StubID}}",
		},
	}

	mocker.budgerigar.PutMany(stub)

	err := mocker.handleServerStream(stream)
	require.Error(t, err)
	require.Contains(t, err.Error(), id.String())
	require.NotContains(t, err.Error(), "{{")
}

func TestHandleServerStreamEOF(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	stream := createTestStream(t, mocker)
	stream.recvMsgError = io.EOF

	err := mocker.handleServerStream(stream)
	require.NoError(t, err)
}

func TestHandleServerStreamRecvError(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	stream := createTestStream(t, mocker)
	stream.recvMsgError = status.Error(codes.Internal, "receive error")

	err := mocker.handleServerStream(stream)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to receive message")
}

func TestHandleServerStreamNotFound(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	mocker.fullMethod = testServiceName + "/" + testMethodName
	mocker.fullServiceName = testServiceName
	mocker.serviceName = testServiceName
	mocker.methodName = testMethodName

	stream := createTestStream(t, mocker)

	err := mocker.handleServerStream(stream)
	require.Error(t, err)
	require.Contains(t, err.Error(), "No matching stub found")
}

func TestHandleServerStreamEmptyStream(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	mocker.fullMethod = testServiceName + "/" + testMethodName
	mocker.fullServiceName = testServiceName
	mocker.serviceName = testServiceName
	mocker.methodName = testMethodName

	stream := createTestStream(t, mocker)

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: testServiceName,
		Method:  testMethodName,
		Input: stuber.InputData{
			Contains: map[string]any{},
		},
		Output: stuber.Output{
			Stream: []any{},
		},
	}

	mocker.budgerigar.PutMany(stub)

	err := mocker.handleServerStream(stream)
	require.NoError(t, err)
	require.Empty(t, stream.sentMessages)
}

func TestHandleNonArrayStreamDataSendsMessages(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	stream := &mockFullServerStream{
		ctx:          t.Context(),
		sentMessages: make([]*dynamicpb.Message, 0),
		recvMsgLimit: 1,
	}

	stub := &stuber.Stub{
		ID: uuid.New(),
		Output: stuber.Output{
			Data: map[string]any{"message": "test"},
		},
	}

	err := mocker.handleNonArrayStreamData(stream, stub, stub.Output, map[string]any{}, time.Now(), 1)
	require.NoError(t, err)
	require.Len(t, stream.sentMessages, 1)
}

func TestHandleNonArrayStreamDataWithDelay(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	stream := &mockFullServerStream{
		ctx:          t.Context(),
		sentMessages: make([]*dynamicpb.Message, 0),
		recvMsgLimit: 1,
	}

	stub := &stuber.Stub{
		ID: uuid.New(),
		Output: stuber.Output{
			Data:  map[string]any{"message": "test"},
			Delay: types.NewDelay(10 * time.Millisecond),
		},
	}

	start := time.Now()
	err := mocker.handleNonArrayStreamData(stream, stub, stub.Output, map[string]any{}, time.Now(), 1)
	duration := time.Since(start)

	require.NoError(t, err)
	require.GreaterOrEqual(t, duration, 10*time.Millisecond)
}

func TestHandleNonArrayStreamDataWithTemplates(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)
	stream := &mockFullServerStream{
		ctx:              t.Context(),
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

	err := mocker.handleNonArrayStreamData(stream, stub, stub.Output, map[string]any{}, time.Now(), 1)
	require.NoError(t, err)
	require.Len(t, stream.sentMessages, 1)
}

func TestHandleNonArrayStreamDataRendersFromCapturedRequest(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	stream := &mockFullServerStream{
		ctx:          t.Context(),
		sentMessages: make([]*dynamicpb.Message, 0),
		recvMsgLimit: 0,
	}

	stub := &stuber.Stub{
		ID:     uuid.New(),
		Output: stuber.Output{Data: map[string]any{"message": "Hello, {{.Request.name}}!"}},
	}

	err := mocker.handleNonArrayStreamData(stream, stub, stub.Output, map[string]any{"name": "Bob"}, time.Now(), 1)
	require.NoError(t, err)
	require.Len(t, stream.sentMessages, 1)

	body, err := protojson.Marshal(stream.sentMessages[0])
	require.NoError(t, err)
	require.Contains(t, string(body), "Hello, Bob!")
	require.NotContains(t, string(body), "{{")
}

func TestHandleNonArrayStreamDataContextCancelled(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	ctx, cancel := context.WithCancel(t.Context())
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

	err := mocker.handleNonArrayStreamData(stream, stub, stub.Output, map[string]any{}, time.Now(), 1)
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
}

func TestHandleNonArrayStreamDataWithError(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	stream := &mockFullServerStream{
		ctx:          t.Context(),
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

	err := mocker.handleNonArrayStreamData(stream, stub, stub.Output, map[string]any{}, time.Now(), 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "test error")
}

func TestHandleNonArrayStreamDataUsesTemplatedError(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	stream := &mockFullServerStream{
		ctx:          t.Context(),
		sentMessages: make([]*dynamicpb.Message, 0),
		recvMsgLimit: 0,
	}

	stub := &stuber.Stub{
		ID: uuid.New(),
		Output: stuber.Output{
			Data:  map[string]any{"message": "test"},
			Error: "{{.Request.name}}-RAW",
		},
	}

	rendered := stub.Output
	rendered.Error = "RENDERED-ERROR"

	err := mocker.handleNonArrayStreamData(stream, stub, rendered, map[string]any{}, time.Now(), 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "RENDERED-ERROR")
	require.NotContains(t, err.Error(), "RAW")
}

func TestSendClientStreamResponseRendersErrorTemplate(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	stream := &mockFullServerStream{
		ctx:          t.Context(),
		sentMessages: make([]*dynamicpb.Message, 0),
	}

	id := uuid.New()
	stub := &stuber.Stub{
		ID: id,
		Output: stuber.Output{
			Data:  map[string]any{"message": "ok"},
			Error: "boom {{.StubID}}",
		},
	}

	err := mocker.sendClientStreamResponse(stream, stub, []map[string]any{{"id": "1"}}, time.Now(), 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), id.String())
	require.NotContains(t, err.Error(), "{{")
}

func TestSendClientStreamResponseRendersHeaderTemplate(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	stream := &mockFullServerStream{
		ctx:          t.Context(),
		sentMessages: make([]*dynamicpb.Message, 0),
	}

	id := uuid.New()
	stub := &stuber.Stub{
		ID: id,
		Output: stuber.Output{
			Data:    map[string]any{"message": "ok"},
			Headers: map[string]string{"x-stub": "{{.StubID}}"},
		},
	}

	err := mocker.sendClientStreamResponse(stream, stub, []map[string]any{{"id": "1"}}, time.Now(), 1)
	require.NoError(t, err)
	require.NotNil(t, stream.headers)
	require.Equal(t, id.String(), stream.headers.Get("x-stub")[0])
	require.Len(t, stream.sentMessages, 1)
}

func TestSendClientStreamResponseSetsHeadersOnError(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	stream := &mockFullServerStream{
		ctx:          t.Context(),
		sentMessages: make([]*dynamicpb.Message, 0),
	}

	id := uuid.New()
	stub := &stuber.Stub{
		ID: id,
		Output: stuber.Output{
			Data:    map[string]any{"message": "ok"},
			Headers: map[string]string{"x-stub": "{{.StubID}}"},
			Error:   "boom",
		},
	}

	err := mocker.sendClientStreamResponse(stream, stub, []map[string]any{{"id": "1"}}, time.Now(), 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "boom")
	require.NotNil(t, stream.headers, "headers must be sent before the error trailer")
	require.Equal(t, id.String(), stream.headers.Get("x-stub")[0])
}

func TestSendClientStreamResponseAppliesEffectsOnError(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)

	victim := &stuber.Stub{
		ID:      uuid.New(),
		Service: "svc",
		Method:  "M",
		Output:  stuber.Output{Data: map[string]any{"x": float64(1)}},
	}
	mocker.budgerigar.PutMany(victim)

	stream := &mockFullServerStream{
		ctx:          t.Context(),
		sentMessages: make([]*dynamicpb.Message, 0),
	}

	stub := &stuber.Stub{
		ID: uuid.New(),
		Output: stuber.Output{
			Data:  map[string]any{"message": "ok"},
			Error: "boom",
		},
		Effects: []stuber.Effect{{Action: stuber.EffectActionDelete, ID: victim.ID.String()}},
	}

	err := mocker.sendClientStreamResponse(stream, stub, []map[string]any{{"id": "1"}}, time.Now(), 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "boom")
	require.Nil(t, mocker.budgerigar.FindByID(victim.ID), "delete effect must run despite error status")
}

func TestReceiveStreamMessageSuccess(t *testing.T) {
	t.Parallel()

	msg := dynamicpb.NewMessage((&structpb.Struct{}).ProtoReflect().Descriptor())
	stream := &mockFullServerStream{
		ctx:              t.Context(),
		receivedMessages: []*dynamicpb.Message{msg},
		recvMsgLimit:     1,
	}

	err := receiveStreamMessage(stream, msg)
	require.NoError(t, err)
}

func TestReceiveStreamMessageError(t *testing.T) {
	t.Parallel()

	msg := dynamicpb.NewMessage((&structpb.Struct{}).ProtoReflect().Descriptor())
	stream := &mockFullServerStream{
		ctx:          t.Context(),
		recvMsgError: status.Error(codes.Internal, "receive error"),
	}

	err := receiveStreamMessage(stream, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to receive message")
}

func TestProcessHeadersEmptyMetadata(t *testing.T) {
	t.Parallel()

	md := metadata.New(map[string]string{})
	result := processHeaders(md)
	require.Nil(t, result)
}

func TestProcessHeadersWithHeaders(t *testing.T) {
	t.Parallel()

	md := metadata.New(map[string]string{
		"x-user":     "testuser",
		"x-request":  "test",
		":authority": "localhost",
	})
	result := processHeaders(md)
	require.NotNil(t, result)
	require.Equal(t, "testuser", result["x-user"])
	require.Equal(t, "test", result["x-request"])
	require.NotContains(t, result, ":authority")
}

func TestProcessHeadersExcludedHeaders(t *testing.T) {
	t.Parallel()

	md := metadata.New(map[string]string{
		"content-type":         "application/grpc",
		"grpc-accept-encoding": "gzip",
		"user-agent":           "test",
		"accept-encoding":      "gzip",
		"x-custom":             "value",
	})
	result := processHeaders(md)
	require.NotNil(t, result)
	require.NotContains(t, result, "content-type")
	require.NotContains(t, result, "grpc-accept-encoding")
	require.NotContains(t, result, "user-agent")
	require.NotContains(t, result, "accept-encoding")
	require.Equal(t, "value", result["x-custom"])
}

func TestProcessHeadersMultipleValues(t *testing.T) {
	t.Parallel()

	md := metadata.Pairs(
		"x-header", "value1",
		"x-header", "value2",
		"x-header", "value3",
	)
	result := processHeaders(md)
	require.NotNil(t, result)
	require.Equal(t, "value1;value2;value3", result["x-header"])
}

func TestSessionFromMetadataEmpty(t *testing.T) {
	t.Parallel()

	md := metadata.New(nil)

	result := sessionFromMetadata(md)

	require.Empty(t, result)
}

func TestSessionFromMetadataFirstNonEmptyTrimmed(t *testing.T) {
	t.Parallel()

	md := metadata.Pairs(
		sessionHeaderKey, "   ",
		sessionHeaderKey, "  test-session  ",
	)

	result := sessionFromMetadata(md)

	require.Equal(t, "test-session", result)
}

func TestSessionFromMetadataHeaderAbsent(t *testing.T) {
	t.Parallel()

	md := metadata.Pairs("x-other", "value")

	result := sessionFromMetadata(md)

	require.Empty(t, result)
}

func TestConvertToMapProto3DefaultValues(t *testing.T) {
	t.Parallel()

	t.Run("wrapperspb_DoubleValue", func(t *testing.T) {
		t.Parallel()

		msg := &wrapperspb.DoubleValue{}
		result := convertToMapWithDepth(msg, defaultConvertDepth)
		require.NotNil(t, result)
		require.Contains(t, result, "value")
		require.InDelta(t, 0.0, result["value"], 1e-9)
	})

	t.Run("dynamicpb_empty_message", func(t *testing.T) {
		t.Parallel()

		desc := (&wrapperspb.DoubleValue{}).ProtoReflect().Descriptor()
		msg := dynamicpb.NewMessage(desc)
		result := convertToMapWithDepth(msg, defaultConvertDepth)
		require.NotNil(t, result)
		require.Contains(t, result, "value")
		require.InDelta(t, 0.0, result["value"], 1e-9)
	})
}

func TestHandleServerStreamSetsTrailers(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	mocker.fullMethod = testServiceName + "/" + testMethodName
	mocker.fullServiceName = testServiceName
	mocker.serviceName = testServiceName
	mocker.methodName = testMethodName

	stream := createTestStream(t, mocker)

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: testServiceName,
		Method:  testMethodName,
		Input:   stuber.InputData{Contains: map[string]any{}},
		Output: stuber.Output{
			Stream:   []any{map[string]any{"message": "test"}},
			Headers:  map[string]string{"x-response": "header-value"},
			Trailers: map[string]string{"x-response": "trailer-value", "x-audit": "a;b"},
		},
	}
	mocker.budgerigar.PutMany(stub)

	require.NoError(t, mocker.handleServerStream(stream))

	require.Equal(t, "header-value", stream.headers.Get("x-response")[0])
	require.Equal(t, "trailer-value", stream.trailers.Get("x-response")[0],
		"headers and trailers are separate channels and may share a key")
	require.Equal(t, []string{"a", "b"}, stream.trailers.Get("x-audit"),
		"';' splits into repeated metadata values, as for headers")
}

func TestTrailersDropKeysOwnedByTheTransport(t *testing.T) {
	t.Parallel()

	md := buildResponseMD(map[string]string{
		"grpc-status":             "0",
		"grpc-message":            "forged",
		"grpc-status-details-bin": "x",
		"content-type":            "application/grpc",
		":status":                 "200",
		"x-keep":                  "yes",
	})

	require.Empty(t, md.Get("grpc-status"), "a stub-set grpc-status would override the real status")
	require.Empty(t, md.Get("grpc-message"))
	require.Empty(t, md.Get("grpc-status-details-bin"))
	require.Empty(t, md.Get("content-type"))
	require.Empty(t, md.Get(":status"))
	require.Equal(t, "yes", md.Get("x-keep")[0])
}

func TestHandleBidiStreamFlushesTrailersOnce(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	mocker.fullMethod = testServiceName + "/" + testMethodName
	mocker.fullServiceName = testServiceName
	mocker.serviceName = testServiceName
	mocker.methodName = testMethodName

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: testServiceName,
		Method:  testMethodName,
		Input:   stuber.InputData{Contains: map[string]any{}},
		Inputs:  []stuber.InputData{{Contains: map[string]any{}}, {Contains: map[string]any{}}},
		Output: stuber.Output{
			Stream:   []any{map[string]any{"status": "ack1"}, map[string]any{"status": "ack2"}},
			Trailers: map[string]string{"x-audit": "done"},
		},
	}
	mocker.budgerigar.PutMany(stub)

	inputMsg1 := dynamicpb.NewMessage(mocker.inputDesc)
	inputMsg2 := dynamicpb.NewMessage(mocker.inputDesc)
	stream := &mockFullServerStream{
		ctx:              t.Context(),
		sentMessages:     make([]*dynamicpb.Message, 0),
		receivedMessages: []*dynamicpb.Message{inputMsg1, inputMsg2},
		recvMsgLimit:     2,
	}

	require.NoError(t, mocker.handleBidiStream(stream))
	require.Equal(t, []string{"done"}, stream.trailers.Get("x-audit"),
		"SetTrailer is cumulative: emitting per message would repeat the value once per message")
}

func TestHandleServerStreamFailsAtMarkedElement(t *testing.T) {
	t.Parallel()

	mocker := createTestMockerWithRecorder(t)
	mocker.fullMethod = testServiceName + "/" + testMethodName
	mocker.fullServiceName = testServiceName
	mocker.serviceName = testServiceName
	mocker.methodName = testMethodName

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: testServiceName,
		Method:  testMethodName,
		Input:   stuber.InputData{Contains: map[string]any{}},
		Output: stuber.Output{Stream: []any{
			map[string]any{"message": "first"},
			map[string]any{stuber.GripMockKey: map[string]any{
				"error": "resources gone",
				"code":  float64(codes.ResourceExhausted),
			}},
			map[string]any{"message": "never sent"},
		}},
	}
	mocker.budgerigar.PutMany(stub)

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)
	stream := &mockFullServerStream{
		ctx:              t.Context(),
		sentMessages:     make([]*dynamicpb.Message, 0),
		receivedMessages: []*dynamicpb.Message{inputMsg},
		recvMsgLimit:     1,
	}

	err := mocker.handleServerStream(stream)
	require.Error(t, err)
	require.Equal(t, codes.ResourceExhausted, status.Code(err))
	require.Contains(t, err.Error(), "resources gone")
	require.Len(t, stream.sentMessages, 1, "the marked element replaces a message, it does not follow one")
}
