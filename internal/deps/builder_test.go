package deps_test

import (
	"testing"

	"github.com/gripmock/environment"
	"github.com/gripmock/shutdown"
	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/deps"
)

func TestBuilder_Basic(t *testing.T) {
	t.Parallel()
	// Test basic builder functionality
	builder := deps.NewBuilder()
	require.NotNil(t, builder)
}

func TestBuilder_Empty(t *testing.T) {
	t.Parallel()
	// Test empty builder case
	builder := deps.NewBuilder()
	require.NotNil(t, builder)
	// Basic test to ensure builder can be created
}

func TestBuilder_Initialization(t *testing.T) {
	t.Parallel()
	// Test builder initialization
	builder := deps.NewBuilder()
	require.NotNil(t, builder)
	// Verify builder is properly initialized
}

func TestBuilder_WithDefaultConfig(t *testing.T) {
	t.Parallel()
	// Test builder with default config
	builder := deps.NewBuilder(deps.WithDefaultConfig())
	require.NotNil(t, builder)
}

func TestBuilder_WithConfig(t *testing.T) {
	t.Parallel()
	// Test builder with custom config
	config, _ := environment.New()
	builder := deps.NewBuilder(deps.WithConfig(config))
	require.NotNil(t, builder)
}

func TestBuilder_WithEnder(t *testing.T) {
	t.Parallel()
	// Test builder with custom ender
	ender := shutdown.New(nil)
	builder := deps.NewBuilder(deps.WithEnder(ender))
	require.NotNil(t, builder)
}

func TestBuilder_MultipleOptions(t *testing.T) {
	t.Parallel()
	// Test builder with multiple options
	config, _ := environment.New()
	ender := shutdown.New(nil)
	builder := deps.NewBuilder(deps.WithConfig(config), deps.WithEnder(ender))
	require.NotNil(t, builder)
}
