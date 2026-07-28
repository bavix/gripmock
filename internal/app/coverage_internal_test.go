package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/domain/rest"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func TestMethodCoverage(t *testing.T) {
	t.Parallel()

	services := []rest.Service{
		{
			Id:   "pkg.SvcA",
			Name: "SvcA",
			Methods: []rest.Method{
				{Name: "M1"},
				{Name: "M2"},
			},
		},
		{
			Id:   "pkg.SvcB",
			Name: "SvcB",
			Methods: []rest.Method{
				{Name: "N1"},
			},
		},
	}

	stubs := []*stuber.Stub{
		{Service: "pkg.SvcA", Method: "M1"}, // matches by FQN
		{Service: "SvcB", Method: "N1"},     // matches by bare name (package omitted)
		{Service: "SvcA", Method: "GHOST"},  // method not in schema — ignored
	}

	covered, total := methodCoverage(services, stubs)
	require.Equal(t, 3, total)
	require.Equal(t, 2, covered) // M1 + N1; M2 uncovered
}

func TestMethodCoverageEmpty(t *testing.T) {
	t.Parallel()

	covered, total := methodCoverage(nil, nil)
	require.Zero(t, covered)
	require.Zero(t, total)
}
