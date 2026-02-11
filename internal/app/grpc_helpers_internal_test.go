package app

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/types/known/wrapperspb"

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

func TestGetPeerAddress(t *testing.T) {
	t.Parallel()

	t.Run("nil peer", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "unknown", getPeerAddress(nil))
	})

	t.Run("peer with nil addr", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "unknown", getPeerAddress(&peer.Peer{}))
	})

	t.Run("peer with addr", func(t *testing.T) {
		t.Parallel()

		addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
		require.NoError(t, err)

		p := &peer.Peer{Addr: addr}
		got := getPeerAddress(p)
		require.NotEqual(t, "unknown", got)
		require.Contains(t, got, "127.0.0.1")
	})
}

func TestProtoToJSON(t *testing.T) {
	t.Parallel()

	t.Run("nil message", func(t *testing.T) {
		t.Parallel()

		require.Nil(t, protoToJSON(nil))
	})

	t.Run("non proto message", func(t *testing.T) {
		t.Parallel()

		require.Nil(t, protoToJSON("not a proto"))
	})

	t.Run("valid proto message", func(t *testing.T) {
		t.Parallel()

		msg := wrapperspb.String("hello")
		got := protoToJSON(msg)
		require.NotNil(t, got)
		require.Contains(t, string(got), "hello")
	})
}

func TestToLogArray(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		arr := toLogArray()
		require.NotNil(t, arr)
	})

	t.Run("with values", func(t *testing.T) {
		t.Parallel()

		arr := toLogArray("a", 1, true)
		require.NotNil(t, arr)
	})

	t.Run("skips nil", func(t *testing.T) {
		t.Parallel()

		arr := toLogArray("a", nil, 1)
		require.NotNil(t, arr)
	})
}
