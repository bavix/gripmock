package yaml2json_test

import (
	"github.com/stretchr/testify/require"
	"github.com/tokopedia/gripmock/pkg/yaml2json"
	"testing"
)

func TestConvertor(t *testing.T) {
	convertor := yaml2json.New()

	bytes, err := convertor.Execute("hello", []byte(`
yaml2json:
  base64: {{ uuid2base64 "77465064-a0ce-48a3-b7e4-d50f88e55093" }}
  highLow: {{ uuid2highLow "e351220b-4847-42f5-8abb-c052b87ff2d4" }}
`))

	expected := `{
  "yaml2json": {
    "base64": "d0ZQZKDOSKO35NUPiOVQkw==",
    "highLow": {"high":-773977811204288029,"low":-3102276763665777782}
  }
}`
	require.NoError(t, err)
	require.JSONEq(t, expected, string(bytes))
}
