package yaml2json_test

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/yaml2json"
)

func aliasBomb(levels, width int) []byte {
	var doc strings.Builder

	doc.WriteString("a0: &a0 [")

	for i := range width {
		if i > 0 {
			doc.WriteString(", ")
		}

		doc.WriteString("\"lol\"")
	}

	doc.WriteString("]\n")

	for level := 1; level < levels; level++ {
		doc.WriteString("a" + strconv.Itoa(level) + ": &a" + strconv.Itoa(level) + " [")

		for i := range width {
			if i > 0 {
				doc.WriteString(", ")
			}

			doc.WriteString("*a" + strconv.Itoa(level-1))
		}

		doc.WriteString("]\n")
	}

	return []byte(doc.String())
}

func TestAliasBombRejected(t *testing.T) {
	t.Parallel()

	convertor := yaml2json.New(nil)

	_, err := convertor.Execute(t.Context(), "bomb", aliasBomb(6, 9))
	require.ErrorIs(t, err, yaml2json.ErrAliasExpansion)
}

func TestDeepAliasBombRejectedQuickly(t *testing.T) {
	t.Parallel()

	convertor := yaml2json.New(nil)
	started := time.Now()

	_, err := convertor.Execute(t.Context(), "bomb", aliasBomb(9, 9))
	require.ErrorIs(t, err, yaml2json.ErrAliasExpansion)
	require.Less(t, time.Since(started), time.Second)
}

func TestAliasGuardAllowsRealisticAnchorReuse(t *testing.T) {
	t.Parallel()

	var doc strings.Builder

	doc.WriteString("defaults: &defaults\n  service: Gripmock\n  method: SayHello\n")
	doc.WriteString("stubs:\n")

	for i := range 200 {
		doc.WriteString("  - <<: *defaults\n    input:\n      equals:\n        n: " + strconv.Itoa(i) + "\n")
	}

	convertor := yaml2json.New(nil)

	out, err := convertor.Execute(t.Context(), "stubs", []byte(doc.String()))
	require.NoError(t, err)
	require.Contains(t, string(out), "SayHello")
}

func TestAliasGuardIgnoresGlobPatterns(t *testing.T) {
	t.Parallel()

	var doc strings.Builder

	doc.WriteString("stubs:\n")

	for range 5000 {
		doc.WriteString("  - input:\n      glob:\n        name: \"*abc\"\n")
	}

	convertor := yaml2json.New(nil)

	_, err := convertor.Execute(t.Context(), "globs", []byte(doc.String()))
	require.NoError(t, err)
}
