package deps

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFeatures_Basic(t *testing.T) {
	// Test basic features functionality
	builder := NewBuilder()
	toggles := builder.toggles()
	assert.NotNil(t, toggles)
}

func TestFeatures_DefaultToggles(t *testing.T) {
	// Test default toggles
	builder := NewBuilder()
	toggles := builder.toggles()
	assert.NotNil(t, toggles)
}

func TestFeatures_WithConfig(t *testing.T) {
	// Test with config
	builder := NewBuilder()
	toggles := builder.toggles()
	assert.NotNil(t, toggles)
}
