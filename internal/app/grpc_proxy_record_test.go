package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

type mockBudgerigar struct {
	stubs []*stuber.Stub
}

func (m *mockBudgerigar) PutMany(stubs ...*stuber.Stub) {
	m.stubs = append(m.stubs, stubs...)
}

type grpcMockerForTest struct {
	budgerigar *mockBudgerigar
}

func TestRecordCapturedStubWithDelays_NilStub(t *testing.T) {
	t.Parallel()

	m := &grpcMockerForTest{
		budgerigar: &mockBudgerigar{},
	}

	m.recordCapturedStubWithDelays(
		func() *stuber.Stub { return nil },
		true,
		[]time.Duration{100 * time.Millisecond},
	)

	require.Empty(t, m.budgerigar.stubs)
}

func (m *grpcMockerForTest) recordCapturedStubWithDelays(
	build func() *stuber.Stub,
	recordDelay bool,
	delays []time.Duration,
) {
	stub := build()
	if stub == nil {
		return
	}

	if recordDelay && len(delays) > 0 {
		for i, d := range delays {
			if d == 0 {
				continue
			}

			if stub.Output.Stream[i] == nil {
				continue
			}

			itemMap, ok := stub.Output.Stream[i].(map[string]any)
			if !ok {
				itemMap = map[string]any{"data": stub.Output.Stream[i]}
				stub.Output.Stream[i] = itemMap
			}

			itemMap["delay"] = d.String()
		}
	}

	m.budgerigar.PutMany(stub)
}

func TestRecordCapturedStubWithDelays_WithDelays(t *testing.T) {
	t.Parallel()

	m := &grpcMockerForTest{
		budgerigar: &mockBudgerigar{},
	}

	stub := &stuber.Stub{
		Service: "test.Service",
		Method:  "TestMethod",
		Output: stuber.Output{
			Stream: []any{
				map[string]any{"data": map[string]any{"result": "first"}},
				map[string]any{"data": map[string]any{"result": "second"}},
				map[string]any{"data": map[string]any{"result": "third"}},
			},
		},
	}

	delays := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		300 * time.Millisecond,
	}

	m.recordCapturedStubWithDelays(
		func() *stuber.Stub { return stub },
		true,
		delays,
	)

	require.Len(t, m.budgerigar.stubs, 1)

	resultStub := m.budgerigar.stubs[0]
	require.Len(t, resultStub.Output.Stream, 3)

	entry0 := resultStub.Output.Stream[0].(map[string]any)
	require.Equal(t, "100ms", entry0["delay"])
	require.Equal(t, map[string]any{"result": "first"}, entry0["data"])

	entry1 := resultStub.Output.Stream[1].(map[string]any)
	require.Equal(t, "200ms", entry1["delay"])
	require.Equal(t, map[string]any{"result": "second"}, entry1["data"])

	entry2 := resultStub.Output.Stream[2].(map[string]any)
	require.Equal(t, "300ms", entry2["delay"])
	require.Equal(t, map[string]any{"result": "third"}, entry2["data"])
}

func TestRecordCapturedStubWithDelays_RecordDelayFalse(t *testing.T) {
	t.Parallel()

	m := &grpcMockerForTest{
		budgerigar: &mockBudgerigar{},
	}

	stub := &stuber.Stub{
		Service: "test.Service",
		Method:  "TestMethod",
		Output: stuber.Output{
			Stream: []any{
				map[string]any{"data": map[string]any{"result": "first"}},
			},
		},
	}

	m.recordCapturedStubWithDelays(
		func() *stuber.Stub { return stub },
		false,
		[]time.Duration{100 * time.Millisecond},
	)

	require.Len(t, m.budgerigar.stubs, 1)

	resultStub := m.budgerigar.stubs[0]
	entry0 := resultStub.Output.Stream[0].(map[string]any)
	require.NotContains(t, entry0, "delay")
}

func TestRecordCapturedStubWithDelays_PartialDelays(t *testing.T) {
	t.Parallel()

	m := &grpcMockerForTest{
		budgerigar: &mockBudgerigar{},
	}

	stub := &stuber.Stub{
		Service: "test.Service",
		Method:  "TestMethod",
		Output: stuber.Output{
			Stream: []any{
				map[string]any{"data": map[string]any{"result": "first"}},
				map[string]any{"data": map[string]any{"result": "second"}},
				map[string]any{"data": map[string]any{"result": "third"}},
			},
		},
	}

	m.recordCapturedStubWithDelays(
		func() *stuber.Stub { return stub },
		true,
		[]time.Duration{100 * time.Millisecond}, // only first delay
	)

	require.Len(t, m.budgerigar.stubs, 1)

	resultStub := m.budgerigar.stubs[0]

	entry0 := resultStub.Output.Stream[0].(map[string]any)
	require.Equal(t, "100ms", entry0["delay"])

	entry1 := resultStub.Output.Stream[1].(map[string]any)
	require.NotContains(t, entry1, "delay")

	entry2 := resultStub.Output.Stream[2].(map[string]any)
	require.NotContains(t, entry2, "delay")
}

func TestRecordCapturedStubWithDelays_ZeroDelaysIgnored(t *testing.T) {
	t.Parallel()

	m := &grpcMockerForTest{
		budgerigar: &mockBudgerigar{},
	}

	stub := &stuber.Stub{
		Service: "test.Service",
		Method:  "TestMethod",
		Output: stuber.Output{
			Stream: []any{
				map[string]any{"data": map[string]any{"result": "first"}},
				map[string]any{"data": map[string]any{"result": "second"}},
			},
		},
	}

	m.recordCapturedStubWithDelays(
		func() *stuber.Stub { return stub },
		true,
		[]time.Duration{0, 0}, // all zero delays
	)

	require.Len(t, m.budgerigar.stubs, 1)

	resultStub := m.budgerigar.stubs[0]
	entry0 := resultStub.Output.Stream[0].(map[string]any)
	require.NotContains(t, entry0, "delay")
}

func TestRecordCapturedStubWithDelays_NilStreamItem(t *testing.T) {
	t.Parallel()

	m := &grpcMockerForTest{
		budgerigar: &mockBudgerigar{},
	}

	stub := &stuber.Stub{
		Service: "test.Service",
		Method:  "TestMethod",
		Output: stuber.Output{
			Stream: []any{
				nil,
				map[string]any{"data": map[string]any{"result": "second"}},
			},
		},
	}

	delays := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
	}

	m.recordCapturedStubWithDelays(
		func() *stuber.Stub { return stub },
		true,
		delays,
	)

	require.Len(t, m.budgerigar.stubs, 1)

	resultStub := m.budgerigar.stubs[0]
	entry1 := resultStub.Output.Stream[1].(map[string]any)
	require.Equal(t, "200ms", entry1["delay"])
}

func TestRecordCapturedStubWithDelays_NonMapStreamItem(t *testing.T) {
	t.Parallel()

	m := &grpcMockerForTest{
		budgerigar: &mockBudgerigar{},
	}

	stub := &stuber.Stub{
		Service: "test.Service",
		Method:  "TestMethod",
		Output: stuber.Output{
			Stream: []any{
				"not a map",
				map[string]any{"data": map[string]any{"result": "second"}},
			},
		},
	}

	delays := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
	}

	m.recordCapturedStubWithDelays(
		func() *stuber.Stub { return stub },
		true,
		delays,
	)

	require.Len(t, m.budgerigar.stubs, 1)

	resultStub := m.budgerigar.stubs[0]

	entry0, ok := resultStub.Output.Stream[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "100ms", entry0["delay"])

	entry1 := resultStub.Output.Stream[1].(map[string]any)
	require.Equal(t, "200ms", entry1["delay"])
}
