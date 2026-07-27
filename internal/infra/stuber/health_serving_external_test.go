package stuber_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func healthStatus(t *testing.T, res *stuber.Result) string {
	t.Helper()
	require.NotNil(t, res.Found())
	data, ok := res.Found().Output.Data.(map[string]any)
	require.True(t, ok)

	return data["status"].(string) //nolint:forcetypeassert
}

// Regression: after SetAlive the internal gripmock health Check stub must report
// SERVING — even when the stub was queried (NOT_SERVING) before readiness, which
// is exactly what the startup `check` polling does. A stale match here left the
// server reporting NOT_SERVING forever (broke every E2E/smoke health check).
func TestGripmockHealthBecomesServing(t *testing.T) {
	t.Parallel()

	b := stuber.NewBudgerigar()
	q := stuber.Query{
		Service: "grpc.health.v1.Health",
		Method:  "Check",
		Input:   []map[string]any{{"service": "gripmock"}},
	}

	// Poll before ready (mimics startup health polling).
	res, err := b.FindByQuery(q)
	require.NoError(t, err)
	require.Equal(t, "NOT_SERVING", healthStatus(t, res))

	b.SetAlive()

	res, err = b.FindByQuery(q)
	require.NoError(t, err)
	require.Equal(t, "SERVING", healthStatus(t, res), "health must flip to SERVING after SetAlive")
}
