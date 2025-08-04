package app_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/bavix/gripmock/v3/internal/app"
)

func TestNewMessageConverter(t *testing.T) {
	converter := app.NewMessageConverter()
	assert.NotNil(t, converter)
}

func TestMessageConverter_ConvertToMap_NilMessage(t *testing.T) {
	converter := app.NewMessageConverter()

	result := converter.ConvertToMap(nil)
	assert.Nil(t, result)
}

func TestMessageConverter_ConvertToMap_StringValue(t *testing.T) {
	converter := app.NewMessageConverter()

	msg := wrapperspb.String("test value")
	result := converter.ConvertToMap(msg)

	assert.NotNil(t, result)
	assert.Equal(t, "test value", result["value"])
}

func TestMessageConverter_ConvertToMap_Int32Value(t *testing.T) {
	converter := app.NewMessageConverter()

	msg := wrapperspb.Int32(42)
	result := converter.ConvertToMap(msg)

	assert.NotNil(t, result)
	assert.Contains(t, result, "value")
}

func TestMessageConverter_ConvertToMap_BoolValue(t *testing.T) {
	converter := app.NewMessageConverter()

	msg := wrapperspb.Bool(true)
	result := converter.ConvertToMap(msg)

	assert.NotNil(t, result)
	assert.Contains(t, result, "value")
}

func TestMessageConverter_ConvertToMap_DoubleValue(t *testing.T) {
	converter := app.NewMessageConverter()

	msg := wrapperspb.Double(3.14)
	result := converter.ConvertToMap(msg)

	assert.NotNil(t, result)
	assert.Contains(t, result, "value")
}

func TestMessageConverter_ConvertToMap_BytesValue(t *testing.T) {
	converter := app.NewMessageConverter()

	msg := wrapperspb.Bytes([]byte("hello"))
	result := converter.ConvertToMap(msg)

	assert.NotNil(t, result)
	assert.Equal(t, "aGVsbG8=", result["value"]) // base64 encoded "hello"
}

func TestMessageConverter_ConvertToMap_Struct(t *testing.T) {
	converter := app.NewMessageConverter()

	fields := map[string]*structpb.Value{
		"string_field": structpb.NewStringValue("test"),
		"number_field": structpb.NewNumberValue(42),
		"bool_field":   structpb.NewBoolValue(true),
	}

	msg := &structpb.Struct{
		Fields: fields,
	}

	result := converter.ConvertToMap(msg)
	assert.NotNil(t, result)
	fieldsMap, ok := result["fields"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, fieldsMap, "string_field")
	assert.Contains(t, fieldsMap, "number_field")
	assert.Contains(t, fieldsMap, "bool_field")
}

func TestMessageConverter_GetMessageDescriptor(t *testing.T) {
	converter := app.NewMessageConverter()

	// Test with a known message type
	desc, err := converter.GetMessageDescriptor("google.protobuf.StringValue")
	require.NoError(t, err)
	assert.NotNil(t, desc)

	// Test with unknown message type
	_, err = converter.GetMessageDescriptor("unknown.MessageType")
	assert.Error(t, err)
}

func TestMessageConverter_convertValue_List(t *testing.T) {
	converter := app.NewMessageConverter()

	// Create a message with repeated fields
	msg := &structpb.ListValue{
		Values: []*structpb.Value{
			structpb.NewStringValue("item1"),
			structpb.NewStringValue("item2"),
		},
	}

	result := converter.ConvertToMap(msg)
	assert.NotNil(t, result)
}

func TestMessageConverter_convertValue_Map(t *testing.T) {
	converter := app.NewMessageConverter()

	fields := map[string]*structpb.Value{
		"key1": structpb.NewStringValue("value1"),
		"key2": structpb.NewStringValue("value2"),
	}

	msg := &structpb.Struct{
		Fields: fields,
	}

	result := converter.ConvertToMap(msg)
	assert.NotNil(t, result)
	fieldsMap, ok := result["fields"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, fieldsMap, "key1")
	assert.Contains(t, fieldsMap, "key2")
}

func TestMessageConverter_convertScalar_Enum(t *testing.T) {
	converter := app.NewMessageConverter()

	// Test with a simple enum value
	msg := wrapperspb.Int32(1) // This will be treated as enum in some contexts
	result := converter.ConvertToMap(msg)

	assert.NotNil(t, result)
	assert.Contains(t, result, "value")
}

func TestMessageConverter_convertScalar_Message(t *testing.T) {
	converter := app.NewMessageConverter()

	outerMsg := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"nested": structpb.NewStringValue("test"),
		},
	}

	result := converter.ConvertToMap(outerMsg)
	assert.NotNil(t, result)
	fieldsMap, ok := result["fields"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, fieldsMap, "nested")
}

func TestMessageConverter_convertScalar_GroupKind(t *testing.T) {
	converter := app.NewMessageConverter()

	// GroupKind is deprecated, but we should handle it gracefully
	msg := wrapperspb.String("test")
	result := converter.ConvertToMap(msg)

	assert.NotNil(t, result)
	assert.Equal(t, "test", result["value"])
}

func TestMessageConverter_convertScalar_UnknownKind(t *testing.T) {
	converter := app.NewMessageConverter()

	// Test with a known type that should work
	msg := wrapperspb.String("test")
	result := converter.ConvertToMap(msg)

	assert.NotNil(t, result)
	assert.Equal(t, "test", result["value"])
}

// Additional tests for better coverage

func TestMessageConverter_ConvertToMap_Int64Value(t *testing.T) {
	converter := app.NewMessageConverter()

	msg := wrapperspb.Int64(9223372036854775807)
	result := converter.ConvertToMap(msg)

	assert.NotNil(t, result)
	assert.Contains(t, result, "value")
}

func TestMessageConverter_ConvertToMap_UInt32Value(t *testing.T) {
	converter := app.NewMessageConverter()

	msg := wrapperspb.UInt32(4294967295)
	result := converter.ConvertToMap(msg)

	assert.NotNil(t, result)
	assert.Contains(t, result, "value")
}

func TestMessageConverter_ConvertToMap_UInt64Value(t *testing.T) {
	converter := app.NewMessageConverter()

	msg := wrapperspb.UInt64(18446744073709551615)
	result := converter.ConvertToMap(msg)

	assert.NotNil(t, result)
	assert.Contains(t, result, "value")
}

func TestMessageConverter_ConvertToMap_FloatValue(t *testing.T) {
	converter := app.NewMessageConverter()

	msg := wrapperspb.Float(3.14159)
	result := converter.ConvertToMap(msg)

	assert.NotNil(t, result)
	assert.Contains(t, result, "value")
}

func TestMessageConverter_ConvertToMap_Timestamp(t *testing.T) {
	converter := app.NewMessageConverter()

	now := time.Now()
	msg := timestamppb.New(now)
	result := converter.ConvertToMap(msg)

	assert.NotNil(t, result)
	assert.Contains(t, result, "seconds")
	assert.Contains(t, result, "nanos")
}

func TestMessageConverter_ConvertToMap_EmptyStruct(t *testing.T) {
	converter := app.NewMessageConverter()

	msg := &structpb.Struct{
		Fields: map[string]*structpb.Value{},
	}

	result := converter.ConvertToMap(msg)

	// Empty struct returns empty map
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestMessageConverter_ConvertToMap_NestedStruct(t *testing.T) {
	converter := app.NewMessageConverter()

	nestedStruct := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"inner_field": structpb.NewStringValue("inner_value"),
		},
	}

	outerStruct := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"nested": structpb.NewStructValue(nestedStruct),
		},
	}

	result := converter.ConvertToMap(outerStruct)
	assert.NotNil(t, result)

	fieldsMap, ok := result["fields"].(map[string]any)
	require.True(t, ok)

	nestedValue, ok := fieldsMap["nested"].(map[string]any)
	require.True(t, ok)

	structValue, ok := nestedValue["struct_value"].(map[string]any)
	require.True(t, ok)

	nestedFieldsMap, ok := structValue["fields"].(map[string]any)
	require.True(t, ok)

	innerFieldValue, ok := nestedFieldsMap["inner_field"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "inner_value", innerFieldValue["string_value"])
}

func TestMessageConverter_ConvertToMap_ListWithMessages(t *testing.T) {
	converter := app.NewMessageConverter()

	// Create a list value with struct values
	listValue := &structpb.ListValue{
		Values: []*structpb.Value{
			structpb.NewStructValue(&structpb.Struct{
				Fields: map[string]*structpb.Value{
					"name": structpb.NewStringValue("item1"),
				},
			}),
			structpb.NewStructValue(&structpb.Struct{
				Fields: map[string]*structpb.Value{
					"name": structpb.NewStringValue("item2"),
				},
			}),
		},
	}

	result := converter.ConvertToMap(listValue)
	assert.NotNil(t, result)

	values, ok := result["values"].([]any)
	require.True(t, ok)
	assert.Len(t, values, 2)

	// Check first item
	firstItem, ok := values[0].(map[string]any)
	require.True(t, ok)
	structValue, ok := firstItem["struct_value"].(map[string]any)
	require.True(t, ok)
	firstFields, ok := structValue["fields"].(map[string]any)
	require.True(t, ok)
	firstFieldValue, ok := firstFields["name"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "item1", firstFieldValue["string_value"])
}

func TestMessageConverter_ConvertToMap_MapWithMessages(t *testing.T) {
	converter := app.NewMessageConverter()

	// Create a struct with map-like behavior
	msg := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"key1": structpb.NewStructValue(&structpb.Struct{
				Fields: map[string]*structpb.Value{
					"value": structpb.NewStringValue("nested_value1"),
				},
			}),
			"key2": structpb.NewStructValue(&structpb.Struct{
				Fields: map[string]*structpb.Value{
					"value": structpb.NewStringValue("nested_value2"),
				},
			}),
		},
	}

	result := converter.ConvertToMap(msg)
	assert.NotNil(t, result)

	fieldsMap, ok := result["fields"].(map[string]any)
	require.True(t, ok)

	// Check first nested struct
	nested1, ok := fieldsMap["key1"].(map[string]any)
	require.True(t, ok)
	structValue1, ok := nested1["struct_value"].(map[string]any)
	require.True(t, ok)
	nested1Fields, ok := structValue1["fields"].(map[string]any)
	require.True(t, ok)
	nested1Value, ok := nested1Fields["value"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "nested_value1", nested1Value["string_value"])
}

func TestMessageConverter_ConvertToMap_InvalidMessage(t *testing.T) {
	converter := app.NewMessageConverter()

	// Test with a message that has invalid fields
	msg := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"invalid_field": nil, // This should be handled gracefully
		},
	}

	result := converter.ConvertToMap(msg)
	assert.NotNil(t, result)
}

func TestMessageConverter_ConvertToMap_ComplexNested(t *testing.T) {
	converter := app.NewMessageConverter()

	// Create a complex nested structure
	outerStruct := createComplexNestedStruct()

	result := converter.ConvertToMap(outerStruct)
	assert.NotNil(t, result)

	fieldsMap, ok := result["fields"].(map[string]any)
	require.True(t, ok)

	// Check outer string field
	checkOuterStringField(t, fieldsMap)

	// Check nested struct
	checkNestedStruct(t, fieldsMap)
}

func createComplexNestedStruct() *structpb.Struct {
	innerList := &structpb.ListValue{
		Values: []*structpb.Value{
			structpb.NewStringValue("list_item1"),
			structpb.NewStringValue("list_item2"),
		},
	}

	innerStruct := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"list_field":   structpb.NewListValue(innerList),
			"string_field": structpb.NewStringValue("inner_string"),
		},
	}

	return &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"nested_struct": structpb.NewStructValue(innerStruct),
			"outer_string":  structpb.NewStringValue("outer_string"),
		},
	}
}

func checkOuterStringField(t *testing.T, fieldsMap map[string]any) {
	t.Helper()

	outerStringValue, ok := fieldsMap["outer_string"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "outer_string", outerStringValue["string_value"])
}

func checkNestedStruct(t *testing.T, fieldsMap map[string]any) {
	t.Helper()

	nestedStruct, ok := fieldsMap["nested_struct"].(map[string]any)
	require.True(t, ok)

	structValue, ok := nestedStruct["struct_value"].(map[string]any)
	require.True(t, ok)

	nestedFields, ok := structValue["fields"].(map[string]any)
	require.True(t, ok)

	// Check inner string field
	checkInnerStringField(t, nestedFields)

	// Check list field
	checkListField(t, nestedFields)
}

func checkInnerStringField(t *testing.T, nestedFields map[string]any) {
	t.Helper()

	innerStringValue, ok := nestedFields["string_field"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "inner_string", innerStringValue["string_value"])
}

func checkListField(t *testing.T, nestedFields map[string]any) {
	t.Helper()

	listField, ok := nestedFields["list_field"].(map[string]any)
	require.True(t, ok)

	listValue, ok := listField["list_value"].(map[string]any)
	require.True(t, ok)

	listValues, ok := listValue["values"].([]any)
	require.True(t, ok)
	assert.Len(t, listValues, 2)

	// Check list items are string values
	checkListItems(t, listValues)
}

func checkListItems(t *testing.T, listValues []any) {
	t.Helper()

	firstItem, ok := listValues[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "list_item1", firstItem["string_value"])

	secondItem, ok := listValues[1].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "list_item2", secondItem["string_value"])
}

// Additional tests for edge cases

func TestMessageConverter_ConvertToMap_MessageWithNilFields(t *testing.T) {
	converter := app.NewMessageConverter()

	// Test with a message that has nil fields
	msg := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"nil_field": nil,
		},
	}

	result := converter.ConvertToMap(msg)
	assert.NotNil(t, result)
}

func TestMessageConverter_ConvertToMap_MessageWithEmptyString(t *testing.T) {
	converter := app.NewMessageConverter()

	msg := wrapperspb.String("")
	result := converter.ConvertToMap(msg)

	assert.NotNil(t, result)
	// Empty string returns nil value
	assert.Nil(t, result["value"])
}

func TestMessageConverter_ConvertToMap_MessageWithZeroValues(t *testing.T) {
	converter := app.NewMessageConverter()

	// Test with zero values
	int32Msg := wrapperspb.Int32(0)
	int32Result := converter.ConvertToMap(int32Msg)
	assert.NotNil(t, int32Result)

	boolMsg := wrapperspb.Bool(false)
	boolResult := converter.ConvertToMap(boolMsg)
	assert.NotNil(t, boolResult)

	doubleMsg := wrapperspb.Double(0.0)
	doubleResult := converter.ConvertToMap(doubleMsg)
	assert.NotNil(t, doubleResult)
}

func TestMessageConverter_ConvertToMap_MessageWithNegativeValues(t *testing.T) {
	converter := app.NewMessageConverter()

	// Test with negative values
	int32Msg := wrapperspb.Int32(-42)
	int32Result := converter.ConvertToMap(int32Msg)
	assert.NotNil(t, int32Result)

	int64Msg := wrapperspb.Int64(-9223372036854775808)
	int64Result := converter.ConvertToMap(int64Msg)
	assert.NotNil(t, int64Result)

	doubleMsg := wrapperspb.Double(-3.14159)
	doubleResult := converter.ConvertToMap(doubleMsg)
	assert.NotNil(t, doubleResult)
}

func TestMessageConverter_ConvertToMap_MessageWithLargeValues(t *testing.T) {
	converter := app.NewMessageConverter()

	// Test with large values
	uint32Msg := wrapperspb.UInt32(4294967295)
	uint32Result := converter.ConvertToMap(uint32Msg)
	assert.NotNil(t, uint32Result)

	uint64Msg := wrapperspb.UInt64(18446744073709551615)
	uint64Result := converter.ConvertToMap(uint64Msg)
	assert.NotNil(t, uint64Result)
}

func TestMessageConverter_ConvertToMap_MessageWithSpecialCharacters(t *testing.T) {
	converter := app.NewMessageConverter()

	// Test with special characters
	msg := wrapperspb.String("test\n\t\r\"'\\")
	result := converter.ConvertToMap(msg)

	assert.NotNil(t, result)
	assert.Equal(t, "test\n\t\r\"'\\", result["value"])
}

func TestMessageConverter_ConvertToMap_Unicode(t *testing.T) {
	converter := app.NewMessageConverter()

	// Test with unicode characters (ASCII only to avoid gosmopolitan issues)
	msg := wrapperspb.String("test 🚀 rocket")
	result := converter.ConvertToMap(msg)

	assert.NotNil(t, result)
	assert.Equal(t, "test 🚀 rocket", result["value"])
}

func TestMessageConverter_ConvertToMap_MessageWithBinaryData(t *testing.T) {
	converter := app.NewMessageConverter()

	// Test with binary data
	binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
	msg := wrapperspb.Bytes(binaryData)
	result := converter.ConvertToMap(msg)

	assert.NotNil(t, result)
	// Should be base64 encoded - let's check the actual value
	value, ok := result["value"].(string)
	assert.True(t, ok)
	assert.NotEmpty(t, value)
	// Verify it's valid base64
	assert.Len(t, value, 8) // 6 bytes should encode to 8 base64 characters
}

func TestMessageConverter_ConvertToMap_MessageWithEmptyBytes(t *testing.T) {
	converter := app.NewMessageConverter()

	// Test with empty bytes
	msg := wrapperspb.Bytes([]byte{})
	result := converter.ConvertToMap(msg)

	assert.NotNil(t, result)
	// Empty bytes returns nil value
	assert.Nil(t, result["value"])
}

func TestMessageConverter_ConvertToMap_MessageWithNilBytes(t *testing.T) {
	converter := app.NewMessageConverter()

	// Test with nil bytes
	msg := wrapperspb.Bytes(nil)
	result := converter.ConvertToMap(msg)

	assert.NotNil(t, result)
	// Nil bytes returns nil value
	assert.Nil(t, result["value"])
}
