package plugins

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	pkgplugins "github.com/bavix/gripmock/v3/pkg/plugintest"
)

func TestRegisterBuiltins(t *testing.T) {
	t.Parallel()

	// Arrange
	reg := pkgplugins.NewRegistry()

	// Act
	RegisterBuiltins(reg)
	funcs := reg.Funcs()

	// Assert
	require.Contains(t, funcs, "upper")
	require.Contains(t, funcs, "lower")
	require.Contains(t, funcs, "json")
	require.Contains(t, funcs, "add")
	require.Contains(t, funcs, "uuid")
}

func TestStringFuncs(t *testing.T) {
	t.Parallel()

	// Arrange & Act
	funcs := stringFuncs()

	// Assert
	require.Contains(t, funcs, "upper")
	require.Contains(t, funcs, "lower")
	require.Contains(t, funcs, "title")
	require.Contains(t, funcs, "join")
	require.Contains(t, funcs, "split")

	upper, ok := funcs["upper"].(func(string) string)
	require.True(t, ok)
	require.Equal(t, "HELLO", upper("hello"))

	lower, ok := funcs["lower"].(func(string) string)
	require.True(t, ok)
	require.Equal(t, "hello", lower("HELLO"))
}

func TestJsonFuncs(t *testing.T) {
	t.Parallel()

	// Arrange
	funcs := jsonFuncs()
	jsonFunc, ok := funcs["json"].(func(any) string)
	require.True(t, ok)

	input := map[string]any{"key": "value"}

	// Act
	result := jsonFunc(input)

	// Assert
	require.Contains(t, funcs, "json")
	require.Contains(t, result, "key")
	require.Contains(t, result, "value")
}

func TestFormatFuncs(t *testing.T) {
	t.Parallel()

	// Arrange
	funcs := formatFuncs()
	sprintf, ok := funcs["sprintf"].(func(string, ...any) string)
	require.True(t, ok)

	format := "hello %d"
	value := 123

	// Act
	result := sprintf(format, value)

	// Assert
	require.Contains(t, funcs, "sprintf")
	require.Contains(t, funcs, "str")
	require.Equal(t, "hello 123", result)
}

func TestNumberFuncs(t *testing.T) {
	t.Parallel()

	// Arrange
	funcs := numberFuncs()
	intFunc, ok1 := funcs["int"].(func(any) int)
	require.True(t, ok1)

	floatFunc, ok2 := funcs["float"].(func(any) float64)
	require.True(t, ok2)

	input := 42.5

	// Act
	intResult := intFunc(input)
	floatResult := floatFunc(input)

	// Assert
	require.Contains(t, funcs, "int")
	require.Contains(t, funcs, "int64")
	require.Contains(t, funcs, "float")
	require.Contains(t, funcs, "decimal")
	require.Equal(t, 42, intResult)
	require.InDelta(t, 42.5, floatResult, 0.001)
}

func TestArrayFuncs(t *testing.T) {
	t.Parallel()

	// Arrange
	funcs := arrayFuncs()
	extractFunc, ok := funcs["extract"].(func(any, any) any)
	require.True(t, ok)

	mapInput := map[string]any{"key": "value"}
	sliceInput := []any{"a", "b", "c"}

	// Act
	mapResult := extractFunc(mapInput, "key")
	sliceResult := extractFunc(sliceInput, 1)

	// Assert
	require.Contains(t, funcs, "extract")
	require.Equal(t, "value", mapResult)
	require.Equal(t, "b", sliceResult)
}

func TestCompareFuncs(t *testing.T) {
	t.Parallel()

	// Arrange
	funcs := compareFuncs()
	gt, ok1 := funcs["gt"].(func(any, any) bool)
	require.True(t, ok1)

	eq, ok2 := funcs["eq"].(func(any, any) bool)
	require.True(t, ok2)

	// Act
	gtResult1 := gt(5, 3)
	gtResult2 := gt(3, 5)
	eqResult1 := eq(5, 5)
	eqResult2 := eq(5, 3)

	// Assert
	require.Contains(t, funcs, "gt")
	require.Contains(t, funcs, "lt")
	require.Contains(t, funcs, "gte")
	require.Contains(t, funcs, "lte")
	require.Contains(t, funcs, "eq")
	require.True(t, gtResult1)
	require.False(t, gtResult2)
	require.True(t, eqResult1)
	require.False(t, eqResult2)
}

func TestMathFuncs(t *testing.T) {
	t.Parallel()

	// Arrange
	funcs := mathFuncs()
	round, ok1 := funcs["round"].(func(any) float64)
	require.True(t, ok1)

	add, ok2 := funcs["add"].(func(...any) float64)
	require.True(t, ok2)

	sub, ok3 := funcs["sub"].(func(...any) float64)
	require.True(t, ok3)

	mul, ok4 := funcs["mul"].(func(...any) float64)
	require.True(t, ok4)

	// Act
	roundResult1 := round(3.4)
	roundResult2 := round(3.6)
	addResult := add(3, 4, 3)
	subResult := sub(5, 3)
	mulResult := mul(3, 4)

	// Assert
	require.Contains(t, funcs, "round")
	require.Contains(t, funcs, "floor")
	require.Contains(t, funcs, "ceil")
	require.Contains(t, funcs, "add")
	require.Contains(t, funcs, "sub")
	require.Contains(t, funcs, "div")
	require.Contains(t, funcs, "mod")
	require.Contains(t, funcs, "sum")
	require.Contains(t, funcs, "mul")
	require.Contains(t, funcs, "avg")
	require.Contains(t, funcs, "min")
	require.Contains(t, funcs, "max")
	require.InDelta(t, 3.0, roundResult1, 0.001)
	require.InDelta(t, 4.0, roundResult2, 0.001)
	require.InDelta(t, 10.0, addResult, 0.001)
	require.InDelta(t, 2.0, subResult, 0.001)
	require.InDelta(t, 12.0, mulResult, 0.001)
}

func TestTimeFuncs(t *testing.T) {
	t.Parallel()

	funcs := timeFuncs()
	require.Contains(t, funcs, "now")
	require.Contains(t, funcs, "unix")
	require.Contains(t, funcs, "format")
}

func TestUuidFuncs(t *testing.T) {
	t.Parallel()

	funcs := uuidFuncMap()
	require.Contains(t, funcs, "uuid")

	uuidFunc, ok := funcs["uuid"].(func() string)
	require.True(t, ok)

	uuid1 := uuidFunc()
	uuid2 := uuidFunc()

	require.NotEqual(t, uuid1, uuid2)
	_, err := uuid.Parse(uuid1)
	require.NoError(t, err)
}

func TestEncodingFuncs(t *testing.T) {
	t.Parallel()

	funcs := encodingFuncs()
	require.Contains(t, funcs, "bytes")
	require.Contains(t, funcs, "string2base64")
	require.Contains(t, funcs, "bytes2base64")
	require.Contains(t, funcs, "uuid2base64")
	require.Contains(t, funcs, "uuid2bytes")
	require.Contains(t, funcs, "uuid2int64")

	bytesFunc, ok := funcs["bytes"].(func(string) []byte)
	require.True(t, ok)
	require.Equal(t, []byte("hello"), bytesFunc("hello"))

	str2b64, ok := funcs["string2base64"].(func(string) string)
	require.True(t, ok)
	require.Equal(t, "aGVsbG8=", str2b64("hello"))

	b2b64, ok := funcs["bytes2base64"].(func([]byte) string)
	require.True(t, ok)
	require.Equal(t, "aGVsbG8=", b2b64([]byte("hello")))

	id := uuid.New().String()
	u2b64, ok := funcs["uuid2base64"].(func(string) (string, error))
	require.True(t, ok)

	res, err := u2b64(id)
	require.NoError(t, err)
	require.NotEmpty(t, res)

	u2bytes, ok := funcs["uuid2bytes"].(func(string) ([]byte, error))
	require.True(t, ok)

	b, err := u2bytes(id)
	require.NoError(t, err)
	require.Len(t, b, 16)

	u2int64, ok := funcs["uuid2int64"].(func(string) (string, error))
	require.True(t, ok)

	s, err := u2int64(id)
	require.NoError(t, err)
	require.Contains(t, s, "high")
	require.Contains(t, s, "low")
}

func TestConvertToFloat64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    any
		expected float64
		ok       bool
	}{
		{"float64", 42.5, 42.5, true},
		{"float32", float32(42.5), 42.5, true},
		{"json.Number", json.Number("42.5"), 42.5, true},
		{"string", "42.5", 42.5, true},
		{"int", 42, 42.0, true},
		{"invalid string", "invalid", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange - test case data is already set up in tt

			// Act
			result, ok := convertToFloat64(tt.input)

			// Assert
			require.Equal(t, tt.ok, ok)

			if ok {
				require.InDelta(t, tt.expected, result, 0.001)
			}
		})
	}
}

func TestConvertToInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    any
		expected int
		ok       bool
	}{
		{"int", 42, 42, true},
		{"int64", int64(42), 42, true},
		{"float64", 42.5, 42, true},
		{"json.Number", json.Number("42"), 42, true},
		{"string", "42", 42, true},
		{"invalid string", "invalid", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange - test case data is already set up in tt

			// Act
			result, ok := convertToInt(tt.input)

			// Assert
			require.Equal(t, tt.ok, ok)

			if ok {
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestAdd(t *testing.T) {
	t.Parallel()

	// Arrange
	validArgs := []any{1, 2, 3, 4}
	invalidArgs := []any{"invalid"}

	// Act
	validResult := add(validArgs...)
	invalidResult := add(invalidArgs...)

	// Assert
	require.InDelta(t, 10.0, validResult, 0.001)
	require.InDelta(t, 0.0, invalidResult, 0.001)
}

func TestSubtract(t *testing.T) {
	t.Parallel()

	// Arrange
	validArgs := []any{5, 3}
	invalidArgs := []any{"invalid"}

	// Act
	validResult := subtract(validArgs...)
	emptyResult := subtract()
	invalidResult := subtract(invalidArgs...)

	// Assert
	require.InDelta(t, 2.0, validResult, 0.001)
	require.InDelta(t, 0.0, emptyResult, 0.001)
	require.InDelta(t, 0.0, invalidResult, 0.001)
}

func TestDivide(t *testing.T) {
	t.Parallel()

	// Arrange
	validArgs := []any{10, 5}
	zeroDivArgs := []any{10, 0}

	// Act
	validResult := divide(validArgs...)
	zeroDivResult := divide(zeroDivArgs...)
	emptyResult := divide()

	// Assert
	require.InDelta(t, 2.0, validResult, 0.001)
	require.InDelta(t, 10.0, zeroDivResult, 0.001) // Division by zero returns original value
	require.InDelta(t, 0.0, emptyResult, 0.001)
}

func TestModulo(t *testing.T) {
	t.Parallel()

	// Arrange
	validArgs := []any{10, 3}
	zeroModArgs := []any{10, 0}
	singleArg := []any{10}

	// Act
	validResult := modulo(validArgs...)
	zeroModResult := modulo(zeroModArgs...)
	singleResult := modulo(singleArg...)

	// Assert
	require.InDelta(t, 1.0, validResult, 0.001)
	require.InDelta(t, 0.0, zeroModResult, 0.001)
	require.InDelta(t, 0.0, singleResult, 0.001)
}

func TestSum(t *testing.T) {
	t.Parallel()

	// Arrange
	validArgs := []any{1, 2, 3, 4, 5}

	// Act
	validResult := sum(validArgs...)
	emptyResult := sum()

	// Assert
	require.InDelta(t, 15.0, validResult, 0.001)
	require.InDelta(t, 0.0, emptyResult, 0.001)
}

func TestProduct(t *testing.T) {
	t.Parallel()

	// Arrange
	validArgs := []any{1, 2, 3, 4, 5}

	// Act
	validResult := product(validArgs...)
	emptyResult := product()

	// Assert
	require.InDelta(t, 120.0, validResult, 0.001)
	require.InDelta(t, 1.0, emptyResult, 0.001)
}

func TestAverage(t *testing.T) {
	t.Parallel()

	// Arrange
	validArgs := []any{1, 2, 3, 4, 5}

	// Act
	validResult := average(validArgs...)
	emptyResult := average()

	// Assert
	require.InDelta(t, 3.0, validResult, 0.001)
	require.InDelta(t, 0.0, emptyResult, 0.001)
}

func TestMinValue(t *testing.T) {
	t.Parallel()

	// Arrange
	validArgs := []any{5, 3, 1, 4, 2}

	// Act
	validResult := minValue(validArgs...)
	emptyResult := minValue()

	// Assert
	require.InDelta(t, 1.0, validResult, 0.001)
	require.InDelta(t, 0.0, emptyResult, 0.001)
}

func TestMaxValue(t *testing.T) {
	t.Parallel()

	// Arrange
	validArgs := []any{1, 3, 5, 2, 4}

	// Act
	validResult := maxValue(validArgs...)
	emptyResult := maxValue()

	// Assert
	require.InDelta(t, 5.0, validResult, 0.001)
	require.InDelta(t, 0.0, emptyResult, 0.001)
}

func TestExtract(t *testing.T) {
	t.Parallel()

	// Arrange
	mapInput := map[string]any{"key": "value"}
	sliceInput := []any{"a", "b", "c"}
	validKey := "key"
	validIndex := 1
	invalidIndex := 10

	// Act
	mapResult := extract(mapInput, validKey)
	sliceResult := extract(sliceInput, validIndex)
	invalidResult := extract(sliceInput, invalidIndex)

	// Assert
	require.Equal(t, "value", mapResult)
	require.Equal(t, "b", sliceResult)
	require.Nil(t, invalidResult)
}

func TestExtractFromSlice(t *testing.T) {
	t.Parallel()

	// Arrange
	length := 3
	validIndex := 1
	invalidIndex := 10
	getter := func(i int) any {
		return []string{"a", "b", "c"}[i]
	}

	// Act
	validResult := extractFromSlice(length, validIndex, getter)
	invalidResult := extractFromSlice(length, invalidIndex, getter)

	// Assert
	require.Equal(t, "b", validResult)
	require.Nil(t, invalidResult)
}

func TestExtractFromObjects(t *testing.T) {
	t.Parallel()

	// Arrange
	items := []any{
		map[string]any{"id": 1, "name": "a"},
		map[string]any{"id": 2, "name": "b"},
	}
	key := "name"

	// Act
	result := extractFromObjects(items, key)

	// Assert
	require.IsType(t, []any{}, result)
	names, ok := result.([]any)
	require.True(t, ok)
	require.Len(t, names, 2)
	require.Contains(t, names, "a")
	require.Contains(t, names, "b")
}

func TestMinFloat(t *testing.T) {
	t.Parallel()

	// Arrange
	a1, b1 := 1.0, 2.0
	a2, b2 := 3.0, 2.0

	// Act
	result1 := minFloat(a1, b1)
	result2 := minFloat(a2, b2)

	// Assert
	require.InDelta(t, 1.0, result1, 0.001)
	require.InDelta(t, 2.0, result2, 0.001)
}

func TestMaxFloat(t *testing.T) {
	t.Parallel()

	// Arrange
	a1, b1 := 1.0, 2.0
	a2, b2 := 3.0, 2.0

	// Act
	result1 := maxFloat(a1, b1)
	result2 := maxFloat(a2, b2)

	// Assert
	require.InDelta(t, 2.0, result1, 0.001)
	require.InDelta(t, 3.0, result2, 0.001)
}

func TestTitleCase(t *testing.T) {
	t.Parallel()

	// Arrange
	input := "hello"

	// Act
	result := titleCase(input)

	// Assert
	require.Equal(t, "HELLO", result)
}

func TestBuiltinInfo(t *testing.T) {
	t.Parallel()

	// Act
	info := builtinInfo()

	// Assert
	require.Equal(t, "gripmock", info.Name)
	require.Equal(t, "builtin", info.Kind)
	require.Contains(t, info.Capabilities, "template-funcs")
}
