package app

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
)

func TestEffectStubKeepsNumbersAsQueriesSeeThem(t *testing.T) {
	t.Parallel()

	payload := map[string]any{
		"service": "catalog.CatalogService",
		"method":  "Search",
		"input":   map[string]any{"contains": map[string]any{"min_stock": json.Number("973")}},
		"output":  map[string]any{"data": map[string]any{"matched": json.Number("73")}},
	}

	stub, err := decodeEffectStub(payload)
	require.NoError(t, err)
	require.Equal(t, json.Number("973"), stub.Input.Contains["min_stock"])
}
