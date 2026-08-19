package app

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/domain/history"
)

func newUnmatchedMocker(t *testing.T) *grpcMocker {
	t.Helper()

	mocker := createTestMockerWithRecorder(t)
	mocker.fullMethod = testServiceName + "/" + testMethodName
	mocker.fullServiceName = testServiceName
	mocker.serviceName = testServiceName
	mocker.methodName = testMethodName
	mocker.errorFormatter = NewErrorFormatter()

	return mocker
}

func unmatchedRecords(t *testing.T, mocker *grpcMocker) []history.CallRecord {
	t.Helper()

	recorder, ok := mocker.recorder.(*history.MemoryStore)
	require.True(t, ok, testRecorderShouldBeMemoryStore)

	return recorder.Filter(history.FilterOpts{})
}

func TestHistoryUnaryRecordsUnmatchedCall(t *testing.T) {
	t.Parallel()

	mocker := newUnmatchedMocker(t)

	_, err := mocker.handleUnary(t.Context(), nil, dynamicpb.NewMessage(mocker.inputDesc))
	require.Error(t, err)

	calls := unmatchedRecords(t, mocker)
	require.Len(t, calls, 1)
	require.Equal(t, uint32(codes.NotFound), calls[0].Code)
	require.NotEmpty(t, calls[0].Error)
}

func TestHistoryServerStreamRecordsUnmatchedCall(t *testing.T) {
	t.Parallel()

	mocker := newUnmatchedMocker(t)

	stream := &mockFullServerStream{
		ctx:              t.Context(),
		sentMessages:     make([]*dynamicpb.Message, 0),
		receivedMessages: []*dynamicpb.Message{dynamicpb.NewMessage(mocker.inputDesc)},
		recvMsgLimit:     1,
	}

	require.Error(t, mocker.handleServerStream(stream))

	calls := unmatchedRecords(t, mocker)
	require.Len(t, calls, 1)
	require.Equal(t, uint32(codes.NotFound), calls[0].Code)
}

func TestHistoryClientStreamRecordsUnmatchedCall(t *testing.T) {
	t.Parallel()

	mocker := newUnmatchedMocker(t)

	stream := &mockFullServerStream{
		ctx:              t.Context(),
		sentMessages:     make([]*dynamicpb.Message, 0),
		receivedMessages: []*dynamicpb.Message{dynamicpb.NewMessage(mocker.inputDesc)},
		recvMsgLimit:     1,
	}

	require.Error(t, mocker.handleClientStream(stream))

	calls := unmatchedRecords(t, mocker)
	require.Len(t, calls, 1)
	require.Equal(t, uint32(codes.NotFound), calls[0].Code)
}
