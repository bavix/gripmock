package deps_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/deps"
)

func TestBuilderBudgerigar(t *testing.T) {
	t.Parallel()

	builder := deps.NewBuilder()
	budgerigar := builder.Budgerigar()
	require.NotNil(t, budgerigar)
}

func TestBuilderExtender(t *testing.T) {
	t.Parallel()

	builder := deps.NewBuilder()
	extender := builder.Extender(t.Context())
	require.NotNil(t, extender)
}

func TestBuilderSingletonPattern(t *testing.T) {
	t.Parallel()

	builder := deps.NewBuilder()

	budgerigar1 := builder.Budgerigar()
	budgerigar2 := builder.Budgerigar()
	require.Equal(t, budgerigar1, budgerigar2)

	extender1 := builder.Extender(t.Context())
	extender2 := builder.Extender(t.Context())
	require.Equal(t, extender1, extender2)
}
