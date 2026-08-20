package app

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/types"
)

const streamDelayStep = 150 * time.Millisecond

func newDelayStreamMocker(t *testing.T, output stuber.Output) (*grpcMocker, *mockFullServerStream) {
	t.Helper()

	mocker := createTestMocker(t)
	mocker.fullMethod = testServiceName + "/" + testMethodName
	mocker.fullServiceName = testServiceName
	mocker.serviceName = testServiceName
	mocker.methodName = testMethodName

	stream := createTestStream(t, mocker)

	output.Delay = types.NewDelay(streamDelayStep)

	mocker.budgerigar.PutMany(&stuber.Stub{
		ID:      uuid.New(),
		Service: testServiceName,
		Method:  testMethodName,
		Input:   stuber.InputData{Contains: map[string]any{}},
		Output:  output,
	})

	return mocker, stream
}

func TestServerStreamDelayAppliesOncePerMessage(t *testing.T) {
	t.Parallel()

	mocker, stream := newDelayStreamMocker(t, stuber.Output{
		Stream: []any{map[string]any{"message": "only"}},
	})

	start := time.Now()

	require.NoError(t, mocker.handleServerStream(stream))

	elapsed := time.Since(start)

	require.Len(t, stream.sentMessages, 1)
	require.Less(t, elapsed, 2*streamDelayStep,
		"a one-message stream waited more than its single delay")
	require.GreaterOrEqual(t, elapsed, streamDelayStep)
}

func TestServerStreamDelayStillAppliesBeforeErrorOnlyOutput(t *testing.T) {
	t.Parallel()

	mocker, stream := newDelayStreamMocker(t, stuber.Output{
		Stream: []any{},
		Error:  "boom",
	})

	start := time.Now()

	require.Error(t, mocker.handleServerStream(stream))

	require.GreaterOrEqual(t, time.Since(start), streamDelayStep)
	require.Empty(t, stream.sentMessages)
}

func TestBidiDelayAppliesOncePerSentMessage(t *testing.T) {
	t.Parallel()

	mocker := createTestMocker(t)
	mocker.fullMethod = testServiceName + "/" + testMethodName
	mocker.fullServiceName = testServiceName
	mocker.serviceName = testServiceName
	mocker.methodName = testMethodName
	mocker.clientStream = true
	mocker.serverStream = true

	stream := createTestStream(t, mocker)

	mocker.budgerigar.PutMany(&stuber.Stub{
		ID:      uuid.New(),
		Service: testServiceName,
		Method:  testMethodName,
		Inputs:  []stuber.InputData{{Contains: map[string]any{}}},
		Output: stuber.Output{
			Delay:  types.NewDelay(streamDelayStep),
			Stream: []any{map[string]any{"message": "one"}},
		},
	})

	start := time.Now()

	require.NoError(t, mocker.handleBidiStream(stream))

	elapsed := time.Since(start)

	require.Len(t, stream.sentMessages, 1)
	require.Less(t, elapsed, 2*streamDelayStep,
		"one bidi response waited more than its single delay")
	require.GreaterOrEqual(t, elapsed, streamDelayStep)
}
