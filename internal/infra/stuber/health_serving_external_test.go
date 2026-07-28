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
// SERVING and take precedence over ANY user stub that tries to override
// gripmock -> NOT_SERVING (the "protected" contract). Without the internal-stub
// priority, the ranking tiebreak let the user stub win, so the server reported
// NOT_SERVING forever (broke every E2E/smoke health check).
func TestGripmockHealthBecomesServing(t *testing.T) {
	t.Parallel()

	b := stuber.NewBudgerigar()

	// A user stub that tries to shadow gripmock -> NOT_SERVING (as examples do).
	b.PutMany(&stuber.Stub{
		Service: "grpc.health.v1.Health", Method: "Check",
		Input:  stuber.InputData{Equals: map[string]any{"service": "gripmock"}},
		Output: stuber.Output{Data: map[string]any{"status": "NOT_SERVING"}},
	})

	base := stuber.Query{
		Service: "grpc.health.v1.Health",
		Method:  "Check",
		Input:   []map[string]any{{"service": "gripmock"}},
	}
	// The gRPC health service marks its query to reach internal stubs.
	healthQuery := stuber.WithInternalStubs(base)

	// Before ready: the internal stub wins for the health path → NOT_SERVING.
	res, err := b.FindByQuery(healthQuery)
	require.NoError(t, err)
	require.Equal(t, "NOT_SERVING", healthStatus(t, res))

	b.SetAlive()

	// After ready: internal stub flips to SERVING for the health path.
	res, err = b.FindByQuery(healthQuery)
	require.NoError(t, err)
	require.Equal(t, "SERVING", healthStatus(t, res), "internal health stub must report SERVING")

	// A normal (user-facing) query must NEVER see the internal stub: it resolves
	// to the competing user stub instead, keeping internal state hidden.
	res, err = b.FindByQuery(base)
	require.NoError(t, err)
	require.NotNil(t, res.Found())
	require.NotEqual(t, "ffffffff-ffff-ffff-ffff-ffffffffffff", res.Found().ID.String(),
		"internal stub must be invisible to non-health queries")
	require.Equal(t, "NOT_SERVING", healthStatus(t, res), "non-health query sees the user stub")
}
