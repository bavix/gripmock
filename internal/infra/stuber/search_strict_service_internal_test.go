package stuber

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQueryStrictServicePreventsMethodFallback(t *testing.T) {
	t.Parallel()

	b := NewBudgerigar()
	b.PutMany(&Stub{Service: "svc.v1.Echo", Method: "Get", Input: InputData{Equals: map[string]any{"id": "1"}}})

	result, err := b.FindByQuery(Query{
		Service: "svc.v2.Echo",
		Method:  "Get",
		Input:   []map[string]any{{"id": "1"}},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Found())

	result, err = b.FindByQuery(Query{
		Service:       "svc.v2.Echo",
		Method:        "Get",
		StrictService: true,
		Input:         []map[string]any{{"id": "1"}},
	})
	require.ErrorIs(t, err, ErrStubNotFound)
	require.Nil(t, result)
}
