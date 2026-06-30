package proxycapture_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/proxycapture"
)

func TestBuildServerStreamStub(t *testing.T) {
	t.Parallel()

	stub := proxycapture.BuildServerStreamStub(
		"test.Service",
		"ServerStreamMethod",
		"session-srv",
		map[string]any{"request": "data"},
		map[string]any{"req-header": "value"},
		[]any{map[string]any{"stream": 1}, map[string]any{"stream": 2}, map[string]any{"stream": 3}},
		map[string]string{"trailer": "value"},
		nil,
	)

	require.Equal(t, "test.Service", stub.Service)
	require.Equal(t, "ServerStreamMethod", stub.Method)
	require.Len(t, stub.Output.Stream, 3)
}
