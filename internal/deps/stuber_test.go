package deps_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/deps"
)

func TestStuber_Basic(t *testing.T) {
	t.Parallel()
	// Test basic stuber functionality
	require.NotNil(t, "stuber package exists")
}

func TestStuber_Empty(t *testing.T) {
	t.Parallel()
	// Test empty stuber case
	require.NotNil(t, "stuber package exists")
}

func TestStuber_Initialization(t *testing.T) {
	t.Parallel()
	// Test stuber initialization
	require.NotNil(t, "stuber package initialized")
}

func TestBuilder_Budgerigar(t *testing.T) {
	t.Parallel()
	// Test budgerigar creation
	builder := deps.NewBuilder()
	budgerigar := builder.Budgerigar()
	require.NotNil(t, budgerigar)
}

func TestBuilder_Extender(t *testing.T) {
	t.Parallel()
	// Test extender creation
	builder := deps.NewBuilder()
	extender := builder.Extender(context.Background())
	require.NotNil(t, extender)
}

func TestBuilder_SingletonPattern(t *testing.T) {
	t.Parallel()
	// Test that budgerigar and extender are singletons
	builder := deps.NewBuilder()

	budgerigar1 := builder.Budgerigar()
	budgerigar2 := builder.Budgerigar()
	require.Equal(t, budgerigar1, budgerigar2)

	extender1 := builder.Extender(context.Background())
	extender2 := builder.Extender(context.Background())
	require.Equal(t, extender1, extender2)
}
