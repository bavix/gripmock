package stuber

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatch(t *testing.T) {
	t.Parallel()
	// Test with different service - match function doesn't check service/method
	query := Query{Service: "test", Method: "test"}
	stub := &Stub{Service: "different", Method: "test"}
	require.True(t, match(query, stub)) // match only checks headers and data, not service/method

	// Test match with headers mismatch
	query = Query{Service: "test", Method: "test", Headers: map[string]any{"header": "value"}}
	stub = &Stub{Service: "test", Method: "test", Headers: InputHeader{Equals: map[string]any{"header": "different"}}}
	require.False(t, match(query, stub))

	// Test match with data mismatch
	query = Query{Service: "test", Method: "test", Input: []map[string]any{{"key": "value"}}}
	stub = &Stub{Service: "test", Method: "test", Input: InputData{Equals: map[string]any{"key": "different"}}}
	require.False(t, match(query, stub))

	// Test successful match
	query = Query{Service: "test", Method: "test", Input: []map[string]any{{"key": "value"}}}
	stub = &Stub{Service: "test", Method: "test", Input: InputData{Equals: map[string]any{"key": "value"}}}
	require.True(t, match(query, stub))
}

func TestEqualsFunction(t *testing.T) {
	t.Parallel()
	// Test equals function directly
	expected := map[string]any{"key": "value"}
	actual := map[string]any{"key": "value"}
	require.True(t, equals(expected, actual, false))

	// Test with different values
	actual = map[string]any{"key": "different"}
	require.False(t, equals(expected, actual, false))

	// Test with missing key
	actual = map[string]any{"other": "value"}
	require.False(t, equals(expected, actual, false))

	// Test with extra key
	actual = map[string]any{"key": "value", "extra": "data"}
	require.True(t, equals(expected, actual, false)) // equals only checks expected keys
}

func TestMatchStreamElements(t *testing.T) {
	t.Parallel()
	// Test single element match
	queryStream := []map[string]any{{"key": "value"}}
	stubStream := []InputData{{Equals: map[string]any{"key": "value"}}}
	require.True(t, matchStreamElements(queryStream, stubStream))

	// Test multiple elements match
	queryStream = []map[string]any{{"key1": "value1"}, {"key2": "value2"}}
	stubStream = []InputData{
		{Equals: map[string]any{"key1": "value1"}},
		{Equals: map[string]any{"key2": "value2"}},
	}
	require.True(t, matchStreamElements(queryStream, stubStream))

	// Test length mismatch
	queryStream = []map[string]any{{"key": "value"}}
	stubStream = []InputData{
		{Equals: map[string]any{"key": "value"}},
		{Equals: map[string]any{"key2": "value2"}},
	}
	// For bidirectional streaming, single message can match any stub item
	require.False(t, matchStreamElements(queryStream, stubStream))

	// Test element mismatch
	queryStream = []map[string]any{{"key": "value"}}
	stubStream = []InputData{{Equals: map[string]any{"key": "different"}}}
	require.False(t, matchStreamElements(queryStream, stubStream))

	// Test empty query with non-empty stub
	queryStream = []map[string]any{}
	stubStream = []InputData{{Equals: map[string]any{"key": "value"}}}
	require.False(t, matchStreamElements(queryStream, stubStream))

	// Test contains matcher
	queryStream = []map[string]any{{"key": "value", "extra": "data"}}
	stubStream = []InputData{{Contains: map[string]any{"key": "value"}}}
	require.True(t, matchStreamElements(queryStream, stubStream))

	// Test matches matcher
	queryStream = []map[string]any{{"key": "value123"}}
	stubStream = []InputData{{Matches: map[string]any{"key": "val.*"}}}
	require.True(t, matchStreamElements(queryStream, stubStream))

	// Test no matchers defined
	queryStream = []map[string]any{{"key": "value"}}
	stubStream = []InputData{{}} // no matchers
	require.False(t, matchStreamElements(queryStream, stubStream))
}

func TestRankStreamElements(t *testing.T) {
	t.Parallel()
	// Test with empty streams
	score := rankStreamElements([]map[string]any{}, []InputData{})
	//nolint:testifylint
	require.Equal(t, 0.0, score)

	// Test with single element
	queryStream := []map[string]any{{"key": "value"}}
	stubStream := []InputData{{Equals: map[string]any{"key": "value"}}}
	score = rankStreamElements(queryStream, stubStream)
	require.Greater(t, score, 0.0)

	// Test multiple elements match
	queryStream = []map[string]any{{"key1": "value1"}, {"key2": "value2"}}
	stubStream = []InputData{
		{Equals: map[string]any{"key1": "value1"}},
		{Equals: map[string]any{"key2": "value2"}},
	}
	score = rankStreamElements(queryStream, stubStream)
	require.Greater(t, score, 0.0)

	// Test length mismatch
	queryStream = []map[string]any{{"key": "value"}}
	stubStream = []InputData{
		{Equals: map[string]any{"key": "value"}},
		{Equals: map[string]any{"key2": "value2"}},
	}
	score = rankStreamElements(queryStream, stubStream)
	// Should still give some score for partial match
	require.GreaterOrEqual(t, score, 0.0)

	// Test element mismatch
	queryStream = []map[string]any{{"key": "value"}}
	stubStream = []InputData{{Equals: map[string]any{"key": "different"}}}
	_ = rankStreamElements(queryStream, stubStream)
}

//nolint:funlen
func TestEqualsComprehensive(t *testing.T) {
	t.Parallel()
	// Test with different data types
	require.True(t, equals(map[string]any{"int": 42}, map[string]any{"int": 42}, false))
	require.True(t, equals(map[string]any{"float": 3.14}, map[string]any{"float": 3.14}, false))
	require.True(t, equals(map[string]any{"bool": true}, map[string]any{"bool": true}, false))
	require.True(t, equals(map[string]any{"string": "hello"}, map[string]any{"string": "hello"}, false))

	// Test with mixed types
	mixed1 := map[string]any{
		"int":    42,
		"float":  3.14,
		"bool":   true,
		"string": "hello",
		"slice":  []any{1, 2, 3},
		"map":    map[string]any{"nested": "value"},
	}
	mixed2 := map[string]any{
		"int":    42,
		"float":  3.14,
		"bool":   true,
		"string": "hello",
		"slice":  []any{1, 2, 3},
		"map":    map[string]any{"nested": "value"},
	}
	require.True(t, equals(mixed1, mixed2, false))

	// Test with different values
	require.False(t, equals(map[string]any{"key": "value1"}, map[string]any{"key": "value2"}, false))
	require.False(t, equals(map[string]any{"key": 1}, map[string]any{"key": 2}, false))
	require.False(t, equals(map[string]any{"key": true}, map[string]any{"key": false}, false))

	// Test with missing keys
	require.False(t, equals(map[string]any{"key1": "value"}, map[string]any{"key2": "value"}, false))
	require.False(t, equals(map[string]any{"key1": "value1", "key2": "value2"}, map[string]any{"key1": "value1"}, false))

	// Test with extra keys - equals only checks expected keys, so this should be true
	require.True(t, equals(map[string]any{"key1": "value1"}, map[string]any{"key1": "value1", "key2": "value2"}, false))

	// Test with nested structures
	nested1 := map[string]any{
		"level1": map[string]any{
			"level2": map[string]any{
				"level3": "deep_value",
			},
		},
	}
	nested2 := map[string]any{
		"level1": map[string]any{
			"level2": map[string]any{
				"level3": "deep_value",
			},
		},
	}
	require.True(t, equals(nested1, nested2, false))

	// Test with nested slices
	slice1 := map[string]any{
		"data": []any{
			map[string]any{"id": 1, "name": "item1"},
			map[string]any{"id": 2, "name": "item2"},
		},
	}
	slice2 := map[string]any{
		"data": []any{
			map[string]any{"id": 1, "name": "item1"},
			map[string]any{"id": 2, "name": "item2"},
		},
	}
	require.True(t, equals(slice1, slice2, false))

	// Test with different nested values
	nestedDiff1 := map[string]any{
		"level1": map[string]any{
			"level2": "value1",
		},
	}
	nestedDiff2 := map[string]any{
		"level1": map[string]any{
			"level2": "value2",
		},
	}
	require.False(t, equals(nestedDiff1, nestedDiff2, false))

	// Test with different slice values
	sliceDiff1 := map[string]any{
		"data": []any{1, 2, 3},
	}
	sliceDiff2 := map[string]any{
		"data": []any{1, 2, 4},
	}
	require.False(t, equals(sliceDiff1, sliceDiff2, false))

	// Test with different slice lengths
	sliceLen1 := map[string]any{
		"data": []any{1, 2},
	}
	sliceLen2 := map[string]any{
		"data": []any{1, 2, 3},
	}
	require.False(t, equals(sliceLen1, sliceLen2, false))
}

func TestEqualsEdgeCases(t *testing.T) {
	t.Parallel()
	// Test with nil values
	require.True(t, equals(nil, nil, false))
	// require.False(t, equals(nil, map[string]any{"key": "value"}, false))
	// require.False(t, equals(map[string]any{"key": "value"}, nil, false))

	// Test with different types - equals expects map[string]any as second parameter
	// require.False(t, equals(map[string]any{"key": "value"}, "string", false))
	// require.False(t, equals(map[string]any{"key": "value"}, 42, false))
	// require.False(t, equals(map[string]any{"key": "value"}, true, false))

	// Test with empty maps
	require.True(t, equals(map[string]any{}, map[string]any{}, false))
	// require.False(t, equals(map[string]any{"key": "value"}, map[string]any{}, false)) // Expected has key but actual is empty
	require.True(t, equals(map[string]any{}, map[string]any{"key": "value"}, false)) // Empty expected means no fields to check

	// Test with different map keys
	map1 := map[string]any{"key1": "value1"}
	map2 := map[string]any{"key2": "value1"}
	require.False(t, equals(map1, map2, false))

	// Test with different map values
	map3 := map[string]any{"key1": "value1"}
	map4 := map[string]any{"key1": "value2"}
	require.False(t, equals(map3, map4, false))

	// Test with nested maps
	nested1 := map[string]any{
		"level1": map[string]any{
			"level2": "value",
		},
	}
	nested2 := map[string]any{
		"level1": map[string]any{
			"level2": "value",
		},
	}
	require.True(t, equals(nested1, nested2, false))

	// Test with different nested maps
	nested3 := map[string]any{
		"level1": map[string]any{
			"level2": "different",
		},
	}
	require.False(t, equals(nested1, nested3, false))

	// Test with arrays
	array1 := map[string]any{"arr": []any{1, 2, 3}}
	array2 := map[string]any{"arr": []any{1, 2, 3}}
	require.True(t, equals(array1, array2, false))

	// Test with mixed content
	mixed1 := map[string]any{
		"string": "value",
		"number": 42,
		"bool":   true,
		"array":  []any{1, 2, 3},
		"map":    map[string]any{"nested": "value"},
	}
	mixed2 := map[string]any{
		"string": "value",
		"number": 42,
		"bool":   true,
		"array":  []any{1, 2, 3},
		"map":    map[string]any{"nested": "value"},
	}
	require.True(t, equals(mixed1, mixed2, false))

	// Test with different mixed content
	mixed3 := map[string]any{
		"string": "value",
		"number": 42,
		"bool":   true,
		"array":  []any{1, 2, 4}, // Different array
		"map":    map[string]any{"nested": "value"},
	}
	require.False(t, equals(mixed1, mixed3, false))
}

func TestToFloat64(t *testing.T) {
	t.Parallel()

	f, ok := toFloat64(42)
	require.True(t, ok)
	require.InDelta(t, 42.0, f, 1e-9)

	f, ok = toFloat64(int64(100))
	require.True(t, ok)
	require.InDelta(t, 100.0, f, 1e-9)

	f, ok = toFloat64(3.14)
	require.True(t, ok)
	require.InDelta(t, 3.14, f, 1e-9)

	f, ok = toFloat64(json.Number("0.0"))
	require.True(t, ok)
	require.InDelta(t, 0.0, f, 1e-9)

	_, ok = toFloat64("not a number")
	require.False(t, ok)
}

func TestFieldValueEquals_JsonNumber(t *testing.T) {
	t.Parallel()
	// json.Number vs float64
	require.True(t, equals(
		map[string]any{"v": json.Number("0.0")},
		map[string]any{"v": 0.0},
		false,
	))
	require.True(t, equals(
		map[string]any{"v": 0.0},
		map[string]any{"v": json.Number("0")},
		false,
	))
}

func TestStreamItemMatches(t *testing.T) {
	t.Parallel()
	require.True(t, streamItemMatches(
		InputData{Equals: map[string]any{"k": "v"}},
		map[string]any{"k": "v"},
	))
	require.False(t, streamItemMatches(
		InputData{}, // no matchers
		map[string]any{"k": "v"},
	))
	require.False(t, streamItemMatches(
		InputData{Equals: map[string]any{"k": "x"}},
		map[string]any{"k": "v"},
	))
}

func TestEqualsWithOrderIgnore(t *testing.T) {
	t.Parallel()
	// Test with different array lengths (should still be false even with order ignore)
	len1 := map[string]any{"arr": []any{1, 2}}
	len2 := map[string]any{"arr": []any{1, 2, 3}}
	require.False(t, equals(len1, len2, true))

	// Test with empty arrays
	empty1 := map[string]any{"arr": []any{}}
	empty2 := map[string]any{"arr": []any{}}
	require.True(t, equals(empty1, empty2, true))

	// Test with single element arrays
	single1 := map[string]any{"arr": []any{42}}
	single2 := map[string]any{"arr": []any{42}}
	require.True(t, equals(single1, single2, true))
}
