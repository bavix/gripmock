package stuber

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInputHasConditions(t *testing.T) {
	t.Parallel()

	require.False(t, inputHasConditions(InputData{}), "no matchers")
	require.False(t, inputHasConditions(InputData{Equals: map[string]any{}}), "empty equals is not a requirement")

	require.True(t, inputHasConditions(InputData{Equals: map[string]any{"a": 1}}))
	require.True(t, inputHasConditions(InputData{Glob: map[string]any{"a": "x*"}}))

	// AnyOf counts only when an alternative actually sets a matcher.
	require.False(t, inputHasConditions(InputData{AnyOf: []AnyOfElement{{}}}), "anyOf of empty alternatives imposes no requirement")
	require.False(t, inputHasConditions(InputData{AnyOf: []AnyOfElement{{Equals: map[string]any{}}}}), "anyOf of empty {} matcher")

	withReal := InputData{AnyOf: []AnyOfElement{{}, {Contains: map[string]any{"a": 1}}}}
	require.True(t, inputHasConditions(withReal), "one real alternative counts")
}

func TestAnyOfElementChecks(t *testing.T) {
	t.Parallel()

	// declares is nil-based: an empty {} still counts as a declaration.
	require.True(t, anyOfElementDeclares(AnyOfElement{Equals: map[string]any{}}))
	require.False(t, anyOfElementDeclares(AnyOfElement{}))

	// hasFields is len-based: an empty {} is not a field requirement.
	require.False(t, anyOfElementHasFields(AnyOfElement{Equals: map[string]any{}}))
	require.True(t, anyOfElementHasFields(AnyOfElement{Glob: map[string]any{"a": "x*"}}))
}
