package yaml2json_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/pkg/yaml2json"
)

func TestConvertor(t *testing.T) {
	convertor := yaml2json.New()

	// see: https://bavix.github.io/uuid-ui/
	// 77465064-a0ce-48a3-b7e4-d50f88e55093 => d0ZQZKDOSKO35NUPiOVQkw==
	// e351220b-4847-42f5-8abb-c052b87ff2d4 => {"high":-773977811204288029,"low":-3102276763665777782}
	bytes, err := convertor.Execute("hello", []byte(`
yaml2json:
  base64: {{ uuid2base64 "77465064-a0ce-48a3-b7e4-d50f88e55093" }}
  highLow: {{ uuid2int64 "e351220b-4847-42f5-8abb-c052b87ff2d4" }}
  string: {{ string2base64 "hello world" }}
  bytes: {{ bytes "hello world" | bytes2base64 }}
`))

	expected := `{
  "yaml2json": {
    "base64": "d0ZQZKDOSKO35NUPiOVQkw==",
    "highLow": {"high":-773977811204288029,"low":-3102276763665777782},
	"string": "aGVsbG8gd29ybGQ=",
	"bytes": "aGVsbG8gd29ybGQ="
  }
}`

	require.NoError(t, err)
	require.JSONEq(t, expected, string(bytes))
}

func TestPanic2Error(t *testing.T) {
	_, err := yaml2json.New().Execute("hello", []byte(`
yaml2json:
  base64: {{ uuid2base64 "no-uuid" }}
`))

	require.Error(t, err)
}
