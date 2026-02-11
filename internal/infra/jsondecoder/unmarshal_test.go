package jsondecoder_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/jsondecoder"
)

type demo struct {
	ID int `json:"id"`
}

func TestUnmarshalSlice(t *testing.T) {
	t.Parallel()

	inputs := [][]byte{
		[]byte(`{"id": 1}`),
		[]byte(`      {"id": 1}`),
		[]byte(`{"id": 1}      `),
		[]byte(`       [{"id": 1}]`),
		[]byte(`[{"id": 1}]`),
	}

	for _, input := range inputs {
		results := make([]demo, 0)

		err := jsondecoder.UnmarshalSlice(input, &results)

		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, 1, results[0].ID)
	}
}

func TestUnmarshalSlice_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("empty returns error", func(t *testing.T) {
		t.Parallel()

		var results []demo

		err := jsondecoder.UnmarshalSlice([]byte{}, &results)
		require.Error(t, err)
	})

	t.Run("single char returns error", func(t *testing.T) {
		t.Parallel()

		var results []demo

		err := jsondecoder.UnmarshalSlice([]byte("{"), &results)
		require.Error(t, err)
	})

	t.Run("whitespace only returns error", func(t *testing.T) {
		t.Parallel()

		var results []demo

		err := jsondecoder.UnmarshalSlice([]byte("  "), &results)
		require.Error(t, err)
	})

	t.Run("single object wrap to map string any", func(t *testing.T) {
		t.Parallel()

		var results []map[string]any

		err := jsondecoder.UnmarshalSlice([]byte(`{"name":"Bob"}`), &results)
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, "Bob", results[0]["name"])
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		t.Parallel()

		var results []demo

		err := jsondecoder.UnmarshalSlice([]byte(`{invalid`), &results)
		require.Error(t, err)
	})
}
