package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func TestSplitMethodName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		fullMethod string
		service    string
		method     string
	}{
		{"/pkg.Service/Method", "pkg.Service", "Method"},
		{"/helloworld.Greeter/SayHello", "helloworld.Greeter", "SayHello"},
		{"/grpc.health.v1.Health/Check", "grpc.health.v1.Health", "Check"},
		{"", "unknown", "unknown"},
		{"/only", "unknown", "unknown"},
		{"/a/b/extra", "unknown", "unknown"}, // 4 parts - only /svc/method (2 segments) supported
	}

	for _, tt := range tests {
		t.Run(tt.fullMethod, func(t *testing.T) {
			t.Parallel()

			svc, mth := splitMethodName(tt.fullMethod)
			require.Equal(t, tt.service, svc)
			require.Equal(t, tt.method, mth)
		})
	}
}

func TestIsNilInterface(t *testing.T) {
	t.Parallel()

	var (
		nilSlice []int
		nilMap   map[string]int
	)

	tests := []struct {
		name string
		v    any
		want bool
	}{
		{"nil", nil, true},
		{"nil slice", nilSlice, true},
		{"nil map", nilMap, true},
		{"empty slice", []int{}, false},
		{"empty map", map[string]int{}, false},
		{"string", "x", false},
		{"int", 42, false},
		{"struct", struct{}{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := isNilInterface(tt.v)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestStubNotFoundError(t *testing.T) {
	t.Parallel()

	query := stuber.Query{
		Service: "TestSvc",
		Method:  "TestMethod",
		Input:   []map[string]any{{"id": "1"}},
	}

	err := stubNotFoundError(query, &stuber.Result{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "TestSvc")
	require.Contains(t, err.Error(), "TestMethod")
}
