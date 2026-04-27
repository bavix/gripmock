package app

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
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
	resp, err := mocker.handleUnary(t.Context(), inputMsg)
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
	require.Equal(t, 0, recorder.Count())
}
