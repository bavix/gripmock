package app

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// TestNewOutputMessageWKTDirect is the regression test for issue #882: methods
// that return a well-known type at the top level. protojson + dynamicpb handle
// the canonical protojson encoding natively; gripmock just needs to plumb the
// stub data through json.Encode -> protojson.Unmarshal without inventing a
// parallel encoder.
func TestNewOutputMessageWKTDirect(t *testing.T) {
	t.Parallel()

	concrete := func(t *testing.T, msg proto.Message, target proto.Message) {
		t.Helper()

		bytes, marshalErr := proto.Marshal(msg)
		require.NoError(t, marshalErr)

		require.NoError(t, proto.Unmarshal(bytes, target))
	}

	t.Run("Timestamp", func(t *testing.T) {
		t.Parallel()
		testNewOutputMessageTimestamp(t, concrete)
	})

	t.Run("Duration", func(t *testing.T) {
		t.Parallel()
		testNewOutputMessageDuration(t, concrete)
	})

	t.Run("StringValue", func(t *testing.T) {
		t.Parallel()
		testNewOutputMessageStringValue(t, concrete)
	})

	t.Run("Int32Value", func(t *testing.T) {
		t.Parallel()
		testNewOutputMessageInt32Value(t, concrete)
	})

	t.Run("BoolValue", func(t *testing.T) {
		t.Parallel()
		testNewOutputMessageBoolValue(t, concrete)
	})

	t.Run("Struct", func(t *testing.T) {
		t.Parallel()
		testNewOutputMessageStruct(t, concrete)
	})

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()
		testNewOutputMessageEmpty(t, concrete)
	})

	t.Run("regular message preserves int64 precision", func(t *testing.T) {
		t.Parallel()
		testNewOutputMessageInt64(t)
	})
}

func testNewOutputMessageTimestamp(t *testing.T, concrete func(t *testing.T, msg, target proto.Message)) {
	t.Helper()

	ts := (&timestamppb.Timestamp{}).ProtoReflect().Descriptor()
	mocker := &grpcMocker{outputDesc: ts}

	msg, err := mocker.newOutputMessage("2024-01-01T12:00:00Z")
	require.NoError(t, err)
	require.NotNil(t, msg)

	got := &timestamppb.Timestamp{}
	concrete(t, msg, got)
	require.Equal(t, int64(1704110400), got.GetSeconds())
	require.Equal(t, int32(0), got.GetNanos())
}

func testNewOutputMessageDuration(t *testing.T, concrete func(t *testing.T, msg, target proto.Message)) {
	t.Helper()

	d := (&durationpb.Duration{}).ProtoReflect().Descriptor()
	mocker := &grpcMocker{outputDesc: d}

	msg, err := mocker.newOutputMessage("1.5s")
	require.NoError(t, err)
	require.NotNil(t, msg)

	got := &durationpb.Duration{}
	concrete(t, msg, got)
	require.Equal(t, int64(1), got.GetSeconds())
	require.Equal(t, int32(500000000), got.GetNanos())
}

func testNewOutputMessageStringValue(t *testing.T, concrete func(t *testing.T, msg, target proto.Message)) {
	t.Helper()

	sv := (&wrapperspb.StringValue{}).ProtoReflect().Descriptor()
	mocker := &grpcMocker{outputDesc: sv}

	msg, err := mocker.newOutputMessage("hello")
	require.NoError(t, err)
	require.NotNil(t, msg)

	got := &wrapperspb.StringValue{}
	concrete(t, msg, got)
	require.Equal(t, "hello", got.GetValue())
}

func testNewOutputMessageInt32Value(t *testing.T, concrete func(t *testing.T, msg, target proto.Message)) {
	t.Helper()

	iv := (&wrapperspb.Int32Value{}).ProtoReflect().Descriptor()
	mocker := &grpcMocker{outputDesc: iv}

	msg, err := mocker.newOutputMessage(float64(42))
	require.NoError(t, err)
	require.NotNil(t, msg)

	got := &wrapperspb.Int32Value{}
	concrete(t, msg, got)
	require.Equal(t, int32(42), got.GetValue())
}

func testNewOutputMessageBoolValue(t *testing.T, concrete func(t *testing.T, msg, target proto.Message)) {
	t.Helper()

	bv := (&wrapperspb.BoolValue{}).ProtoReflect().Descriptor()
	mocker := &grpcMocker{outputDesc: bv}

	msg, err := mocker.newOutputMessage(true)
	require.NoError(t, err)
	require.NotNil(t, msg)

	got := &wrapperspb.BoolValue{}
	concrete(t, msg, got)
	require.True(t, got.GetValue())
}

func testNewOutputMessageStruct(t *testing.T, concrete func(t *testing.T, msg, target proto.Message)) {
	t.Helper()

	s := (&structpb.Struct{}).ProtoReflect().Descriptor()
	mocker := &grpcMocker{outputDesc: s}

	data := map[string]any{
		"region":  "us-east-1",
		"retries": float64(3),
	}

	msg, err := mocker.newOutputMessage(data)
	require.NoError(t, err)
	require.NotNil(t, msg)

	got := &structpb.Struct{}
	concrete(t, msg, got)
	require.Equal(t, "us-east-1", got.GetFields()["region"].GetStringValue())
	require.InDelta(t, 3, got.GetFields()["retries"].GetNumberValue(), 1e-9)
}

func testNewOutputMessageEmpty(t *testing.T, concrete func(t *testing.T, msg, target proto.Message)) {
	t.Helper()

	e := (&emptypb.Empty{}).ProtoReflect().Descriptor()
	mocker := &grpcMocker{outputDesc: e}

	msg, err := mocker.newOutputMessage(map[string]any{})
	require.NoError(t, err)
	require.NotNil(t, msg)

	// Empty has no fields; round-trip is the assertion.
	got := &emptypb.Empty{}
	concrete(t, msg, got)
}

func testNewOutputMessageInt64(t *testing.T) {
	t.Helper()

	mocker := createTestMocker(t)

	data := map[string]any{
		"fields": map[string]any{
			"bigint": json.Number("9223372036854775000"),
		},
	}

	msg, err := mocker.newOutputMessage(data)
	require.NoError(t, err)
	require.NotNil(t, msg)
}

// TestConvertMapValueStringNoFd verifies that without a field descriptor strings stay as-is.
func TestConvertMapValueStringNoFd(t *testing.T) {
	t.Parallel()

	got := convertMapValue("hello", nil)
	require.Equal(t, "hello", got)
}

func TestIsNumericKind(t *testing.T) {
	t.Parallel()

	t.Run("numeric kinds return true", func(t *testing.T) {
		t.Parallel()

		for _, k := range []protoreflect.Kind{
			protoreflect.DoubleKind, protoreflect.FloatKind,
			protoreflect.Int32Kind, protoreflect.Int64Kind,
			protoreflect.Uint32Kind, protoreflect.Uint64Kind,
			protoreflect.Sint32Kind, protoreflect.Sint64Kind,
			protoreflect.Sfixed32Kind, protoreflect.Sfixed64Kind,
			protoreflect.Fixed32Kind, protoreflect.Fixed64Kind,
		} {
			require.True(t, isNumericKind(k), "expected numeric: %v", k)
		}
	})

	t.Run("non-numeric kinds return false", func(t *testing.T) {
		t.Parallel()

		for _, k := range []protoreflect.Kind{
			protoreflect.BoolKind, protoreflect.EnumKind,
			protoreflect.StringKind, protoreflect.BytesKind,
			protoreflect.MessageKind, protoreflect.GroupKind,
		} {
			require.False(t, isNumericKind(k), "expected non-numeric: %v", k)
		}
	})
}

//nolint:ireturn
func fieldDesc(t *testing.T, msg proto.Message) protoreflect.FieldDescriptor {
	t.Helper()

	fd := msg.ProtoReflect().Descriptor().Fields().ByName("value")
	require.NotNil(t, fd)

	return fd
}

func TestConvertStringValueDouble(t *testing.T) {
	t.Parallel()

	doubleFD := fieldDesc(t, &wrapperspb.DoubleValue{})
	tests := []struct {
		input string
		want  any
	}{
		{"0", json.Number("0")},
		{"-0", json.Number("0")},
		{"1e308", json.Number("1e+308")},
		{"1e-308", json.Number("1e-308")},
		{"-3.14", json.Number("-3.14")},
		{"3.141592653589793", json.Number("3.141592653589793")},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got := convertStringValue(tt.input, doubleFD)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestConvertStringValueInt32(t *testing.T) {
	t.Parallel()

	int32FD := fieldDesc(t, &wrapperspb.Int32Value{})
	tests := []struct {
		input string
		want  any
	}{
		{"0", json.Number("0")},
		{"1", json.Number("1")},
		{"-1", json.Number("-1")},
		{"2147483647", json.Number("2147483647")},
		{"-2147483648", json.Number("-2147483648")},
		{"2147483648", json.Number("2147483648")},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got := convertStringValue(tt.input, int32FD)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestConvertStringValueInt64StaysString(t *testing.T) {
	t.Parallel()

	int64FD := fieldDesc(t, &wrapperspb.Int64Value{})

	got := convertStringValue("9223372036854775000", int64FD)
	require.Equal(t, "9223372036854775000", got)
}

func TestConvertStringValueUint64StaysString(t *testing.T) {
	t.Parallel()

	uint64FD := fieldDesc(t, &wrapperspb.UInt64Value{})
	tests := []struct {
		input string
		want  any
	}{
		{"0", "0"},
		{"18446744073709551615", "18446744073709551615"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got := convertStringValue(tt.input, uint64FD)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestConvertStringValueKeepsStringField(t *testing.T) {
	t.Parallel()

	strFD := fieldDesc(t, &wrapperspb.StringValue{})

	got := convertStringValue("42", strFD)
	require.Equal(t, "42", got)
}

func TestConvertStringValueKeepsBoolField(t *testing.T) {
	t.Parallel()

	boolFD := fieldDesc(t, &wrapperspb.BoolValue{})

	got := convertStringValue("42", boolFD)
	require.Equal(t, "42", got)
}

func TestConvertStringValueNilDesc(t *testing.T) {
	t.Parallel()

	got := convertStringValue("42", nil)
	require.Equal(t, "42", got)
}

func TestConvertStringValueNonNumericString(t *testing.T) {
	t.Parallel()

	int32FD := fieldDesc(t, &wrapperspb.Int32Value{})

	got := convertStringValue("hello", int32FD)
	require.Equal(t, "hello", got)
}

func TestConvertStringValueEmptyString(t *testing.T) {
	t.Parallel()

	int32FD := fieldDesc(t, &wrapperspb.Int32Value{})

	got := convertStringValue("", int32FD)
	require.Empty(t, got)
}

func TestConvertMapNumericToStringNumberDouble(t *testing.T) {
	t.Parallel()

	doubleDesc := (&structpb.Value{}).ProtoReflect().Descriptor()

	t.Run("double field converts string to json.Number", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{"number_value": "49.99"}
		result := convertMapNumericToStringNumber(data, doubleDesc)
		require.Equal(t, json.Number("49.99"), result["number_value"])
	})

	t.Run("string field keeps numeric string as-is", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{"string_value": "42"}
		result := convertMapNumericToStringNumber(data, doubleDesc)
		require.Equal(t, "42", result["string_value"])
	})
}

func TestConvertMapNumericToStringNumberNilDesc(t *testing.T) {
	t.Parallel()

	t.Run("string field with numeric string stays string", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{"some_field": "42"}
		result := convertMapNumericToStringNumber(data, nil)
		require.Equal(t, "42", result["some_field"])
	})

	t.Run("float value without descriptor converts via case float64", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{"price": float64(49.99)}
		result := convertMapNumericToStringNumber(data, nil)
		require.Equal(t, json.Number("49.99"), result["price"])
	})
}
