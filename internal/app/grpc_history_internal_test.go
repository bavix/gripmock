package app

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
	"github.com/bavix/gripmock/v3/pkg/plugintest"
)

const testRecorderShouldBeMemoryStore = "recorder should be *MemoryStore"

func createTestMockerWithRecorder(t *testing.T) *grpcMocker {
	t.Helper()

	structDesc := (&structpb.Struct{}).ProtoReflect().Descriptor()

	return &grpcMocker{
		budgerigar:     stuber.NewBudgerigar(),
		templateEngine: template.New(t.Context(), plugintest.NewRegistry()),
		inputDesc:      structDesc,
		outputDesc:     structDesc,
		recorder:       history.NewMemoryStore(0),
	}
}

func TestHistoryUnary(t *testing.T) {
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
		Output:  stuber.Output{Data: map[string]any{"result": 100}},
	}
	mocker.budgerigar.PutMany(stub)

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)
	resp, err := mocker.handleUnary(t.Context(), nil, inputMsg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	recorder, ok := mocker.recorder.(*history.MemoryStore)
	require.True(t, ok, testRecorderShouldBeMemoryStore)
	require.Equal(t, 1, recorder.Count())

	calls := recorder.Filter(history.FilterOpts{})
	require.Len(t, calls, 1)
	require.Len(t, calls[0].Requests, 1)
	require.Len(t, calls[0].Responses, 1)
	require.Equal(t, stub.ID, calls[0].StubID)
	require.Equal(t, uint32(0), calls[0].Code)
	require.NotNil(t, calls[0].Request)
	require.NotNil(t, calls[0].Response)
}

func TestUnaryStructuralDataTemplate(t *testing.T) {
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
		Output: stuber.Output{DataTemplate: `items:
{{ range .Request.fields.items.list_value.values }}
  - id: "{{ .struct_value.fields.id.string_value }}"
{{ else }}
  []
{{ end }}`},
	}
	mocker.budgerigar.PutMany(stub)

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)
	require.NoError(t, protojson.Unmarshal([]byte(`{"items":[{"id":"a"},{"id":"b"},{"id":"c"}]}`), inputMsg))

	resp, err := mocker.handleUnary(t.Context(), nil, inputMsg)
	require.NoError(t, err)

	rendered, err := protojson.Marshal(resp)
	require.NoError(t, err)
	require.JSONEq(t, `{"items":[{"id":"a"},{"id":"b"},{"id":"c"}]}`, string(rendered))
}

func TestHistoryServerStream1N(t *testing.T) {
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
			map[string]any{"message": "test1"},
			map[string]any{"message": "test2"},
			map[string]any{"message": "test3"},
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
	require.NoError(t, err)

	recorder, ok := mocker.recorder.(*history.MemoryStore)
	require.True(t, ok, testRecorderShouldBeMemoryStore)
	require.Equal(t, 1, recorder.Count())
	calls := recorder.Filter(history.FilterOpts{})
	require.Len(t, calls, 1)
	require.Equal(t, stub.ID, calls[0].StubID)
	require.Len(t, calls[0].Requests, 1)
	require.Len(t, calls[0].Responses, 3)
}

func TestHistoryClientStreamN1(t *testing.T) {
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
		Inputs:  []stuber.InputData{{Contains: map[string]any{}}},
		Output:  stuber.Output{Data: map[string]any{"result": 30}},
	}
	mocker.budgerigar.PutMany(stub)

	inputMsg1 := dynamicpb.NewMessage(mocker.inputDesc)
	inputMsg2 := dynamicpb.NewMessage(mocker.inputDesc)
	inputMsg3 := dynamicpb.NewMessage(mocker.inputDesc)
	stream := &mockFullServerStream{
		ctx:              t.Context(),
		sentMessages:     make([]*dynamicpb.Message, 0),
		receivedMessages: []*dynamicpb.Message{inputMsg1, inputMsg2, inputMsg3},
		recvMsgLimit:     3,
	}
	err := mocker.handleClientStream(stream)
	require.NoError(t, err)

	recorder, ok := mocker.recorder.(*history.MemoryStore)
	require.True(t, ok, testRecorderShouldBeMemoryStore)
	require.Equal(t, 1, recorder.Count())
	calls := recorder.Filter(history.FilterOpts{})
	require.Len(t, calls, 1)
	require.Equal(t, stub.ID, calls[0].StubID)
	require.Len(t, calls[0].Requests, 3)
	require.Len(t, calls[0].Responses, 1)
	require.NotNil(t, calls[0].Request)
	require.NotNil(t, calls[0].Response)
}

func TestClientStreamStructuralDataTemplate(t *testing.T) {
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
		Inputs:  []stuber.InputData{{Contains: map[string]any{}}},
		Output: stuber.Output{DataTemplate: `items:
{{ range .Requests }}
  - received: true
{{ else }}
  []
{{ end }}`},
	}
	mocker.budgerigar.PutMany(stub)

	stream := &mockFullServerStream{
		ctx: t.Context(),
		receivedMessages: []*dynamicpb.Message{
			dynamicpb.NewMessage(mocker.inputDesc),
			dynamicpb.NewMessage(mocker.inputDesc),
			dynamicpb.NewMessage(mocker.inputDesc),
		},
		recvMsgLimit: 3,
	}

	err := mocker.handleClientStream(stream)
	require.NoError(t, err)
	require.Len(t, stream.sentMessages, 1)

	rendered, err := protojson.Marshal(stream.sentMessages[0])
	require.NoError(t, err)
	require.JSONEq(t, `{"items":[{"received":true},{"received":true},{"received":true}]}`, string(rendered))
}

func TestHistoryBidiStreamNM(t *testing.T) {
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
		Inputs:  []stuber.InputData{{Contains: map[string]any{}}, {Contains: map[string]any{}}},
		Output: stuber.Output{Stream: []any{
			map[string]any{"status": "ack1"},
			map[string]any{"status": "ack2"},
		}},
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
	err := mocker.handleBidiStream(stream)
	require.NoError(t, err)

	recorder, ok := mocker.recorder.(*history.MemoryStore)
	require.True(t, ok, testRecorderShouldBeMemoryStore)
	require.Equal(t, 1, recorder.Count())
	calls := recorder.Filter(history.FilterOpts{})
	require.Len(t, calls, 1)
	require.Equal(t, stub.ID, calls[0].StubID)
	require.Len(t, calls[0].Requests, 2)
	require.Len(t, calls[0].Responses, 2)
	require.Equal(t, uint32(0), calls[0].Code)
	require.NotNil(t, calls[0].Request)
	require.NotNil(t, calls[0].Response)
}

func TestBidiStructuralStreamTemplate(t *testing.T) {
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
		Inputs:  []stuber.InputData{{Contains: map[string]any{}}, {Contains: map[string]any{}}},
		Output:  stuber.Output{StreamTemplate: `- status: "ack-{{ .MessageIndex }}"`},
	}
	mocker.budgerigar.PutMany(stub)

	stream := &mockFullServerStream{
		ctx: t.Context(),
		receivedMessages: []*dynamicpb.Message{
			dynamicpb.NewMessage(mocker.inputDesc),
			dynamicpb.NewMessage(mocker.inputDesc),
		},
		recvMsgLimit: 2,
	}

	err := mocker.handleBidiStream(stream)
	require.NoError(t, err)
	require.Len(t, stream.sentMessages, 2)

	first, err := protojson.Marshal(stream.sentMessages[0])
	require.NoError(t, err)
	second, err := protojson.Marshal(stream.sentMessages[1])
	require.NoError(t, err)
	require.JSONEq(t, `{"status":"ack-0"}`, string(first))
	require.JSONEq(t, `{"status":"ack-1"}`, string(second))
}

func TestHistoryBidiStream11(t *testing.T) {
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
		Inputs:  []stuber.InputData{{Contains: map[string]any{}}},
		Output:  stuber.Output{Stream: []any{map[string]any{"status": "ok"}}},
	}
	mocker.budgerigar.PutMany(stub)

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)
	stream := &mockFullServerStream{
		ctx:              t.Context(),
		sentMessages:     make([]*dynamicpb.Message, 0),
		receivedMessages: []*dynamicpb.Message{inputMsg},
		recvMsgLimit:     1,
	}
	err := mocker.handleBidiStream(stream)
	require.NoError(t, err)

	recorder, ok := mocker.recorder.(*history.MemoryStore)
	require.True(t, ok, testRecorderShouldBeMemoryStore)
	require.Equal(t, 1, recorder.Count())
	calls := recorder.Filter(history.FilterOpts{})
	require.Len(t, calls, 1)
	require.Equal(t, stub.ID, calls[0].StubID)
	require.Len(t, calls[0].Requests, 1)
	require.Len(t, calls[0].Responses, 1)
}

// Regression: recordBidiStream hardcoded codes.Unknown for every error, so a
// bidi stub returning a configured Output.Code was recorded with the wrong code.
func TestHistoryBidiStreamRecordsConfiguredErrorCode(t *testing.T) {
	t.Parallel()

	mocker := createTestMockerWithRecorder(t)
	mocker.fullMethod = testServiceName + "/" + testMethodName
	mocker.fullServiceName = testServiceName
	mocker.serviceName = testServiceName
	mocker.methodName = testMethodName

	notFound := codes.NotFound
	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: testServiceName,
		Method:  testMethodName,
		Input:   stuber.InputData{Contains: map[string]any{}},
		Inputs:  []stuber.InputData{{Contains: map[string]any{}}},
		Output:  stuber.Output{Error: "boom", Code: &notFound},
	}
	mocker.budgerigar.PutMany(stub)

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)
	stream := &mockFullServerStream{
		ctx:              t.Context(),
		sentMessages:     make([]*dynamicpb.Message, 0),
		receivedMessages: []*dynamicpb.Message{inputMsg},
		recvMsgLimit:     1,
	}

	err := mocker.handleBidiStream(stream)
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))

	recorder, ok := mocker.recorder.(*history.MemoryStore)
	require.True(t, ok, testRecorderShouldBeMemoryStore)

	calls := recorder.Filter(history.FilterOpts{})
	require.Len(t, calls, 1)
	require.Equal(t, uint32(codes.NotFound), calls[0].Code, "must record the configured code, not Unknown")
}

// Regression: a server stream that failed returned before recordCall, so the
// call vanished from history and from /api/verify — the bidi path already
// recorded partial exchanges. Now both record the error with its real code.
func TestHistoryServerStreamWithError(t *testing.T) {
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
		Output:  stuber.Output{Data: map[string]any{"result": 100}, Error: "stub error"},
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
	require.Contains(t, err.Error(), "stub error")

	recorder, ok := mocker.recorder.(*history.MemoryStore)
	require.True(t, ok, testRecorderShouldBeMemoryStore)

	calls := recorder.Filter(history.FilterOpts{})
	require.Len(t, calls, 1)
	require.Equal(t, stub.ID, calls[0].StubID)
	require.Equal(t, uint32(codes.Aborted), calls[0].Code, "Output.Error without Code means Aborted")
	require.Contains(t, calls[0].Error, "stub error")
}

// Regression: when a server stream broke partway, handleArrayStreamData
// returned before recordCall and the whole call disappeared from history —
// precisely the case a user is testing. The messages already sent must be
// recorded alongside the terminal status.
func TestHistoryServerStreamRecordsPartialStream(t *testing.T) {
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
			map[string]any{"result": 1},
			map[string]any{"result": 2},
			map[string]any{"result": 3},
		}},
	}
	mocker.budgerigar.PutMany(stub)

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)
	stream := &mockFullServerStream{
		ctx:              t.Context(),
		sentMessages:     make([]*dynamicpb.Message, 0),
		receivedMessages: []*dynamicpb.Message{inputMsg},
		recvMsgLimit:     1,
		sendMsgFailAfter: 2,
	}

	err := mocker.handleServerStream(stream)
	require.Error(t, err)
	require.Len(t, stream.sentMessages, 2, "two messages reached the client before the break")

	recorder, ok := mocker.recorder.(*history.MemoryStore)
	require.True(t, ok, testRecorderShouldBeMemoryStore)

	calls := recorder.Filter(history.FilterOpts{})
	require.Len(t, calls, 1)
	require.Equal(t, stub.ID, calls[0].StubID)
	require.Len(t, calls[0].Responses, 2, "only the sent messages are recorded, not all three")
	require.NotEmpty(t, calls[0].Error)
}
