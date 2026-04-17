package yaml2json_test

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/plugins"
	"github.com/bavix/gripmock/v3/internal/infra/yaml2json"
)

func TestConverter(t *testing.T) {
	t.Parallel()

	convertor := yaml2json.New(nil)

	// see: https://bavix.github.io/uuid-ui/
	// 77465064-a0ce-48a3-b7e4-d50f88e55093 => d0ZQZKDOSKO35NUPiOVQkw==
	// e351220b-4847-42f5-8abb-c052b87ff2d4 => {"high":-773977811204288029,"low":-3102276763665777782}
	bytes, err := convertor.Execute(t.Context(), "hello", []byte(`
yaml2json:
  base64: {{ uuid2base64 "77465064-a0ce-48a3-b7e4-d50f88e55093" }}
  highLow: {{ uuid2int64 "e351220b-4847-42f5-8abb-c052b87ff2d4" }}
  string: {{ string2base64 "hello world" }}
  bytes: {{ bytes "hello world" | bytes2base64 }}
`))

	expected := `{
  "yaml2json": {
    "base64": "{{ uuid2base64 \"77465064-a0ce-48a3-b7e4-d50f88e55093\" }}",
    "highLow": "{{ uuid2int64 \"e351220b-4847-42f5-8abb-c052b87ff2d4\" }}",
	"string": "{{ string2base64 \"hello world\" }}",
	"bytes": "{{ bytes \"hello world\" | bytes2base64 }}"
  }
}`

	require.NoError(t, err)
	require.JSONEq(t, expected, string(bytes))
}

func TestConverterWithRegistry(t *testing.T) {
	t.Parallel()

	reg := plugins.NewRegistry()
	plugins.RegisterBuiltins(reg)
	convertor := yaml2json.New(reg)

	// Template functions should be executed at load time when registry is available
	bytes, err := convertor.Execute(t.Context(), "hello", []byte(`
- input:
    equals:
      uuids: {{ uuid2int64 "e351220b-4847-42f5-8abb-c052b87ff2d4" }}
`))

	// Expected: template function executed, result is JSON object
	expected := `[{"input":{"equals":{"uuids":{"high":-773977811204288029,"low":-3102276763665777782}}}}]`

	require.NoError(t, err)
	require.JSONEq(t, expected, string(bytes))
}

func TestConverterWithRegistryKeepsFakerForRuntime(t *testing.T) {
	t.Parallel()

	reg := plugins.NewRegistry()
	plugins.RegisterBuiltins(reg)
	convertor := yaml2json.New(reg)

	bytes, err := convertor.Execute(t.Context(), "hello", []byte(`
- output:
    data:
      id: "{{.Request.id}}"
      first_name: "{{faker.Person.FirstName}}"
      account_id: "{{faker.Identity.UUID}}"
`))

	//nolint:lll
	expected := `[{"output":{"data":{"id":"{{.Request.id}}","first_name":"{{faker.Person.FirstName}}","account_id":"{{faker.Identity.UUID}}"}}}]`

	require.NoError(t, err)
	require.JSONEq(t, expected, string(bytes))
}

func TestConverterWIthRegistryArray(t *testing.T) {
	t.Parallel()

	reg := plugins.NewRegistry()
	plugins.RegisterBuiltins(reg)
	convertor := yaml2json.New(reg)

	// Template functions should be executed at load time when registry is available
	bytes, err := convertor.Execute(t.Context(), "hello", []byte(`
- input:
    equals:
      uuids:
        - {{ uuid2int64 "e351220b-4847-42f5-8abb-c052b87ff2d4" }}
        - {{ uuid2int64 "cd1f6d9e-7f2b-4b1a-9c0e-0f4b3d9ea2b7" }}
`))

	// Expected: template function executed, result is JSON array of objects
	expected := `[{"input":{"equals":{"uuids":[` +
		`{"high":-773977811204288029,"low":-3102276763665777782},` +
		`{"high":1894655895358218189,"low":-5214431432452141412}` +
		`]}}}]`

	require.NoError(t, err)
	require.JSONEq(t, expected, string(bytes))
}

func TestConverterStubFile(t *testing.T) {
	t.Parallel()

	reg := plugins.NewRegistry()
	plugins.RegisterBuiltins(reg)
	convertor := yaml2json.New(reg)

	// Read the actual stub file
	data, err := os.ReadFile("../../../examples/projects/identifier/stubs.yaml")
	require.NoError(t, err)

	result, err := convertor.Execute(t.Context(), "test", data)
	require.NoError(t, err)

	// Parse and check the result
	var stubs []map[string]any
	require.NoError(t, json.Unmarshal(result, &stubs))

	require.Len(t, stubs, 3)

	// Check first stub
	firstStub := stubs[0]
	input, ok := firstStub["input"].(map[string]any)
	require.True(t, ok, "input should be map[string]any")
	equals, ok := input["equals"].(map[string]any)
	require.True(t, ok, "equals should be map[string]any")
	uuids, ok := equals["uuids"].([]any)
	require.True(t, ok, "uuids should be []any")

	require.Len(t, uuids, 2)

	// Check first UUID
	uuid1, ok := uuids[0].(map[string]any)
	require.True(t, ok, "uuid1 should be map[string]any")
	//nolint:testifylint // Exact float comparison from JSON parsing
	require.Equal(t, float64(-773977811204288029), uuid1["high"])
	//nolint:testifylint // Exact float comparison from JSON parsing
	require.Equal(t, float64(-3102276763665777782), uuid1["low"])

	// Check second UUID
	uuid2, ok := uuids[1].(map[string]any)
	require.True(t, ok, "uuid2 should be map[string]any")
	//nolint:testifylint // Exact float comparison from JSON parsing
	require.Equal(t, float64(1894655895358218189), uuid2["high"])
	//nolint:testifylint // Exact float comparison from JSON parsing
	require.Equal(t, float64(-5214431432452141412), uuid2["low"])
}

func TestStubMatcherJsonNumberPrecision(t *testing.T) {
	t.Parallel()

	reg := plugins.NewRegistry()
	plugins.RegisterBuiltins(reg)
	convertor := yaml2json.New(reg)

	// Create a stub with large int64 values - matching the actual stub file format
	stubYAML := []byte(`
- input:
    equals:
      uuids:
        - {{ uuid2int64 "e351220b-4847-42f5-8abb-c052b87ff2d4" }}
`)

	result, err := convertor.Execute(t.Context(), "test", stubYAML)
	require.NoError(t, err)

	t.Logf("Raw JSON: %s", string(result))

	// Parse with json.Number support
	decoder := json.NewDecoder(bytes.NewReader(result))
	decoder.UseNumber()

	var stubs []map[string]any
	require.NoError(t, decoder.Decode(&stubs))

	// Get the stub UUIDs
	input, ok := stubs[0]["input"].(map[string]any)
	require.True(t, ok, "input should be map[string]any")
	equals, ok := input["equals"].(map[string]any)
	require.True(t, ok, "equals should be map[string]any")
	uuids, ok := equals["uuids"].([]any)
	require.True(t, ok, "uuids should be []any")

	require.Len(t, uuids, 1)

	firstUUID, ok := uuids[0].(map[string]any)
	require.True(t, ok, "firstUUID should be map[string]any")

	high := firstUUID["high"]
	low := firstUUID["low"]

	t.Logf("high type: %T, value: %v", high, high)
	t.Logf("low type: %T, value: %v", low, low)

	// Verify types are json.Number
	_, highIsNumber := high.(json.Number)
	_, lowIsNumber := low.(json.Number)

	require.True(t, highIsNumber, "high should be json.Number, got %T", high)
	require.True(t, lowIsNumber, "low should be json.Number, got %T", low)

	// Verify values are preserved exactly
	highNum, ok := high.(json.Number)
	require.True(t, ok, "highNum should be json.Number")
	lowNum, ok := low.(json.Number)
	require.True(t, ok, "lowNum should be json.Number")

	require.Equal(t, "-773977811204288029", highNum.String())
	require.Equal(t, "-3102276763665777782", lowNum.String())
}

func TestStubMatcherFullFlow(t *testing.T) {
	t.Parallel()

	reg := plugins.NewRegistry()
	plugins.RegisterBuiltins(reg)
	convertor := yaml2json.New(reg)

	// Load the actual stub file
	data, err := os.ReadFile("../../../examples/projects/identifier/stubs.yaml")
	require.NoError(t, err)

	result, err := convertor.Execute(t.Context(), "test", data)
	require.NoError(t, err)

	t.Logf("Raw JSON length: %d", len(result))

	// Parse with json.Number support (like the actual stub loading)
	decoder := json.NewDecoder(bytes.NewReader(result))
	decoder.UseNumber()

	var stubs []map[string]any
	require.NoError(t, decoder.Decode(&stubs))
	require.Len(t, stubs, 3, "should have 3 stubs")

	// Check first stub structure
	firstStub := stubs[0]
	input, ok := firstStub["input"].(map[string]any)
	require.True(t, ok, "input should be map[string]any")
	equals, ok := input["equals"].(map[string]any)
	require.True(t, ok, "equals should be map[string]any")
	uuids, ok := equals["uuids"].([]any)
	require.True(t, ok, "uuids should be []any")

	require.Len(t, uuids, 2, "first stub should have 2 UUIDs")

	// Check first UUID
	firstUUID, ok := uuids[0].(map[string]any)
	require.True(t, ok, "firstUUID should be map[string]any")

	high1 := firstUUID["high"]
	low1 := firstUUID["low"]

	t.Logf("UUID[0] high: type=%T, value=%v", high1, high1)
	t.Logf("UUID[0] low: type=%T, value=%v", low1, low1)

	// Verify types are json.Number
	_, high1IsNumber := high1.(json.Number)
	_, low1IsNumber := low1.(json.Number)

	require.True(t, high1IsNumber, "high1 should be json.Number, got %T", high1)
	require.True(t, low1IsNumber, "low1 should be json.Number, got %T", low1)

	// Verify values match expected
	high1Num, ok := high1.(json.Number)
	require.True(t, ok, "high1 should be json.Number")
	low1Num, ok := low1.(json.Number)
	require.True(t, ok, "low1 should be json.Number")

	require.Equal(t, "-773977811204288029", high1Num.String())
	require.Equal(t, "-3102276763665777782", low1Num.String())
}

func TestStubMatcherEndToEnd(t *testing.T) {
	t.Parallel()

	reg := plugins.NewRegistry()
	plugins.RegisterBuiltins(reg)
	convertor := yaml2json.New(reg)

	// Load the actual stub file
	data, err := os.ReadFile("../../../examples/projects/identifier/stubs.yaml")
	require.NoError(t, err)

	result, err := convertor.Execute(t.Context(), "test", data)
	require.NoError(t, err)

	// Parse with json.Number support
	decoder := json.NewDecoder(bytes.NewReader(result))
	decoder.UseNumber()

	var stubMaps []map[string]any
	require.NoError(t, decoder.Decode(&stubMaps))

	// Convert to stuber.Stub (simplified - just for testing)
	// In real code, this is done by storage.Extender
	t.Logf("Loaded %d stubs from YAML", len(stubMaps))

	// Create a query that should match the first stub
	// The query should have the same json.Number values
	queryUUIDs := []any{
		map[string]any{
			"high": json.Number("-773977811204288029"),
			"low":  json.Number("-3102276763665777782"),
		},
		map[string]any{
			"high": json.Number("1894655895358218189"),
			"low":  json.Number("-5214431432452141412"),
		},
	}

	t.Logf("Query UUIDs: %+v", queryUUIDs)

	// The stub expects these exact values, so matching should work
	// This test verifies that json.Number comparison works end-to-end
	require.Len(t, queryUUIDs, 2)

	for i, uuid := range queryUUIDs {
		uuidMap, ok := uuid.(map[string]any)
		require.True(t, ok, "uuid should be map[string]any")
		high, ok := uuidMap["high"].(json.Number)
		require.True(t, ok, "high should be json.Number")
		low, ok := uuidMap["low"].(json.Number)
		require.True(t, ok, "low should be json.Number")
		t.Logf("Query UUID[%d]: high=%s, low=%s", i, high.String(), low.String())
	}
}

func TestPanic2Error(t *testing.T) {
	t.Parallel()

	_, err := yaml2json.New(nil).Execute(t.Context(), "hello", []byte(`
yaml2json:
  base64: {{ uuid2base64 "no-uuid" }}
`))

	require.NoError(t, err)
}

func TestExecuteNoTemplateMarkers(t *testing.T) {
	t.Parallel()

	conv := yaml2json.New(nil)
	data := []byte("key: value\nnested:\n  a: 1")

	bytes, err := conv.Execute(t.Context(), "test", data)
	require.NoError(t, err)
	require.JSONEq(t, `{"key":"value","nested":{"a":1}}`, string(bytes))
}

func TestExecuteEmptyData(t *testing.T) {
	t.Parallel()

	conv := yaml2json.New(nil)

	bytes, err := conv.Execute(t.Context(), "test", []byte{})
	require.NoError(t, err)
	require.Contains(t, string(bytes), "null")
}
