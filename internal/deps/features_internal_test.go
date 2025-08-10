package deps

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFeatures_Basic(t *testing.T) {
	t.Parallel()

	// Test basic features functionality
	builder := NewBuilder()
	toggles := builder.toggles()
	require.NotNil(t, toggles)
}

func TestFeatures_DefaultToggles(t *testing.T) {
	t.Parallel()

	// Test default toggles
	builder := NewBuilder()
	toggles := builder.toggles()
	require.NotNil(t, toggles)
}

func TestFeatures_WithConfig(t *testing.T) {
	t.Parallel()

	// Test with config
	builder := NewBuilder()
	toggles := builder.toggles()
	require.NotNil(t, toggles)
}
