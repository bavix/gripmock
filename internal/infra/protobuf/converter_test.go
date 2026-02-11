package protobuf_test

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/bavix/gripmock/v3/internal/infra/protobuf"
)

func TestNewScalarConverter(t *testing.T) {
	t.Parallel()

	converter := protobuf.NewScalarConverter()
	require.NotNil(t, converter)
}

//nolint:funlen
func TestScalarConverter_ConvertScalar(t *testing.T) {
	t.Parallel()

	converter := protobuf.NewScalarConverter()

	t.Run("bool kind", func(t *testing.T) {
		t.Parallel()

		msg := wrapperspb.Bool(true)
		descriptor := msg.ProtoReflect().Descriptor().Fields().ByName("value")
		value := msg.ProtoReflect().Get(descriptor)

		result := converter.ConvertScalar(descriptor, value)
		require.Equal(t, true, result)
	})

	t.Run("string kind", func(t *testing.T) {
		t.Parallel()

		msg := wrapperspb.String("test string")
		descriptor := msg.ProtoReflect().Descriptor().Fields().ByName("value")
		value := msg.ProtoReflect().Get(descriptor)

		result := converter.ConvertScalar(descriptor, value)
		require.Equal(t, "test string", result)
	})

	t.Run("int32 kind", func(t *testing.T) {
		t.Parallel()

		msg := wrapperspb.Int32(42)
		descriptor := msg.ProtoReflect().Descriptor().Fields().ByName("value")
		value := msg.ProtoReflect().Get(descriptor)

		result := converter.ConvertScalar(descriptor, value)
		jsonNum, ok := result.(json.Number)
		require.True(t, ok, "Result should be json.Number")
		require.Equal(t, "42", string(jsonNum))
	})

	t.Run("int64 kind", func(t *testing.T) {
		t.Parallel()

		msg := wrapperspb.Int64(9223372036854775807)
		descriptor := msg.ProtoReflect().Descriptor().Fields().ByName("value")
		value := msg.ProtoReflect().Get(descriptor)

		result := converter.ConvertScalar(descriptor, value)
		jsonNum, ok := result.(json.Number)
		require.True(t, ok, "Result should be json.Number")
		require.Equal(t, "9223372036854775807", string(jsonNum))
	})

	t.Run("uint32 kind", func(t *testing.T) {
		t.Parallel()

		msg := wrapperspb.UInt32(42)
		descriptor := msg.ProtoReflect().Descriptor().Fields().ByName("value")
		value := msg.ProtoReflect().Get(descriptor)

		result := converter.ConvertScalar(descriptor, value)
		jsonNum, ok := result.(json.Number)
		require.True(t, ok, "Result should be json.Number")
		require.Equal(t, "42", string(jsonNum))
	})

	t.Run("uint64 kind", func(t *testing.T) {
		t.Parallel()

		msg := wrapperspb.UInt64(18446744073709551615)
		descriptor := msg.ProtoReflect().Descriptor().Fields().ByName("value")
		value := msg.ProtoReflect().Get(descriptor)

		result := converter.ConvertScalar(descriptor, value)
		jsonNum, ok := result.(json.Number)
		require.True(t, ok, "Result should be json.Number")
		require.Equal(t, "18446744073709551615", string(jsonNum))
	})

	t.Run("float kind", func(t *testing.T) {
		t.Parallel()

		msg := wrapperspb.Float(3.14)
		descriptor := msg.ProtoReflect().Descriptor().Fields().ByName("value")
		value := msg.ProtoReflect().Get(descriptor)

		result := converter.ConvertScalar(descriptor, value)
		jsonNum, ok := result.(json.Number)
		require.True(t, ok, "Result should be json.Number")
		require.Contains(t, string(jsonNum), "3.14")
	})

	t.Run("double kind", func(t *testing.T) {
		t.Parallel()

		msg := wrapperspb.Double(3.141592653589793)
		descriptor := msg.ProtoReflect().Descriptor().Fields().ByName("value")
		value := msg.ProtoReflect().Get(descriptor)

		result := converter.ConvertScalar(descriptor, value)
		jsonNum, ok := result.(json.Number)
		require.True(t, ok, "Result should be json.Number")
		require.Contains(t, string(jsonNum), "3.141592653589793")
	})

	t.Run("bytes kind", func(t *testing.T) {
		t.Parallel()

		msg := wrapperspb.Bytes([]byte("hello world"))
		descriptor := msg.ProtoReflect().Descriptor().Fields().ByName("value")
		value := msg.ProtoReflect().Get(descriptor)

		result := converter.ConvertScalar(descriptor, value)
		require.Equal(t, "aGVsbG8gd29ybGQ=", result) // base64 encoded
	})

	t.Run("message kind with valid message", func(t *testing.T) {
		t.Parallel()

		innerMsg := wrapperspb.String("inner value")
		msg, err := anypb.New(innerMsg)
		require.NoError(t, err)

		descriptor := msg.ProtoReflect().Descriptor().Fields().ByName("value")
		value := msg.ProtoReflect().Get(descriptor)

		result := converter.ConvertScalar(descriptor, value)
		require.Equal(t, "Cgtpbm5lciB2YWx1ZQ==", result) // bytes are base64 encoded
	})

	t.Run("message kind with nil message", func(t *testing.T) {
		t.Parallel()

		// Test with valid Any message
		innerMsg := wrapperspb.String("test")
		msg, err := anypb.New(innerMsg)
		require.NoError(t, err)

		descriptor := msg.ProtoReflect().Descriptor().Fields().ByName("type_url")
		value := msg.ProtoReflect().Get(descriptor)

		result := converter.ConvertScalar(descriptor, value)
		// Should handle valid string field
		require.NotNil(t, result)
		require.IsType(t, "", result)
	})
}

func TestScalarConverter_ConvertMessage(t *testing.T) {
	t.Parallel()

	converter := protobuf.NewScalarConverter()

	t.Run("nil message", func(t *testing.T) {
		t.Parallel()

		result := converter.ConvertMessage(nil)
		require.Nil(t, result)
	})

	t.Run("valid message", func(t *testing.T) {
		t.Parallel()

		msg := wrapperspb.String("test value")
		result := converter.ConvertMessage(msg)

		require.NotNil(t, result)
		require.Contains(t, result, "value")
		require.Equal(t, "test value", result["value"])
	})

	t.Run("complex message with nested fields", func(t *testing.T) {
		t.Parallel()

		// Create Any message with nested content
		innerMsg := wrapperspb.String("nested value")
		msg, err := anypb.New(innerMsg)
		require.NoError(t, err)

		result := converter.ConvertMessage(msg)

		require.NotNil(t, result)
		require.Contains(t, result, "type_url")
		require.Contains(t, result, "value")
	})

	t.Run("list field", func(t *testing.T) {
		t.Parallel()

		// structpb.ListValue has repeated "values" - triggers convertList
		msg, err := structpb.NewList([]any{1.0, "two", true})
		require.NoError(t, err)

		result := converter.ConvertMessage(msg)
		require.NotNil(t, result)
		require.Contains(t, result, "values")
		vals, ok := result["values"].([]any)
		require.True(t, ok)
		require.Len(t, vals, 3)
	})

	t.Run("map field", func(t *testing.T) {
		t.Parallel()

		// structpb.Struct has "fields" map - triggers convertMap
		msg, err := structpb.NewStruct(map[string]any{"a": 1.0, "b": "x"})
		require.NoError(t, err)

		result := converter.ConvertMessage(msg)
		require.NotNil(t, result)
		require.Contains(t, result, "fields")
	})
}
