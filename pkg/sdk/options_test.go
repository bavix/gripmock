package sdk

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestDeriveRestUrlFromGrpcAddr(t *testing.T) {
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

func TestWithFileDescriptor(t *testing.T) {
	t.Parallel()

	o := &options{}
	WithFileDescriptor(wrapperspb.File_google_protobuf_wrappers_proto)(o)

	require.Len(t, o.descriptorFiles, 1)
	require.Equal(t, "google/protobuf/wrappers.proto", o.descriptorFiles[0].GetName())
}
