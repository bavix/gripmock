package stuber

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The runtime reads a stub in one shape: Output.Messages() for the response and
// Stub.Matchers() for the request. The raw wire fields belong to the encoder and to
// the accessors themselves; anything else reaching for them brings back the "which
// field is set" guessing this package moved away from.
func TestRuntimeReadsTheResponseThroughMessages(t *testing.T) {
	t.Parallel()

	allowed := map[string]struct{}{
		"stub.go": {},
		"dump.go": {},
	}

	entries, err := os.ReadDir(".")
	require.NoError(t, err)

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		if _, ok := allowed[name]; ok {
			continue
		}

		body, err := os.ReadFile(filepath.Clean(name))
		require.NoError(t, err)

		require.NotContains(t, string(body), "Output.Data",
			"%s must read the response through Output.Messages()", name)
		require.NotContains(t, string(body), "stub.Input,",
			"%s must read the request through Stub.Matchers()", name)
	}
}
