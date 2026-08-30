package app

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestConvertToMapRendersNestedWellKnownTypesAsProtoJSON(t *testing.T) {
	t.Parallel()

	value, err := structpb.NewStruct(map[string]any{"a": 1, "b": "x", "c": []any{true}})
	require.NoError(t, err)

	converted := convertToMapWithDepth(value, defaultConvertDepth)

	fields, ok := converted["fields"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, json.Number("1"), fields["a"])
	require.Equal(t, "x", fields["b"])
	require.Equal(t, []any{true}, fields["c"])
}

func TestConvertToMapSkipsUnsetOneofMembers(t *testing.T) {
	t.Parallel()

	set := convertToMapWithDepth(&structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "v"}}, defaultConvertDepth)
	require.Equal(t, map[string]any{"string_value": "v"}, set)

	empty := convertToMapWithDepth(&structpb.Value{}, defaultConvertDepth)
	require.Empty(t, empty)
}
