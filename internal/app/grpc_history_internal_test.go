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

func TestHistoryUnary(t *testing.T) {
	t.Parallel()

	mocker := createTestMockerWithRecorder(t)
	mocker.fullMethod = testServiceName + "/" + testMethodName
	mocker.fullServiceName = testServiceName
	mocker.serviceName = testServiceName
	mocker.methodName = testMethodName

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)

	ctx := t.Context()

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: testServiceName,
		Method:  testMethodName,
		Input: stuber.InputData{
			Contains: map[string]any{},
		},
		Output: stuber.Output{
			Data: map[string]any{"result": 100},
		},
	}

	mocker.budgerigar.PutMany(stub)

	resp, err := mocker.handleUnary(ctx, inputMsg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	recorder, ok := mocker.recorder.(*history.MemoryStore)
	require.True(t, ok, testRecorderShouldBeMemoryStore)
	require.Equal(t, 1, recorder.Count(), "unary call should be recorded")

	calls := recorder.Filter(history.FilterOpts{})
	require.Len(t, calls, 1)
	require.Equal(t, testServiceName, calls[0].Service)
	require.Equal(t, testMethodName, calls[0].Method)
	require.Len(t, calls[0].Requests, 1, "1 request")
	require.Len(t, calls[0].Responses, 1, "1 response")
	require.Equal(t, stub.ID, calls[0].StubID)
	require.Equal(t, uint32(0), calls[0].Code, "OK code")
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

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)

	stream := &mockFullServerStream{
		ctx:              t.Context(),
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
				map[string]any{"message": "test3"},
			},
		},
	}

	mocker.budgerigar.PutMany(stub)

	err := mocker.handleServerStream(stream)
	require.NoError(t, err)

	recorder, ok := mocker.recorder.(*history.MemoryStore)
	require.True(t, ok, testRecorderShouldBeMemoryStore)
	require.Equal(t, 1, recorder.Count(), "server stream call should be recorded as 1 call")

	calls := recorder.Filter(history.FilterOpts{})
	require.Len(t, calls, 1)
	require.Equal(t, testServiceName, calls[0].Service)
	require.Equal(t, testMethodName, calls[0].Method)
	require.Len(t, calls[0].Requests, 1, "1 request (single input)")
	require.Len(t, calls[0].Responses, 3, "3 responses (stream items)")
	require.Equal(t, stub.ID, calls[0].StubID)
}

func TestHistoryClientStreamN1(t *testing.T) {
	t.Parallel()

	mocker := createTestMockerWithRecorder(t)
	mocker.fullMethod = testServiceName + "/" + testMethodName
	mocker.fullServiceName = testServiceName
	mocker.serviceName = testServiceName
	mocker.methodName = testMethodName

	inputMsg1 := dynamicpb.NewMessage(mocker.inputDesc)
	inputMsg2 := dynamicpb.NewMessage(mocker.inputDesc)
	inputMsg3 := dynamicpb.NewMessage(mocker.inputDesc)

	stream := &mockFullServerStream{
		ctx:              t.Context(),
		sentMessages:     make([]*dynamicpb.Message, 0),
		receivedMessages: []*dynamicpb.Message{inputMsg1, inputMsg2, inputMsg3},
		recvMsgLimit:     3,
	}

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: testServiceName,
		Method:  testMethodName,
		Inputs: []stuber.InputData{
			{Contains: map[string]any{}},
		},
		Output: stuber.Output{
			Data: map[string]any{"result": 30},
		},
	}

	mocker.budgerigar.PutMany(stub)

	err := mocker.handleClientStream(stream)
	require.NoError(t, err)

	recorder, ok := mocker.recorder.(*history.MemoryStore)
	require.True(t, ok, testRecorderShouldBeMemoryStore)
	require.Equal(t, 1, recorder.Count(), "client stream call should be recorded as 1 call")

	calls := recorder.Filter(history.FilterOpts{})
	require.Len(t, calls, 1)
	require.Equal(t, testServiceName, calls[0].Service)
	require.Equal(t, testMethodName, calls[0].Method)
	require.Len(t, calls[0].Requests, 3, "3 requests (multiple inputs)")
	require.Len(t, calls[0].Responses, 1, "1 response")
	require.NotNil(t, calls[0].Request, "deprecated Request field should be populated")
	require.NotNil(t, calls[0].Response, "deprecated Response field should be populated")
}

func TestHistoryBidiStreamNM(t *testing.T) {
	t.Parallel()

	mocker := createTestMockerWithRecorder(t)
	mocker.fullMethod = testServiceName + "/" + testMethodName
	mocker.fullServiceName = testServiceName
	mocker.serviceName = testServiceName
	mocker.methodName = testMethodName

	inputMsg1 := dynamicpb.NewMessage(mocker.inputDesc)
	inputMsg2 := dynamicpb.NewMessage(mocker.inputDesc)

	stream := &mockFullServerStream{
		ctx:              t.Context(),
		sentMessages:     make([]*dynamicpb.Message, 0),
		receivedMessages: []*dynamicpb.Message{inputMsg1, inputMsg2},
		recvMsgLimit:     2,
	}

	stub := &stuber.Stub{
		ID:      uuid.New(),
		Service: testServiceName,
		Method:  testMethodName,
		Input: stuber.InputData{
			Contains: map[string]any{},
		},
		Inputs: []stuber.InputData{
			{Contains: map[string]any{}},
			{Contains: map[string]any{}},
		},
		Output: stuber.Output{
			Stream: []any{
				map[string]any{"status": "ack1"},
				map[string]any{"status": "ack2"},
			},
		},
	}

	mocker.budgerigar.PutMany(stub)

	err := mocker.handleBidiStream(stream)
	require.NoError(t, err)

	recorder, ok := mocker.recorder.(*history.MemoryStore)
	require.True(t, ok, testRecorderShouldBeMemoryStore)
	require.Equal(t, 1, recorder.Count(), "bidi stream with 2 messages should be 1 call (semantic call approach)")

	calls := recorder.Filter(history.FilterOpts{})
	require.Len(t, calls, 1)
	require.Equal(t, testServiceName, calls[0].Service)
	require.Equal(t, testMethodName, calls[0].Method)
	require.Len(t, calls[0].Requests, 2, "2 client messages")
	require.Len(t, calls[0].Responses, 2, "2 server messages")
	require.Equal(t, stub.ID, calls[0].StubID)
	require.Equal(t, uint32(0), calls[0].Code, "OK code")
	require.NotNil(t, calls[0].Request, "deprecated Request field should be populated")
	require.NotNil(t, calls[0].Response, "deprecated Response field should be populated")
}

func TestHistoryBidiStream11(t *testing.T) {
	t.Parallel()

	mocker := createTestMockerWithRecorder(t)
	mocker.fullMethod = testServiceName + "/" + testMethodName
	mocker.fullServiceName = testServiceName
	mocker.serviceName = testServiceName
	mocker.methodName = testMethodName

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)

	stream := &mockFullServerStream{
		ctx:              t.Context(),
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
		Inputs: []stuber.InputData{
			{Contains: map[string]any{}},
		},
		Output: stuber.Output{
			Stream: []any{
				map[string]any{"status": "ok"},
			},
		},
	}

	mocker.budgerigar.PutMany(stub)

	err := mocker.handleBidiStream(stream)
	require.NoError(t, err)

	recorder, ok := mocker.recorder.(*history.MemoryStore)
	require.True(t, ok, testRecorderShouldBeMemoryStore)
	require.Equal(t, 1, recorder.Count(), "bidi stream with single message should be 1 call")

	calls := recorder.Filter(history.FilterOpts{})
	require.Len(t, calls, 1)
	require.Equal(t, testServiceName, calls[0].Service)
	require.Equal(t, testMethodName, calls[0].Method)
	require.Len(t, calls[0].Requests, 1, "1 request")
	require.Len(t, calls[0].Responses, 1, "1 response")
	require.Equal(t, stub.ID, calls[0].StubID)
}

func TestHistoryServerStreamWithError(t *testing.T) {
	t.Parallel()

	mocker := createTestMockerWithRecorder(t)
	mocker.fullMethod = testServiceName + "/" + testMethodName
	mocker.fullServiceName = testServiceName
	mocker.serviceName = testServiceName
	mocker.methodName = testMethodName

	inputMsg := dynamicpb.NewMessage(mocker.inputDesc)

	stream := &mockFullServerStream{
		ctx:              t.Context(),
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
			Data:  map[string]any{"result": 100},
			Error: "stub error",
		},
	}

	mocker.budgerigar.PutMany(stub)

	err := mocker.handleServerStream(stream)
	require.Error(t, err, "handleServerStream should return error when stub has Error message")
	require.Contains(t, err.Error(), "stub error", "error message should contain stub error")

	recorder, ok := mocker.recorder.(*history.MemoryStore)
	require.True(t, ok, testRecorderShouldBeMemoryStore)
	require.Equal(t, 0, recorder.Count(), "server stream with error should NOT be recorded (error returned before recordCall)")
}

func createTestMockerWithRecorder(t *testing.T) *grpcMocker {
	t.Helper()

	structDesc := (&structpb.Struct{}).ProtoReflect().Descriptor()
	testRegistry := plugintest.NewRegistry()
	templateEngine := template.New(t.Context(), testRegistry)

	recorder := history.NewMemoryStore(0)

	return &grpcMocker{
		budgerigar:     stuber.NewBudgerigar(),
		templateEngine: templateEngine,
		inputDesc:      structDesc,
		outputDesc:     structDesc,
		recorder:       recorder,
	}
}
