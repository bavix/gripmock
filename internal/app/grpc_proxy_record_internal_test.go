package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func TestApplyStreamDelays_NilStub(t *testing.T) {
	t.Parallel()

	applyStreamDelays(nil, true, []time.Duration{100 * time.Millisecond})
}

// TestApplyStreamDelays_WithDelays verifies that delays[i] is attached to
// stream[i+1] (the message that arrives AFTER the measured gap), not to
// stream[i]. This matches the playback contract in handleArrayStreamData
// where stream[k].delay is consumed BEFORE stream[k] is sent.
func TestApplyStreamDelays_WithDelays(t *testing.T) {
	t.Parallel()

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
		100 * time.Millisecond, // between first and second
		200 * time.Millisecond, // between second and third
		300 * time.Millisecond, // would describe gap after third - dropped
	}

	applyStreamDelays(stub, true, delays)

	require.Len(t, stub.Output.Stream, 3)

	// stream[0]: no delay, sent immediately
	entry0 := stub.Output.Stream[0].(map[string]any) //nolint:forcetypeassert
	require.NotContains(t, entry0, "delay")
	require.Equal(t, map[string]any{"result": "first"}, entry0["data"])

	// stream[1]: delay 100ms before sending
	entry1 := stub.Output.Stream[1].(map[string]any) //nolint:forcetypeassert
	require.Equal(t, "100ms", entry1["delay"])
	require.Equal(t, map[string]any{"result": "second"}, entry1["data"])

	// stream[2]: delay 200ms before sending
	entry2 := stub.Output.Stream[2].(map[string]any) //nolint:forcetypeassert
	require.Equal(t, "200ms", entry2["delay"])
	require.Equal(t, map[string]any{"result": "third"}, entry2["data"])
}

func TestApplyStreamDelays_RecordDelayFalse(t *testing.T) {
	t.Parallel()

	stub := &stuber.Stub{
		Service: "test.Service",
		Method:  "TestMethod",
		Output: stuber.Output{
			Stream: []any{
				map[string]any{"data": map[string]any{"result": "first"}},
			},
		},
	}

	applyStreamDelays(stub, false, []time.Duration{100 * time.Millisecond})

	entry0 := stub.Output.Stream[0].(map[string]any) //nolint:forcetypeassert
	require.NotContains(t, entry0, "delay")
}

// TestApplyStreamDelays_PartialDelays verifies the off-by-one boundary: with
// delays[0] only, stream[1] should receive the delay and stream[2] should
// not.
func TestApplyStreamDelays_PartialDelays(t *testing.T) {
	t.Parallel()

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

	applyStreamDelays(stub, true, []time.Duration{100 * time.Millisecond}) // only first delay

	entry0 := stub.Output.Stream[0].(map[string]any) //nolint:forcetypeassert
	require.NotContains(t, entry0, "delay")

	entry1 := stub.Output.Stream[1].(map[string]any) //nolint:forcetypeassert
	require.Equal(t, "100ms", entry1["delay"])

	entry2 := stub.Output.Stream[2].(map[string]any) //nolint:forcetypeassert
	require.NotContains(t, entry2, "delay")
}

func TestApplyStreamDelays_ZeroDelaysIgnored(t *testing.T) {
	t.Parallel()

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

	applyStreamDelays(stub, true, []time.Duration{0, 0})

	// stream[1] would have a zero delay attached; both entries should not
	// have a delay key.
	entry0 := stub.Output.Stream[0].(map[string]any) //nolint:forcetypeassert
	require.NotContains(t, entry0, "delay")

	entry1 := stub.Output.Stream[1].(map[string]any) //nolint:forcetypeassert
	require.NotContains(t, entry1, "delay")
}

func TestApplyStreamDelays_NilStreamItem(t *testing.T) {
	t.Parallel()

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

	applyStreamDelays(stub, true, []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
	})

	// delays[0] targets stream[1] which is the second entry; it already
	// is a map and the delay should land there. delays[1] would target
	// stream[2] which is out of bounds.
	entry1 := stub.Output.Stream[1].(map[string]any) //nolint:forcetypeassert
	require.Equal(t, "100ms", entry1["delay"])
}

func TestApplyStreamDelays_NonMapStreamItem(t *testing.T) {
	t.Parallel()

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

	applyStreamDelays(stub, true, []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
	})

	// delays[0] targets stream[1] (a map). delays[1] is dropped (out of
	// bounds for stream length 2).
	entry1 := stub.Output.Stream[1].(map[string]any) //nolint:forcetypeassert
	require.Equal(t, "100ms", entry1["delay"])
}

func TestApplyStreamDelays_EmptyDelays(t *testing.T) {
	t.Parallel()

	stub := &stuber.Stub{
		Service: "test.Service",
		Method:  "TestMethod",
		Output: stuber.Output{
			Stream: []any{
				map[string]any{"data": map[string]any{"result": "first"}},
			},
		},
	}

	applyStreamDelays(stub, true, nil)
	applyStreamDelays(stub, true, []time.Duration{})

	entry0 := stub.Output.Stream[0].(map[string]any) //nolint:forcetypeassert
	require.NotContains(t, entry0, "delay")
}
