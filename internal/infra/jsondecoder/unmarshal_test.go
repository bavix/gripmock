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
