package sdk

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_deriveRestURLFromGrpcAddr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		addr string
		want string
	}{
		{"127.0.0.1:4770", "http://127.0.0.1:4771"},
		{"localhost:4770", "http://localhost:4771"},
		{"[::1]:4770", "http://[::1]:4771"},
		{"[2001:db8::1]:4770", "http://[2001:db8::1]:4771"},
	}
	for _, tt := range tests {
		got := deriveRestURLFromGrpcAddr(tt.addr)
		require.Equal(t, tt.want, got, "addr=%q", tt.addr)
	}
}
