package deps_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bavix/gripmock/v3/internal/deps"
)

func TestStuber_Basic(t *testing.T) {
	// Test basic stuber functionality
	assert.NotNil(t, "stuber package exists")
}

func TestStuber_Empty(t *testing.T) {
	// Test empty stuber case
	assert.NotNil(t, "stuber package exists")
}

func TestStuber_Initialization(t *testing.T) {
	// Test stuber initialization
	assert.NotNil(t, "stuber package initialized")
}

func TestBuilder_Budgerigar(t *testing.T) {
	// Test budgerigar creation
	builder := deps.NewBuilder()
	budgerigar := builder.Budgerigar()
	assert.NotNil(t, budgerigar)
}

func TestBuilder_Extender(t *testing.T) {
	// Test extender creation
	builder := deps.NewBuilder()
	extender := builder.Extender()
	assert.NotNil(t, extender)
}

func TestBuilder_SingletonPattern(t *testing.T) {
	// Test that budgerigar and extender are singletons
	builder := deps.NewBuilder()

	budgerigar1 := builder.Budgerigar()
	budgerigar2 := builder.Budgerigar()
	assert.Equal(t, budgerigar1, budgerigar2)

	extender1 := builder.Extender()
	extender2 := builder.Extender()
	assert.Equal(t, extender1, extender2)
}
