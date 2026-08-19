package stuber

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestWriteDumpKeepsEveryMatchingField(t *testing.T) {
	t.Parallel()

	code := codes.NotFound
	stub := &Stub{
		ID:       uuid.MustParse("11111111-2222-3333-4444-555555555555"),
		Service:  "svc.Service",
		Method:   "Method",
		Session:  "team-a",
		Priority: 7,
		Options:  StubOptions{Times: 3},
		Inputs:   []InputData{{Equals: map[string]any{"id": "1"}}},
		Headers:  InputHeader{Equals: map[string]any{"authorization": "Bearer x"}},
		Output:   Output{Code: &code, Error: "nope"},
		Effects:  []Effect{{Action: EffectActionDelete, ID: "abc"}},
		Source:   SourceRest,
	}

	var buf bytes.Buffer
	require.NoError(t, WriteDump(&buf, []*Stub{stub}, DumpFormatJSON))

	var records []map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &records))
	require.Len(t, records, 1)

	rec := records[0]
	require.Equal(t, "11111111-2222-3333-4444-555555555555", rec["id"])
	require.Equal(t, "svc.Service", rec["service"])
	require.Equal(t, "Method", rec["method"])
	require.Equal(t, "team-a", rec["session"])
	require.InDelta(t, 7.0, rec["priority"], 0.0001)
	require.Equal(t, map[string]any{"times": 3.0}, rec["options"])
	require.NotEmpty(t, rec["inputs"])
	require.NotEmpty(t, rec["headers"])
	require.NotEmpty(t, rec["effects"])
	require.Equal(t, map[string]any{"source": "rest"}, rec["_meta"])
}

func TestWriteDumpKeepsIntegersIntegral(t *testing.T) {
	t.Parallel()

	code := codes.PermissionDenied
	stub := &Stub{
		ID:       uuid.MustParse("11111111-2222-3333-4444-666666666666"),
		Service:  "svc.Service",
		Method:   "Method",
		Priority: 7,
		Options:  StubOptions{Times: 2},
		Input:    InputData{Equals: map[string]any{"id": "1"}},
		Output:   Output{Code: &code, Error: "nope", Data: map[string]any{"count": 3, "ratio": 1.5}},
	}

	var buf bytes.Buffer
	require.NoError(t, WriteDump(&buf, []*Stub{stub}, DumpFormatYAML))

	dumped := buf.String()
	require.Contains(t, dumped, "code: 7")
	require.NotContains(t, dumped, "code: 7.0")
	require.Contains(t, dumped, "times: 2")
	require.NotContains(t, dumped, "times: 2.0")
	require.Contains(t, dumped, "count: 3")
	require.Contains(t, dumped, "ratio: 1.5")

	var reloaded []*Stub
	require.NoError(t, yaml.Unmarshal(buf.Bytes(), &reloaded))
	require.Len(t, reloaded, 1)
	require.Equal(t, codes.PermissionDenied, *reloaded[0].Output.Code)
	require.Equal(t, 2, reloaded[0].Options.Times)
}

func TestWriteDumpKeepsUnsignedBeyondInt64(t *testing.T) {
	t.Parallel()

	stub := &Stub{
		ID:      uuid.MustParse("11111111-2222-3333-4444-777777777777"),
		Service: "svc.Service",
		Method:  "Method",
		Input:   InputData{Equals: map[string]any{"big": uint64(math.MaxUint64)}},
		Output:  Output{Data: map[string]any{"ok": true}},
	}

	for _, format := range []string{DumpFormatYAML, DumpFormatJSON} {
		var buf bytes.Buffer
		require.NoError(t, WriteDump(&buf, []*Stub{stub}, format))

		require.Containsf(t, buf.String(), "18446744073709551615", "format %s", format)
		require.NotContainsf(t, buf.String(), "e+19", "format %s", format)
	}
}

func TestWriteDumpOmitsEmptyMatcherBlocks(t *testing.T) {
	t.Parallel()

	stub := &Stub{
		Service: "svc.Service",
		Method:  "Method",
		Input:   InputData{Equals: map[string]any{"id": "1"}},
		Output:  Output{Data: map[string]any{"ok": true}},
	}

	var buf bytes.Buffer
	require.NoError(t, WriteDump(&buf, []*Stub{stub}, DumpFormatJSON))

	var records []map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &records))

	rec := records[0]
	require.NotContains(t, rec, "headers", "an empty header matcher must not be written")
	require.NotContains(t, rec, "session")
	require.NotContains(t, rec, "priority")
	require.NotContains(t, rec, "options")

	input, ok := rec["input"].(map[string]any)
	require.True(t, ok)
	require.NotContains(t, input, "contains", "null matcher kinds must be dropped")

	output, ok := rec["output"].(map[string]any)
	require.True(t, ok)
	require.NotContains(t, output, "headers")
	require.NotContains(t, output, "error", "an empty error means nothing without a status code")
}

func TestStubSchemaCopiesAgree(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "public", "schema", "stub.json"))
	require.NoError(t, err)

	var schema struct {
		OneOf []struct {
			Properties map[string]any `json:"properties"`
			Items      *struct {
				Properties map[string]any `json:"properties"`
			} `json:"items"`
		} `json:"oneOf"`
	}

	require.NoError(t, json.Unmarshal(raw, &schema))
	require.Len(t, schema.OneOf, 2)

	single := schema.OneOf[0].Properties
	require.NotEmpty(t, single)
	require.NotNil(t, schema.OneOf[1].Items)

	array := schema.OneOf[1].Items.Properties
	require.ElementsMatch(t, keysOf(single), keysOf(array), "the two stub shapes describe different fields")
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	return out
}

func TestOpenAPIStubCoversEngineFields(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "api", "api.yaml"))
	require.NoError(t, err)

	var spec struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]any `yaml:"properties"`
			} `yaml:"schemas"`
		} `yaml:"components"`
	}

	require.NoError(t, yaml.Unmarshal(raw, &spec))

	stub, ok := spec.Components.Schemas["Stub"]
	require.True(t, ok, "api.yaml lost its Stub schema")

	for _, field := range engineStubJSONFields(t) {
		require.Contains(t, stub.Properties, field,
			"api.yaml does not describe Stub.%s, so generated clients cannot use it", field)
	}
}

func engineStubJSONFields(t *testing.T) []string {
	t.Helper()

	typ := reflect.TypeFor[Stub]()
	fields := make([]string, 0, typ.NumField())

	for field := range typ.Fields() {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")

		if name == "" || name == "-" {
			continue
		}

		fields = append(fields, name)
	}

	return fields
}
