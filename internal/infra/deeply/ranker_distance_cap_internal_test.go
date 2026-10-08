package deeply

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDistanceCapsOversizedComparisons(t *testing.T) {
	t.Parallel()

	huge := strings.Repeat("a", 1<<20)
	other := strings.Repeat("b", 1<<20)

	start := time.Now()
	score := distance(huge, other)
	elapsed := time.Since(start)

	require.InDelta(t, 0.0, score, 1e-9, "oversized comparisons score as not similar")
	require.Less(t, elapsed, time.Second, "oversized comparison must not run the quadratic algorithm")
}

func TestRankMatchStaysFastOnLargePayloads(t *testing.T) {
	t.Parallel()

	expected := map[string]any{"field": strings.Repeat("x", 512)}
	actual := map[string]any{"field": strings.Repeat("y", 1<<20)}

	start := time.Now()

	for range 100 {
		RankMatch(expected, actual)
	}

	require.Less(t, time.Since(start), 2*time.Second, "ranking large payloads must stay cheap")
}

func TestDistanceStillScoresNormalStrings(t *testing.T) {
	t.Parallel()

	require.InDelta(t, 1.0, distance("gripmock", "gripmock"), 1e-9)
	require.Greater(t, distance("gripmock", "gripmocc"), 0.5)
	require.Greater(t, distance("gripmock", "gripmocc"), distance("gripmock", "totally-different"))
}

func TestDistanceExactlyAtCap(t *testing.T) {
	t.Parallel()

	side := 1
	for side*side <= maxDistanceCells {
		side *= 2
	}

	side /= 2

	within := distance(strings.Repeat("a", side), strings.Repeat("a", side-1)+"b")
	require.Greater(t, within, 0.0, "a comparison at the cap is still scored")
}
