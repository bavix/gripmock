package app

import (
	"encoding/json"
	"testing"

	"github.com/bufbuild/protocompile"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

const convertProto = `
syntax = "proto3";
package convert;
message Nested { string label = 1; }
message Wide {
  uint64 big = 1;
  int64 negative = 2;
  uint32 small = 3;
  double ratio = 4;
  bytes blob = 5;
  string text = 6;
  repeated Nested items = 7;
  map<string, Nested> lookup = 8;
  repeated uint64 counters = 9;
}
`

//nolint:ireturn // protoreflect returns interfaces
func compileConvertMessage(t *testing.T) protoreflect.MessageDescriptor {
	t.Helper()

	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			Accessor: protocompile.SourceAccessorFromMap(map[string]string{"convert.proto": convertProto}),
		}),
	}

	files, err := compiler.Compile(t.Context(), "convert.proto")
	require.NoError(t, err)

	return files[0].Messages().ByName("Wide")
}

func TestConvertToMapKeepsLargeIntegersExact(t *testing.T) {
	t.Parallel()

	desc := compileConvertMessage(t)
	msg := dynamicpb.NewMessage(desc)

	msg.Set(desc.Fields().ByName("big"), protoreflect.ValueOfUint64(18446744073709551615))
	msg.Set(desc.Fields().ByName("negative"), protoreflect.ValueOfInt64(-9223372036854775808))
	msg.Set(desc.Fields().ByName("small"), protoreflect.ValueOfUint32(7))
	msg.Set(desc.Fields().ByName("ratio"), protoreflect.ValueOfFloat64(1.5))
	msg.Set(desc.Fields().ByName("blob"), protoreflect.ValueOfBytes([]byte{1, 2, 3}))
	msg.Set(desc.Fields().ByName("text"), protoreflect.ValueOfString("plain"))

	converted := convertToMap(msg)

	require.Equal(t, json.Number("18446744073709551615"), converted["big"])
	require.Equal(t, json.Number("-9223372036854775808"), converted["negative"])
	require.Equal(t, json.Number("7"), converted["small"])
	require.InDelta(t, 1.5, converted["ratio"], 0.0001)
	require.Equal(t, "AQID", converted["blob"])
	require.Equal(t, "plain", converted["text"])
}

func TestConvertToMapWalksRepeatedAndMapFields(t *testing.T) {
	t.Parallel()

	desc := compileConvertMessage(t)
	nested := desc.ParentFile().Messages().ByName("Nested")
	msg := dynamicpb.NewMessage(desc)

	items := msg.Mutable(desc.Fields().ByName("items")).List()

	for _, label := range []string{"first", "second"} {
		entry := dynamicpb.NewMessage(nested)

		entry.Set(nested.Fields().ByName("label"), protoreflect.ValueOfString(label))
		items.Append(protoreflect.ValueOfMessage(entry))
	}

	lookup := msg.Mutable(desc.Fields().ByName("lookup")).Map()
	stored := dynamicpb.NewMessage(nested)

	stored.Set(nested.Fields().ByName("label"), protoreflect.ValueOfString("mapped"))
	lookup.Set(protoreflect.ValueOfString("key").MapKey(), protoreflect.ValueOfMessage(stored))

	counters := msg.Mutable(desc.Fields().ByName("counters")).List()
	counters.Append(protoreflect.ValueOfUint64(1))
	counters.Append(protoreflect.ValueOfUint64(18446744073709551615))

	converted := convertToMap(msg)

	list, ok := converted["items"].([]any)
	require.True(t, ok)
	require.Len(t, list, 2)

	first, ok := list[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "first", first["label"])

	mapped, ok := converted["lookup"].(map[string]any)
	require.True(t, ok)

	entry, ok := mapped["key"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "mapped", entry["label"])

	numbers, ok := converted["counters"].([]any)
	require.True(t, ok)
	require.Equal(t, []any{json.Number("1"), json.Number("18446744073709551615")}, numbers)
}

func TestConvertMapNumericToStringNumberNormalizesStubData(t *testing.T) {
	t.Parallel()

	desc := compileConvertMessage(t)

	converted := convertMapNumericToStringNumber(map[string]any{
		"big":      float64(1 << 53),
		"small":    7,
		"ratio":    1.5,
		"text":     "plain",
		"counters": []any{1, 2},
		"items":    []any{map[string]any{"label": "first"}},
		"unknown":  "kept",
	}, desc)

	require.Equal(t, json.Number("9007199254740992"), converted["big"])
	require.Equal(t, json.Number("7"), converted["small"])
	require.Equal(t, json.Number("1.5"), converted["ratio"], "a double keeps its exact text form")
	require.Equal(t, "plain", converted["text"])
	require.Equal(t, "kept", converted["unknown"])
	require.Equal(t, []any{json.Number("1"), json.Number("2")}, converted["counters"])

	items, ok := converted["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
}
