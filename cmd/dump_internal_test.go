package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetchStubsPreservesLargeNumbersAsJSONNumber(t *testing.T) {
	t.Parallel()

	const large = "9223372036854775807"

	var gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{
				"service":"demo.Service",
				"method":"Ping",
				"input":{"equals":{"big":` + large + `}},
				"output":{"data":{"ok":true}}
			}
		]`))
	}))
	t.Cleanup(server.Close)

	stubs, err := fetchStubs(context.Background(), server.URL, "")
	require.NoError(t, err)
	require.Equal(t, "/api/stubs", gotPath)
	require.Len(t, stubs, 1)

	value, ok := stubs[0].Input.Equals["big"]
	require.True(t, ok)

	number, ok := value.(json.Number)
	require.True(t, ok, "expected json.Number, got %T", value)
	require.Equal(t, large, number.String())
}
