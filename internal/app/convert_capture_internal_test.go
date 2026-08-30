package app

import (
	"encoding/json"
	"testing"

	"github.com/bufbuild/protocompile"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

const captureProto = `
syntax = "proto3";
package capture;
message Order {
  string id = 1;
  optional int64 placed_at = 2;
  oneof delivery {
    string address = 3;
    string pickup_point = 4;
  }
  repeated string tags = 5;
}
`

//nolint:ireturn // protoreflect returns interfaces
func compileCaptureMessage(t *testing.T) protoreflect.MessageDescriptor {
	t.Helper()

	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			Accessor: protocompile.SourceAccessorFromMap(map[string]string{"capture.proto": captureProto}),
		}),
	}

	files, err := compiler.Compile(t.Context(), "capture.proto")
	require.NoError(t, err)

	return files[0].Messages().ByName("Order")
}

// A captured stub matches by the fields it lists, so it has to list the ones the
// caller left unset too: without "placed_at" the recording would also answer a
// request that does set it, which is a different call with a different answer.
func TestConvertRequestForCaptureKeepsUnsetSingularFields(t *testing.T) {
	t.Parallel()

	desc := compileCaptureMessage(t)
	msg := dynamicpb.NewMessage(desc)
	msg.Set(desc.Fields().ByName("id"), protoreflect.ValueOfString("order-1"))

	captured := convertRequestForCapture(msg, defaultConvertDepth)

	require.Equal(t, "order-1", captured["id"])
	require.Contains(t, captured, "placed_at")
	require.Contains(t, captured, "address")
	require.Contains(t, captured, "pickup_point")
	require.NotContains(t, captured, "tags")
}

func TestConvertToMapDropsUnsetSingularFields(t *testing.T) {
	t.Parallel()

	desc := compileCaptureMessage(t)
	msg := dynamicpb.NewMessage(desc)
	msg.Set(desc.Fields().ByName("id"), protoreflect.ValueOfString("order-1"))

	converted := convertToMapWithDepth(msg, defaultConvertDepth)

	require.Equal(t, "order-1", converted["id"])
	require.NotContains(t, converted, "placed_at")
	require.NotContains(t, converted, "address")
	require.NotContains(t, converted, "pickup_point")
}

func TestConvertRequestForCaptureKeepsSetValues(t *testing.T) {
	t.Parallel()

	desc := compileCaptureMessage(t)
	msg := dynamicpb.NewMessage(desc)
	msg.Set(desc.Fields().ByName("id"), protoreflect.ValueOfString("order-2"))
	msg.Set(desc.Fields().ByName("placed_at"), protoreflect.ValueOfInt64(1745081266))
	msg.Set(desc.Fields().ByName("address"), protoreflect.ValueOfString("Baker street"))

	captured := convertRequestForCapture(msg, defaultConvertDepth)

	require.Equal(t, json.Number("1745081266"), captured["placed_at"])
	require.Equal(t, "Baker street", captured["address"])
}
